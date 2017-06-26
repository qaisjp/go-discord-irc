package bridge

import (
	"fmt"
	"time"

	irc "github.com/thoj/go-ircevent"
)

// Should only be used from one thread.
type ircManager struct {
	ircConnections   map[string]*ircConnection
	ircServerAddress string
	webIRCPass       string

	h *home
}

func prepareIRCManager(ircServerAddress, webIRCPass string) *ircManager {
	return &ircManager{
		ircConnections:   make(map[string]*ircConnection),
		ircServerAddress: ircServerAddress,
		webIRCPass:       webIRCPass,
	}
}

func (m *ircManager) DisconnectAll() {
	for key, con := range m.ircConnections {
		close(con.messages)
		con.Close()

		m.ircConnections[key] = nil
	}
}

func (m *ircManager) CreateConnection(user DiscordUser) (*ircConnection, error) {
	if con, ok := m.ircConnections[user.ID]; ok {
		con.UpdateDetails(user.Discriminator, user.Nick)
		return con, nil
	}

	nick := m.generateNickname(user.Discriminator, user.Nick)

	innerCon := irc.IRC(nick, "discord")
	// innerCon.Debug = true

	var ip string
	{
		baseip := "fd75:f5f5:226f:"
		if user.Bot {
			baseip += "2"
		} else {
			baseip += "1"
		}
		ip = SnowflakeToIP(baseip, user.ID)
	}

	hostname := user.ID
	if user.Bot {
		hostname += ".bot.discord"
	} else {
		hostname += ".user.discord"
	}

	setupIRCConnection(innerCon, m.webIRCPass, hostname, ip)

	con := &ircConnection{
		Connection: innerCon,

		messages: make(chan DiscordNewMessage),

		manager: m,
	}

	con.AddCallback("001", con.OnWelcome)

	m.ircConnections[user.ID] = con

	err := con.Connect(m.ircServerAddress)
	if err != nil {
		fmt.Println("error opening irc connection,", err)
		return nil, err
	}

	go innerCon.Loop()

	return con, nil
}

// TODO: Catch username changes, and cache UserID:Username mappings somewhere
func (m *ircManager) generateNickname(_ string, nick string) string {
	return nick + "~d"
	// return fmt.Sprintf("[%s-%s]", username, discriminator), nil
}

func (m *ircManager) SendMessage(userID, channel, message string) {
	con, ok := m.ircConnections[userID]
	if !ok {
		panic("Could not find connection")
	}

	msg := DiscordNewMessage{
		ircChannel: channel,
		str:        message,
	}

	select {
	// Try to send the message immediately
	case con.messages <- msg:
	// If it can't after 5ms, do it in a separate goroutine
	case <-time.After(time.Millisecond * 5):
		go func() {
			con.messages <- msg
		}()
	}
}

// TODO
// Find all the Discord channels this user belongs to,
// and then find pairings in the global pairings list
// Currently just returns all participating IRC channels
func (m *ircManager) RequestChannels(userID string) []string {
	return m.h.GetIRCChannels()
}
