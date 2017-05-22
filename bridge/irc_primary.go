package bridge

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/thoj/go-ircevent"
)

func prepareIRC(dib *Bridge) {
	irccon := irc.IRC(dib.opts.IRCPrimaryName, "BetterDiscordBot")
	dib.ircPrimary = irccon

	// irccon.VerboseCallbackHandler = true
	irccon.Debug = true
	irccon.UseTLS = true
	irccon.TLSConfig = &tls.Config{InsecureSkipVerify: true} // TODO: Insecure TLS!

	// Welcome event
	irccon.AddCallback("001", func(e *irc.Event) {
		// Join all channels
		e.Connection.SendRaw("JOIN " + strings.Join(dib.chanIRC, ","))
	})

	// Called when received channel names... essentially OnJoinChannel
	irccon.AddCallback("366", func(e *irc.Event) { fmt.Printf("Joined IRC channel %s.", e.Arguments[1]) })

	irccon.AddCallback("PRIVMSG", func(event *irc.Event) {
		go func(_ *irc.Event) {
			//event.Message() contains the message
			//event.Nick Contains the sender
			//event.Arguments[0] Contains the channel
		}(event)
	})
}
