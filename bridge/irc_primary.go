package bridge

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/thoj/go-ircevent"
)

type ircPrimary struct {
	*irc.Connection
	h *home
}

func prepareIRC(dib *Bridge) *ircPrimary {
	irccon := irc.IRC(dib.ircPrimaryName, "BetterDiscordBot")
	irc := &ircPrimary{irccon, nil}

	// irccon.VerboseCallbackHandler = true
	irccon.Debug = true
	irccon.UseTLS = true
	irccon.TLSConfig = &tls.Config{InsecureSkipVerify: true} // TODO: Insecure TLS!

	// Welcome event
	irccon.AddCallback("001", irc.OnWelcome)

	// Called when received channel names... essentially OnJoinChannel
	irccon.AddCallback("366", irc.OnJoinChannel)
	irccon.AddCallback("PRIVMSG", irc.OnPrivateMessage)

	return irc
}

func (i *ircPrimary) OnWelcome(e *irc.Event) {
	// Join all channels
	e.Connection.SendRaw("JOIN " + strings.Join(i.h.GetIRCChannels(), ","))
}

func (i *ircPrimary) OnJoinChannel(e *irc.Event) {
	fmt.Printf("Joined IRC channel %s.\n", e.Arguments[1])
}

func (i *ircPrimary) OnPrivateMessage(e *irc.Event) {
	go func(e *irc.Event) {
		i.h.SendDiscordMessage(DiscordNewMessage{
			ircChannel: e.Arguments[0],
			str:        fmt.Sprintf("<%s> %s", e.Nick, e.Message()),
		})
	}(e)
}
