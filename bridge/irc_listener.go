package bridge

import (
	"strings"

	ircf "github.com/qaisjp/go-discord-irc/irc/format"
	irc "github.com/qaisjp/go-ircevent"
	log "github.com/sirupsen/logrus"
)

type ircListener struct {
	*irc.Connection
	bridge *Bridge

	joinQuitCallbacks map[string]int
}

func newIRCListener(dib *Bridge, webIRCPass string) *ircListener {
	irccon := irc.IRC(dib.Config.IRCListenerName, "discord")
	listener := &ircListener{irccon, dib, nil}

	dib.SetupIRCConnection(irccon, "discord.", "fd75:f5f5:226f::")
	listener.SetDebugMode(dib.Config.Debug)

	// Nick tracker for nick tracking
	irccon.SetupNickTrack()

	// Welcome event
	irccon.AddCallback("001", listener.OnWelcome)

	// Called when received channel names... essentially OnJoinChannel
	irccon.AddCallback("366", listener.OnJoinChannel)
	irccon.AddCallback("PRIVMSG", listener.OnPrivateMessage)
	irccon.AddCallback("NOTICE", listener.OnPrivateMessage)
	irccon.AddCallback("CTCP_ACTION", listener.OnPrivateMessage)

	irccon.AddCallback("900", func(e *irc.Event) {
		// Try to rejoni channels after authenticated with NickServ
		listener.JoinChannels()
	})

	// Note that this might override SetupNickTrack!
	listener.OnJoinQuitSettingChange()

	return listener
}

func (i *ircListener) nickTrackNick(event *irc.Event) {
	oldNick := event.Nick
	newNick := event.Message()
	if con, ok := i.bridge.ircManager.puppetNicks[oldNick]; ok {
		i.bridge.ircManager.puppetNicks[newNick] = con
		delete(i.bridge.ircManager.puppetNicks, oldNick)
	}
}

// From irc_nicktrack.go.
func (i *ircListener) nickTrackQuit(e *irc.Event) {
	for k := range i.Connection.Channels {
		delete(i.Channels[k].Users, e.Nick)
	}
}

func (i *ircListener) OnJoinQuitSettingChange() {
	// Clear Nicktrack QUIT callback as it races with this
	i.ClearCallback("QUIT")
	i.ClearCallback("NICK")
	i.AddCallback("NICK", i.nickTrackNick)

	// If remove callbacks...
	if !i.bridge.Config.ShowJoinQuit {
		for event, id := range i.joinQuitCallbacks {
			i.RemoveCallback(event, id) // note that QUIT was already removed above
		}

		// Add back Nicktrack QUIT since it was removed
		i.AddCallback("QUIT", i.nickTrackQuit)
		return
	}

	callbacks := []string{"JOIN", "PART", "QUIT", "KICK"}
	cbs := make(map[string]int, len(callbacks))
	for _, cb := range callbacks {
		i.AddCallback(cb, i.OnJoinQuitCallback)
	}

	i.joinQuitCallbacks = cbs
}

func (i *ircListener) OnJoinQuitCallback(event *irc.Event) {
	// This checks if the source of the event was from a puppet.
	// It won't work correctly for KICK, as the source is always the person that
	// performed the kick. But that's okay because puppets aren't supposed to be kicked.
	if i.isPuppetNick(event.Nick) {
		return
	}

	message := ""
	content := ""
	target := ""
	manager := i.bridge.ircManager

	switch event.Code {
	case "JOIN":
		message = manager.formatDiscordMessage(event.Code, event, "", "")
	case "PART":
		if len(event.Arguments) > 1 {
			content = event.Arguments[1]
		}
		message = manager.formatDiscordMessage(event.Code, event, content, "")
	case "QUIT":
		content := event.Nick
		if len(event.Arguments) == 1 {
			content = event.Arguments[0]
		}
		message = manager.formatDiscordMessage(event.Code, event, content, "")
	case "KICK":
		target, content = event.Arguments[1], event.Arguments[2]
		message = manager.formatDiscordMessage(event.Code, event, content, target)
	}

	// if the message is empty...
	if message == "" {
		return // do nothing, Discord doesn't like empty messages anyway
	}

	msg := IRCMessage{
		// IRCChannel: set on the fly
		Username: "",
		Message:  message,
	}

	if event.Code == "QUIT" {
		// Notify channels that the user is in
		for _, m := range i.bridge.mappings {
			channel := m.IRCChannel
			channelObj, ok := i.Connection.Channels[channel]
			if !ok {
				log.WithField("channel", channel).WithField("who", event.Nick).Warnln("Trying to process QUIT. Channel not found in irc listener cache.")
				continue
			}
			if _, ok := channelObj.Users[event.Nick]; !ok {
				continue
			}
			msg.IRCChannel = channel
			i.bridge.discordMessagesChan <- msg
		}

		// Call nicktrack QUIT now (this avoids a race)
		i.nickTrackQuit(event)
	} else {
		msg.IRCChannel = event.Arguments[0]
		i.bridge.discordMessagesChan <- msg
	}
}

func (i *ircListener) DoesUserExist(user string) bool {
	i.Lock()
	defer i.Unlock()
	for _, channel := range i.Channels {
		_, ok := channel.Users[user]
		if ok {
			return true
		}
	}

	return false
}

func (i *ircListener) SetDebugMode(debug bool) {
	// i.VerboseCallbackHandler = debug
	// i.Debug = debug
}

func (i *ircListener) OnWelcome(e *irc.Event) {
	identify := i.bridge.Config.NickServIdentify
	// identify as listener
	if identify != "" {
		i.Privmsgf("nickserv", "identify %s", identify)
	}

	// Join all channels
	i.JoinChannels()
}

func (i *ircListener) JoinChannels() {
	i.SendRaw(i.bridge.GetJoinCommand(i.bridge.mappings))
}

func (i *ircListener) OnJoinChannel(e *irc.Event) {
	log.Infof("Listener has joined IRC channel %s.", e.Arguments[1])
}

func (i *ircListener) isPuppetNick(nick string) bool {
	if i.GetNick() == nick {
		return true
	}
	if _, ok := i.bridge.ircManager.puppetNicks[nick]; ok {
		return true
	}
	return false
}

func (i *ircListener) OnPrivateMessage(e *irc.Event) {
	// Ignore private messages
	if string(e.Arguments[0][0]) != "#" {
		// If you decide to extend this to respond to PMs, make sure
		// you do not respond to NOTICEs, see issue #50.
		return
	}

	// Discord doesn't accept an empty message
	if strings.TrimSpace(e.Message()) == "" {
		return
	}

	// Ignore messages from Discord bots
	if i.isPuppetNick(e.Nick) {
		return
	}

	replacements := []string{}
	for _, con := range i.bridge.ircManager.ircConnections {
		replacements = append(replacements, con.nick, "<@!"+con.discord.ID+">")
	}

	msg := strings.NewReplacer(
		replacements...,
	).Replace(e.Message())

	if e.Code == "CTCP_ACTION" {
		msg = "_" + msg + "_"
	}

	msg = ircf.BlocksToMarkdown(ircf.Parse(ircf.StripColor(msg)))

	go func(e *irc.Event) {
		i.bridge.discordMessagesChan <- IRCMessage{
			IRCChannel: e.Arguments[0],
			Username:   e.Nick,
			Message:    msg,
		}
	}(e)
}
