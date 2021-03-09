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

	listenerCallbackIDs map[string]int
}

func newIRCListener(dib *Bridge, webIRCPass string) *ircListener {
	irccon := irc.IRC(dib.Config.IRCListenerName, "discord")
	listener := &ircListener{irccon, dib, make(map[string]int)}

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

	// we are assuming this will be posible to run independent of any
	// future NICK callbacks added, otherwise do it like the STQUIT callback
	listener.AddCallback("NICK", listener.nickTrackNick)

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

func (i *ircListener) OnNickRelayToDiscord(event *irc.Event) {
	// ignored hostmasks, or we're a puppet? no relay
	if i.bridge.ircManager.isIgnoredHostmask(event.Source) ||
		i.isPuppetNick(event.Nick) ||
		i.isPuppetNick(event.Message()) {
		return
	}

	newNick := event.Message()
	message := i.bridge.ircManager.formatDiscordMessage("NICK", event, newNick, "")

	// if the message is empty...
	if message == "" {
		return // do nothing, Discord doesn't like empty messages anyway
	}

	msg := IRCMessage{
		Username: "",
		Message:  message,
	}

	for _, m := range i.bridge.mappings {
		channel := m.IRCChannel
		if channelObj, ok := i.Connection.GetChannel(channel); ok {
			if _, ok := channelObj.GetUser(newNick); ok {
				msg.IRCChannel = channel
				i.bridge.discordMessagesChan <- msg
			}
		}
	}
}

func (i *ircListener) nickTrackPuppetQuit(e *irc.Event) {
	// Protect against HostServ changing nicks or ircd's with CHGHOST/CHGIDENT or similar
	// sending us a QUIT for a puppet nick only for it to rejoin right after.
	// The puppet nick won't see a true disconnection itself and thus will still see itself
	// as connected.
	if con, ok := i.bridge.ircManager.puppetNicks[e.Nick]; ok && !con.Connected() {
		delete(i.bridge.ircManager.puppetNicks, e.Nick)
	}
}

func (i *ircListener) OnJoinQuitSettingChange() {
	// always remove our listener callbacks
	for ev, id := range i.listenerCallbackIDs {
		i.RemoveCallback(ev, id)
		delete(i.listenerCallbackIDs, ev)
	}

	// we're either going to track quits, or track and relay said, so swap out the callback
	// based on which is in effect.
	if i.bridge.Config.ShowJoinQuit {
		i.listenerCallbackIDs["STNICK"] = i.AddCallback("STNICK", i.OnNickRelayToDiscord)

		// KICK is not state tracked!
		callbacks := []string{"STJOIN", "STPART", "STQUIT", "KICK"}
		for _, cb := range callbacks {
			id := i.AddCallback(cb, i.OnJoinQuitCallback)
			i.listenerCallbackIDs[cb] = id
		}
	} else {
		id := i.AddCallback("STQUIT", i.nickTrackPuppetQuit)
		i.listenerCallbackIDs["STQUIT"] = id
	}
}

func (i *ircListener) OnJoinQuitCallback(event *irc.Event) {
	// This checks if the source of the event was from a puppet.
	if (event.Code == "KICK" && i.isPuppetNick(event.Arguments[1])) || i.isPuppetNick(event.Nick) {
		// since we replace the STQUIT callback we have to manage our puppet nicks when
		// this call back is active!
		if event.Code == "STQUIT" {
			i.nickTrackPuppetQuit(event)
		}
		return
	}

	// Ignored hostmasks
	if i.bridge.ircManager.isIgnoredHostmask(event.Source) {
		return
	}

	message := ""
	content := ""
	target := ""
	manager := i.bridge.ircManager

	switch event.Code {
	case "STJOIN":
		message = manager.formatDiscordMessage("JOIN", event, "", "")
	case "STPART":
		if len(event.Arguments) > 1 {
			content = event.Arguments[1]
		}
		message = manager.formatDiscordMessage("PART", event, content, "")
	case "STQUIT":
		content := event.Nick
		if len(event.Arguments) == 1 {
			content = event.Arguments[0]
		}
		message = manager.formatDiscordMessage("QUIT", event, content, "")
	case "KICK":
		target, content = event.Arguments[1], event.Arguments[2]
		message = manager.formatDiscordMessage("KICK", event, content, target)
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

	if event.Code == "STQUIT" {
		// Notify channels that the user is in
		for _, m := range i.bridge.mappings {
			channel := m.IRCChannel
			channelObj, ok := i.Connection.GetChannel(channel)
			if !ok {
				log.WithField("channel", channel).WithField("who", event.Nick).Warnln("Trying to process QUIT. Channel not found in irc listener cache.")
				continue
			}
			if _, ok := channelObj.GetUser(event.Nick); !ok {
				continue
			}
			msg.IRCChannel = channel
			i.bridge.discordMessagesChan <- msg
		}
	} else {
		msg.IRCChannel = event.Arguments[0]
		i.bridge.discordMessagesChan <- msg
	}
}

// FIXME: the user might not be on any channel that we're in and that would
// lead to incorrect assumptions the user doesn't exist!
// Good way to check is to utilize ISON
func (i *ircListener) DoesUserExist(user string) bool {
	ret := false
	i.IterChannels(func(name string, ch *irc.Channel) {
		if !ret {
			_, ret = ch.GetUser(user)
		}
	})
	return ret
}

func (i *ircListener) SetDebugMode(debug bool) {
	// i.VerboseCallbackHandler = debug
	// i.Debug = debug
}

func (i *ircListener) OnWelcome(e *irc.Event) {
	// Execute prejoin commands
	for _, com := range i.bridge.Config.IRCListenerPrejoinCommands {
		i.SendRaw(strings.ReplaceAll(com, "${NICK}", i.GetNick()))
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

	if strings.TrimSpace(e.Message()) == "" || // Discord doesn't accept an empty message
		i.isPuppetNick(e.Nick) || // ignore msg's from our puppets
		i.bridge.ircManager.isIgnoredHostmask(e.Source) || //ignored hostmasks
		i.bridge.ircManager.isFilteredIRCMessage(e.Message()) { // filtered
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
