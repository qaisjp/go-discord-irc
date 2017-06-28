package bridge

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/qaisjp/go-discord-irc/ircnick"
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

func (m *ircManager) CloseConnection(i *ircConnection) {
	delete(m.ircConnections, i.userID)
	close(i.messages)
	i.innerCon.Quit()
	// i.innerCon.Disconnect()

}

func (m *ircManager) Close() {
	i := 0
	for _, con := range m.ircConnections {
		con.innerCon.QuitMessage = "[Bridge going down] "
		if i < len(quitMessages) {
			con.innerCon.QuitMessage += quitMessages[i]
		}

		m.CloseConnection(con)

		i++
	}
}

func (m *ircManager) HandleUser(user DiscordUser) {
	if con, ok := m.ircConnections[user.ID]; ok {
		// Close the connection if they are not online
		if !user.Online {
			m.CloseConnection(con)
			return
		}

		// Otherwise update their nickname / username
		// TODO: Support username changes
		// Note: this event is still called when their status is changed
		//       from `online` to `dnd` (online related states)
		//       In UpdateDetails we handle nickname changes so it is
		//       OK to call the below potentially redundant function
		con.UpdateDetails(user.Discriminator, user.Nick)
		return
	}

	// Don't create a connection if they are not online
	if !user.Online {
		return
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
		innerCon: innerCon,

		userID:        user.ID,
		discriminator: user.Discriminator,
		nick:          user.Nick,

		messages: make(chan DiscordNewMessage),

		manager: m,
	}

	con.innerCon.AddCallback("001", con.OnWelcome)

	m.ircConnections[user.ID] = con

	err := con.innerCon.Connect(m.ircServerAddress)
	if err != nil {
		fmt.Println("error opening irc connection,", err)
		// TODO: HANDLE THIS SITUATION
		return
	}

	go innerCon.Loop()

	return
}

// TODO: Catch username changes, and cache UserID:Username mappings somewhere
func (m *ircManager) generateNickname(_ string, nick string) string {
	// First clean it
	nick = ircnick.NickClean(nick)

	return nick + "~d"
	// return fmt.Sprintf("[%s-%s]", username, discriminator), nil
}

func (m *ircManager) SendMessage(userID, channel, message string) {
	con, ok := m.ircConnections[userID]

	// Person is likely appearing offline... :/
	if !ok {
		// Should not be doing m.h..discord....
		member, err := m.h.discord.State.Member(m.h.discord.guildID, userID)

		if err != nil {
			panic(errors.Wrap(err, "Could not find connection and member!"))
		}

		// Should not be doing m.h.ircListener....
		m.h.ircListener.Privmsg(channel, fmt.Sprintf("<%s#%s> %s", member.User.Username, member.User.Discriminator, message))

		return
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
