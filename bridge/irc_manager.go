package bridge

import (
	"fmt"
	"regexp"
	"time"

	"github.com/qaisjp/go-discord-irc/ircnick"
	irc "github.com/qaisjp/go-ircevent"
	log "github.com/sirupsen/logrus"
)

var cooldownDuration = time.Minute * 10

// IRCManager should only be used from one thread.
type IRCManager struct {
	ircConnections map[string]*ircConnection

	bridge *Bridge
}

// NewIRCManager creates a new IRCManager
func NewIRCManager(bridge *Bridge) *IRCManager {
	return &IRCManager{
		ircConnections: make(map[string]*ircConnection),
		bridge:         bridge,
	}
}

func (m *IRCManager) CloseConnection(i *ircConnection) {
	// Destroy the cooldown timer
	if i.cooldownTimer != nil {
		i.cooldownTimer.Stop()
		i.cooldownTimer = nil
	}

	delete(m.ircConnections, i.discord.ID)
	close(i.messages)
	i.innerCon.Quit()
}

func (m *IRCManager) Close() {
	i := 0
	for _, con := range m.ircConnections {
		m.CloseConnection(con)
		i++
	}
}

func (m *IRCManager) SetConnectionCooldown(con *ircConnection) {
	if con.cooldownTimer != nil {
		con.cooldownTimer.Stop()
	}

	con.cooldownTimer = time.AfterFunc(
		cooldownDuration,
		func() {
			m.CloseConnection(con)
		},
	)
}

func (m *IRCManager) HandleUser(user DiscordUser) {
	// Does the user exist on the IRC side?
	if con, ok := m.ircConnections[user.ID]; ok {
		// Close the connection if they are not
		// online on Discord anymore (after cooldown)
		if !user.Online {
			m.SetConnectionCooldown(con)
		}

		// If user.Nick is empty then we probably just had a status change
		if user.Nick == "" {
			return
		}

		// Update their nickname / username
		// TODO: Support username changes
		// Note: this event is still called when their status is changed
		//       from `online` to `dnd` (online related states)
		//       In UpdateDetails we handle nickname changes so it is
		//       OK to call the below potentially redundant function
		con.UpdateDetails(user)
		return
	}

	// If they are not online, do not create a connection.
	if !user.Online {
		return
	}

	nick := m.generateNickname(user)

	innerCon := irc.IRC(nick, "discord")
	innerCon.Debug = m.bridge.Config.Debug

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

	m.bridge.SetupIRCConnection(innerCon, hostname, ip)

	con := &ircConnection{
		innerCon: innerCon,

		discord: user,
		nick:    nick,

		messages:      make(chan IRCMessage),
		cooldownTimer: nil,

		manager: m,
	}

	con.innerCon.AddCallback("001", con.OnWelcome)
	con.innerCon.AddCallback("PRIVMSG", con.OnPrivateMessage)

	m.ircConnections[user.ID] = con

	err := con.innerCon.Connect(m.bridge.Config.IRCServer)
	if err != nil {
		log.WithField("error", err).Errorln("error opening irc connection")
		return
	}

	go innerCon.Loop()

	return
}

// Converts a nickname to a sanitised form.
// Does not check IRC or Discord existence, so don't use this method
// unless you're also checking IRC and Discord.
func sanitiseNickname(nick string) string {
	// https://github.com/lp0/charybdis/blob/9ced2a7932dddd069636fe6fe8e9faa6db904703/ircd/client.c#L854-L884
	if nick[0] == '-' {
		nick = "_" + nick
	}
	if ircnick.IsDigit(nick[0]) {
		nick = "_" + nick
	}

	newNick := []byte(nick)

	// Replace bad characters with underscores
	for i, c := range []byte(nick) {
		if !ircnick.IsNickChar(c) || ircnick.IsFakeNickChar(c) {
			newNick[i] = ' '
		}
	}

	// Now every invalid character has been replaced with a space (just some invalid character)
	// Lets replace each sequence of invalid characters with a single underscore
	newNick = regexp.MustCompile(` +`).ReplaceAllLiteral(newNick, []byte{'_'})

	return string(newNick)
}

func (m *IRCManager) generateNickname(discord DiscordUser) string {
	nick := sanitiseNickname(discord.Nick)
	suffix := m.bridge.Config.Suffix
	newNick := nick + suffix

	useFallback := len(newNick) > ircnick.MAXLENGTH || m.bridge.ircListener.DoesUserExist(newNick)
	// log.WithFields(log.Fields{
	// 	"length":      len(newNick) > ircnick.MAXLENGTH,
	// 	"useFallback": useFallback,
	// }).Infoln("nickgen: fallback?")

	if !useFallback {
		guild, err := m.bridge.discord.State.Guild(m.bridge.Config.GuildID)
		if err != nil {
			// log.Fatalln("nickgen: guild not found when generating nickname")
			return ""
		}

		for _, member := range guild.Members {
			if member.User.ID == discord.ID {
				continue
			}

			name := member.Nick
			if member.Nick == "" {
				name = member.User.Username
			}

			if name == "" {
				log.WithField("member", member).Errorln("blank username encountered")
				continue
			}

			if sanitiseNickname(name) == nick {
				// log.WithField("member", member).Infoln("nickgen: using fallback because of discord")
				useFallback = true
				break
			}
		}
	}

	if useFallback {
		discriminator := discord.Discriminator
		username := sanitiseNickname(discord.Username)
		suffix = "~" + discriminator + suffix

		// Maximum length of a username but without the suffix
		length := ircnick.MAXLENGTH - len(suffix)
		if length >= len(username) {
			length = len(username)
			// log.Infoln("nickgen: maximum length limit not reached")
		}

		newNick = username[:length] + suffix
		// log.WithFields(log.Fields{
		// 	"nick":     discord.Nick,
		// 	"username": discord.Username,
		// 	"newNick":  newNick,
		// }).Infoln("nickgen: resultant nick after falling back")
		return newNick
	}

	// log.WithFields(log.Fields{
	// 	"nick":     discord.Nick,
	// 	"username": discord.Username,
	// 	"newNick":  newNick,
	// }).Infoln("nickgen: resultant nick WITHOUT falling back")

	return newNick
}

func (m *IRCManager) SendMessage(channel string, msg *DiscordMessage) {
	con, ok := m.ircConnections[msg.Author.ID]

	content := msg.Content

	// Person is appearing offline (or the bridge is running in Simple Mode)
	if !ok {
		length := len(msg.Author.Username)
		m.bridge.ircListener.Privmsg(channel, fmt.Sprintf(
			"<%s#%s> %s",
			msg.Author.Username[:1]+"\u200B"+msg.Author.Username[1:length],
			msg.Author.Discriminator,
			content,
		))
		return
	}

	// If there is a cooldown, reset the cooldown
	if con.cooldownTimer != nil {
		m.SetConnectionCooldown(con)
	}

	ircMessage := IRCMessage{
		IRCChannel: channel,
		Message:    content,
		IsAction:   msg.IsAction,
	}

	select {
	// Try to send the message immediately
	case con.messages <- ircMessage:
	// If it can't after 5ms, do it in a separate goroutine
	case <-time.After(time.Millisecond * 5):
		go func() {
			con.messages <- ircMessage
		}()
	}
}

// TODO
// Find all the Discord channels this user belongs to,
// and then find pairings in the global pairings list
// Currently just returns all participating IRC channels
func (m *IRCManager) RequestChannels(userID string) []string {
	return m.bridge.GetIRCChannels()
}
