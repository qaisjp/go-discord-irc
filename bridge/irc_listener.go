package bridge

import (
	"fmt"
	"strings"

	"github.com/qaisjp/go-ircevent"
)

type ircListener struct {
	*irc.Connection
	bridge *Bridge
}

func NewIRCListener(dib *Bridge, webIRCPass string) *ircListener {
	irccon := irc.IRC(dib.Config.IRCListenerName, "discord")
	irc := &ircListener{irccon, dib}

	dib.SetupIRCConnection(irccon, "discord.", "fd75:f5f5:226f::")
	if dib.Config.Debug {
		irccon.VerboseCallbackHandler = true
		irccon.Debug = true
	}

	// Welcome event
	irccon.AddCallback("001", irc.OnWelcome)

	// Called when received channel names... essentially OnJoinChannel
	irccon.AddCallback("366", irc.OnJoinChannel)
	irccon.AddCallback("PRIVMSG", irc.OnPrivateMessage)
	irccon.AddCallback("CTCP_ACTION", irc.OnPrivateMessage)

	return irc
}

func (i *ircListener) OnWelcome(e *irc.Event) {
	// Join all channels
	i.SendRaw("JOIN " + strings.Join(i.bridge.GetIRCChannels(), ","))
}

func (i *ircListener) OnJoinChannel(e *irc.Event) {
	fmt.Printf("Listener has joined IRC channel %s.\n", e.Arguments[1])
}

func (i *ircListener) OnPrivateMessage(e *irc.Event) {
	// Ignore private messages
	if string(e.Arguments[0][0]) != "#" {
		i.Privmsg(e.Nick, "Private messaging Discord users is not supported.")
		return
	}

	// Ignore messages from Discord bots
	if strings.HasSuffix(e.Nick, i.bridge.Config.Suffix) {
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
