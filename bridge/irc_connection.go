package bridge

import (
	"strings"

	irc "github.com/thoj/go-ircevent"
)

// An ircConnection should only ever communicate with its manager
// Refer to `(m *ircManager) CreateConnection` to see how these are spawned
type ircConnection struct {
	*irc.Connection

	userID        string
	discriminator string
	username      string

	messages chan DiscordNewMessage

	manager *ircManager
}

func (i *ircConnection) Close() {
	i.Quit()
	i.Disconnect()
}

func (i *ircConnection) OnWelcome(e *irc.Event) {
	i.JoinChannels()

	go func(i *ircConnection) {
		for m := range i.messages {
			i.Privmsg(m.ircChannel, m.str)
		}
	}(i)
}

func (i *ircConnection) JoinChannels() {
	channels := i.manager.RequestChannels(i.userID)
	i.SendRaw("JOIN " + strings.Join(channels, ","))
}

func (i *ircConnection) UpdateDetails(discriminator string, nickname string) {
	username := i.manager.generateNickname(discriminator, nickname)

	i.discriminator = discriminator
	i.username = username

	if i.Connected() {
		i.SendRaw("NICK " + username)
	}
}
