package bridge

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/thoj/go-ircevent"
)

type ircPrimary struct {
	h *home
}

func prepareIRC(dib *Bridge) {
	irccon := irc.IRC(dib.opts.IRCPrimaryName, "BetterDiscordBot")
	dib.ircPrimary = irccon

	irc := &ircPrimary{h: dib.h}

	// irccon.VerboseCallbackHandler = true
	irccon.Debug = true
	irccon.UseTLS = true
	irccon.TLSConfig = &tls.Config{InsecureSkipVerify: true} // TODO: Insecure TLS!

	// Welcome event
	irccon.AddCallback("001", irc.OnWelcome)

	// Called when received channel names... essentially OnJoinChannel
	irccon.AddCallback("366", irc.OnJoinChannel)
	irccon.AddCallback("PRIVMSG", irc.OnPrivateMessage)
}

func (i *ircPrimary) OnWelcome(e *irc.Event) {
	// Join all channels
	e.Connection.SendRaw("JOIN " + strings.Join(i.h.GetIRCChannels(), ","))
}

func (i *ircPrimary) OnJoinChannel(e *irc.Event) {
	fmt.Printf("Joined IRC channel %s.", e.Arguments[1])
}

func (i *ircPrimary) OnPrivateMessage(e *irc.Event) {
	go func(e *irc.Event) {
		//event.Message() contains the message
		//event.Nick Contains the sender
		//event.Arguments[0] Contains the channel
		i.h.SendDiscordMessage(DiscordMessage{
			ircChannel: e.Arguments[0],
			str:        fmt.Sprintf("<%s> %s", e.Nick, e.Message()),
		})
	}(e)
}
