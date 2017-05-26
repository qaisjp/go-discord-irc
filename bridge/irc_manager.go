package bridge

import (
	"fmt"
	"strings"

	"github.com/thoj/go-ircevent"
)

type ircConnection struct {
	*irc.Connection

	userID   string
	username string

	messages chan DiscordNewMessage

	manager *ircManager
}

// Should only be used from one thread.
type ircManager struct {
	ircConnections   map[string]*ircConnection
	ircServerAddress string

	h *home
}

func prepareIRCManager(ircServerAddress string) *ircManager {
	return &ircManager{
		ircConnections:   make(map[string]*ircConnection),
		ircServerAddress: ircServerAddress,
	}
}

func (m *ircManager) DisconnectAll() {
	for key, con := range m.ircConnections {
		close(con.messages)
		con.Close()

		m.ircConnections[key] = nil
	}
}

func (m *ircManager) CreateConnection(userID string) (*ircConnection, error) {
	if con, ok := m.ircConnections[userID]; ok {
		fmt.Println("Returning cached IRC connection")

		return con, nil
	}

	username, err := m.generateUsername(userID)
	if err != nil {
		return nil, err
	}

	innerCon := irc.IRC(username, "BetterDiscordBot")
	setupIRCConnection(innerCon)

	con := &ircConnection{
		Connection: innerCon,

		messages: make(chan DiscordNewMessage),

		manager: m,
	}

	con.AddCallback("001", con.OnWelcome)

	m.ircConnections[userID] = con

	err = con.Connect(m.ircServerAddress)
	if err != nil {
		fmt.Println("error opening irc connection,", err)
		return nil, err
	}

	go innerCon.Loop()

	return con, nil
}

// TODO: Catch username changes, and cache UserID:Username mappings somewhere
func (m *ircManager) generateUsername(userID string) (string, error) {
	discriminator, username, err := m.h.GetDiscordUserInfo(userID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("[%s-%s]", username, discriminator), nil
}

func (m *ircManager) PulseID(userID string) {
	_, err := m.CreateConnection(userID)

	if err != nil {
		panic(err)
		return
	}
}

func (i *ircManager) SendMessage(userID, channel, message string) {
	con, err := i.CreateConnection(userID)
	if err != nil {
		panic(err)
	}

	con.messages <- DiscordNewMessage{
		ircChannel: channel,
		str:        message,
	}
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
