package bridge

import (
	"strings"

	irc "github.com/thoj/go-ircevent"
)

func (i *ircConnection) Close() {

}

func (i *ircConnection) OnWelcome(e *irc.Event) {
	// Join all channels
	e.Connection.SendRaw("JOIN " + strings.Join(i.manager.h.GetIRCChannels(), ","))

	go func(i *ircConnection) {
		for m := range i.messages {
			i.Privmsg(m.ircChannel, m.str)
		}
	}(i)
}

func (i *ircConnection) RefreshUsername() (err error) {
	username, err := i.manager.generateUsername(i.userID)

	if err != nil {
		return
	}

	i.username = username

	if i.Connected() {
		i.SendRaw("NICK " + username)
	}

	return
}
