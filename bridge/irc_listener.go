package bridge

import (
	"strings"

	"github.com/qaisjp/go-ircevent"
	log "github.com/sirupsen/logrus"
)

type ircListener struct {
	*irc.Connection
	bridge *Bridge
}

func newIRCListener(dib *Bridge, webIRCPass string) *ircListener {
	irccon := irc.IRC(dib.Config.IRCListenerName, "discord")
	listener := &ircListener{irccon, dib}

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

	return listener
}

func (i *ircListener) DoesUserExist(user string) bool {
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
	i.SendRaw("JOIN " + strings.Join(i.bridge.GetIRCChannels(), ","))
}

func (i *ircListener) OnJoinChannel(e *irc.Event) {
	log.Infof("Listener has joined IRC channel %s.", e.Arguments[1])
}

func (i *ircListener) OnPrivateMessage(e *irc.Event) {
	// Ignore private messages
	if string(e.Arguments[0][0]) != "#" {
		if e.Message() == "help" {
			i.Privmsg(e.Nick, "Commands: help, who")
		} else if e.Message() == "who" {
			i.Privmsg(e.Nick, "I am the bot listener.")
		} else {
			i.Privmsg(e.Nick, "Private messaging Discord users is not supported, but I support commands! Type 'help'.")
		}
		return
	}

	// Discord doesn't accept an empty message
	if strings.TrimSpace(e.Message()) == "" {
		return
	}

	// Ignore messages from Discord bots
	if strings.HasSuffix(strings.TrimRight(e.Nick, "_"), i.bridge.Config.Suffix) {
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

	go func(e *irc.Event) {
		i.bridge.discordMessagesChan <- IRCMessage{
			IRCChannel: e.Arguments[0],
			Username:   e.Nick,
			Message:    msg,
		}
	}(e)
}
