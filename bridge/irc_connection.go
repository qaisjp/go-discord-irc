package bridge

import (
	"log"
	"strings"

	irc "github.com/qaisjp/go-ircevent"
)

// An ircConnection should only ever communicate with its manager
// Refer to `(m *ircManager) CreateConnection` to see how these are spawned
type ircConnection struct {
	innerCon *irc.Connection

	userID        string
	discriminator string
	originalNick  string
	nick          string

	messages chan IRCMessage

	manager *IRCManager
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
	if (i.originalNick == nick) && (i.discriminator == discriminator) {
		return
	}

	i.originalNick = nick

	nick = i.manager.generateNickname(discriminator, nick)

	i.discriminator = discriminator
	i.nick = nick

	go i.innerCon.Nick(nick)
}

func (i *ircConnection) OnPrivateMessage(e *irc.Event) {
	// Alert private messages
	if string(e.Arguments[0][0]) != "#" {
		if e.Message() == "help" {
			i.innerCon.Privmsg(e.Nick, "help, who")
		} else if e.Message() == "who" {
			i.innerCon.Privmsgf(e.Nick, "I am: %s#%s with ID %s", i.originalNick, i.discriminator, i.userID)
		} else {
			i.innerCon.Privmsg(e.Nick, "Private messaging Discord users is not supported, but I support commands! Type 'help'.")
		}
		return
	}

	log.Println("Non listener IRC connection received PRIVMSG from channel. Something went wrong.")
}
