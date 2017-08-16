package bridge

import (
	"strings"

	irc "github.com/thoj/go-ircevent"
)

// An ircConnection should only ever communicate with its manager
// Refer to `(m *ircManager) CreateConnection` to see how these are spawned
type ircConnection struct {
	innerCon *irc.Connection

	userID        string
	discriminator string
	nick          string

	messages chan IRCMessage

	manager *ircManager
}

func (i *ircConnection) OnWelcome(e *irc.Event) {
	i.JoinChannels()
	i.innerCon.SendRawf("MODE %s +D", i.innerCon.GetNick())

	go func(i *ircConnection) {
		for m := range i.messages {
			if m.IsAction {
				i.innerCon.Action(m.IRCChannel, m.Message)
			} else {
				i.innerCon.Privmsg(m.IRCChannel, m.Message)
			}
		}
	}(i)
}

func (i *ircConnection) JoinChannels() {
	channels := i.manager.RequestChannels(i.userID)
	i.innerCon.SendRaw("JOIN " + strings.Join(channels, ","))
}

func (i *ircConnection) UpdateDetails(discriminator, nick string) {
	// if their details haven't changed, don't do anything
	if (i.nick == nick) && (i.discriminator == discriminator) {
		return
	}

	nick = i.manager.generateNickname(discriminator, nick)

	i.discriminator = discriminator
	i.nick = nick

	go i.innerCon.Nick(nick)
}
