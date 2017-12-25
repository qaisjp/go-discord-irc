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

func NewIRCListener(dib *Bridge, webIRCPass string) *ircListener {
	irccon := irc.IRC(dib.Config.IRCListenerName, "discord")
	irc := &ircListener{irccon, dib}

	dib.SetupIRCConnection(irccon, "discord.", "fd75:f5f5:226f::")
	irc.SetDebugMode(dib.Config.Debug)

	// Welcome event
	irccon.AddCallback("001", irc.OnWelcome)

	// Called when received channel names... essentially OnJoinChannel
	irccon.AddCallback("366", irc.OnJoinChannel)
	irccon.AddCallback("PRIVMSG", irc.OnPrivateMessage)
	irccon.AddCallback("CTCP_ACTION", irc.OnPrivateMessage)

	return irc
}

func (i *ircListener) SetDebugMode(debug bool) {
	i.VerboseCallbackHandler = debug
	i.Debug = debug
}

func (i *ircListener) OnWelcome(e *irc.Event) {
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
			i.Privmsg(e.Nick, "help, who")
		} else if e.Message() == "who" {
			i.Privmsg(e.Nick, "I am the bot listener.")
		} else {
			i.Privmsg(e.Nick, "Private messaging Discord users is not supported, but I support commands! Type 'help'.")
		}
		return
	}

	// Ignore messages from Discord bots
	if strings.HasSuffix(strings.TrimRight(e.Nick, "_"), i.bridge.Config.Suffix) {
		return
	}

	msg := e.Message()
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
