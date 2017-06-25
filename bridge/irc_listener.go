package bridge

import (
	"fmt"
	"strings"

	"github.com/thoj/go-ircevent"
)

type ircListener struct {
	*irc.Connection
	h *home
}

func prepareIRCListener(dib *Bridge, webIRCPass string) *ircListener {
	irccon := irc.IRC(dib.ircPrimaryName, "github.com/qaisjp/go-discord-irc")
	irc := &ircListener{irccon, nil}

	setupIRCConnection(irccon, webIRCPass, "discord.", "fd75:f5f5:226f::")
	// irccon.VerboseCallbackHandler = true
	// irccon.Debug = true

	// Welcome event
	irccon.AddCallback("001", irc.OnWelcome)

	// Called when received channel names... essentially OnJoinChannel
	irccon.AddCallback("366", irc.OnJoinChannel)
	irccon.AddCallback("PRIVMSG", irc.OnPrivateMessage)

	return irc
}

func (i *ircListener) OnWelcome(e *irc.Event) {
	// Join all channels
	e.Connection.SendRaw("JOIN " + strings.Join(i.h.GetIRCChannels(), ","))
}

func (i *ircListener) OnJoinChannel(e *irc.Event) {
	fmt.Printf("Joined IRC channel %s.\n", e.Arguments[1])
}

func (i *ircListener) OnPrivateMessage(e *irc.Event) {
	// Ignore private messages
	if string(e.Arguments[0][0]) != "#" {
		return
	}

	go func(e *irc.Event) {
		i.h.discordMessagesChan <- DiscordNewMessage{
			ircChannel: e.Arguments[0],
			str:        fmt.Sprintf("<%s> %s", e.Nick, e.Message()),
		}
	}(e)
}
