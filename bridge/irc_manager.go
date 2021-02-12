package bridge

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mozillazg/go-unidecode"
	"github.com/pkg/errors"

	ircnick "github.com/qaisjp/go-discord-irc/irc/nick"
	irc "github.com/qaisjp/go-ircevent"
	log "github.com/sirupsen/logrus"
)

var DevMode = false

// IRCManager should only be used from one thread.
type IRCManager struct {
	ircConnections map[string]*ircConnection
	puppetNicks    map[string]*ircConnection

	bridge *Bridge
}

// NewIRCManager creates a new IRCManager
func newIRCManager(bridge *Bridge) *IRCManager {
	return &IRCManager{
		ircConnections: make(map[string]*ircConnection),
		puppetNicks:    make(map[string]*ircConnection),
		bridge:         bridge,
	}
}

// CloseConnection shuts down a particular connection and its channels.
func (m *IRCManager) CloseConnection(i *ircConnection) {
	log.WithField("nick", i.nick).Println("Closing connection.")
	// Destroy the cooldown timer
	if i.cooldownTimer != nil {
		i.cooldownTimer.Stop()
		i.cooldownTimer = nil
	}

	delete(m.ircConnections, i.discord.ID)
	close(i.messages)

	if DevMode {
		fmt.Println("Decrementing total connections. It's now", len(m.ircConnections))
	}

	if i.innerCon.Connected() {
		i.innerCon.Quit()
	}
}

// Close closes all of an IRCManager's connections.
func (m *IRCManager) Close() {
	i := 0
	for _, con := range m.ircConnections {
		m.CloseConnection(con)
		i++
	}
}

// SetConnectionCooldown renews/starts a timer for expiring a connection.
func (m *IRCManager) SetConnectionCooldown(con *ircConnection) {
	if con.cooldownTimer != nil {
		log.WithField("nick", con.nick).Println("IRC connection cooldownTimer stopped!")
		con.cooldownTimer.Stop()
	}

	con.cooldownTimer = time.AfterFunc(
		m.bridge.Config.CooldownDuration,
		func() {
			log.WithField("nick", con.nick).Println("IRC connection expired by cooldownTimer...")
			m.CloseConnection(con)
		},
	)

	log.WithField("nick", con.nick).Println("IRC connection cooldownTimer created...")
}

// DisconnectUser immediately disconnects a Discord user if it exists
func (m *IRCManager) DisconnectUser(userID string) {
	con, ok := m.ircConnections[userID]
	if !ok {
		return
	}
	m.CloseConnection(con)
}

var connectionsIgnored = 0

// HandleUser deals with messages sent from a DiscordUser
//
// When `user.Online == false`, we make `user.ID` the only other data present in discord.handlePresenceUpdate
func (m *IRCManager) HandleUser(user DiscordUser) {
	// Does the user exist on the IRC side?
	if con, ok := m.ircConnections[user.ID]; ok {
		// Close the connection if they are not
		// online on Discord anymore (after cooldown)
		if !user.Online {
			m.SetConnectionCooldown(con)
			con.SetAway("offline on discord")
		} else {
			// The user is online, destroy any connection cooldown.
			if con.cooldownTimer != nil {
				log.WithField("nick", user.Nick).Println("Destroying connection cooldown.")
				con.cooldownTimer.Stop()
				con.cooldownTimer = nil

				con.SetAway("")
			}
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

	if user.Username == "" || user.Discriminator == "" {
		// If they are not online, we don't care, because this was likely an offline event
		if !user.Online {
			return
		}

		log.WithFields(log.Fields{
			"err":                errors.WithStack(errors.New("Username or Discriminator is empty")).Error(),
			"user.Username":      user.Username,
			"user.Discriminator": user.Discriminator,
			"user.ID":            user.ID,
		}).Println("ignoring a HandleUser (in irc_manager.go)")
		return
	}

	// DEV MODE: Only create a connection if it sounds like qaisjp or if we have 10 connections
	if DevMode {
		if len(m.ircConnections) > 4 && !strings.Contains(user.Username, "qais") {
			connectionsIgnored++
			// fmt.Println("Not letting", user.Username, "connect. We have", len(m.ircConnections), "connections. Ignored", connectionsIgnored, "connections.")
			return
		}
	}

	nick := m.generateNickname(user)
	username := m.generateUsername(user)

	innerCon := irc.IRC(nick, username)
	// innerCon.Debug = m.bridge.Config.Debug
	innerCon.RealName = user.Username
	innerCon.QuitMessage = fmt.Sprintf("Offline for %s", m.bridge.Config.CooldownDuration)

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

		pmNoticedSenders: make(map[string]struct{}),
	}

	con.innerCon.AddCallback("001", con.OnWelcome)
	con.innerCon.AddCallback("PRIVMSG", con.OnPrivateMessage)

	m.ircConnections[user.ID] = con
	m.puppetNicks[nick] = con

	if DevMode {
		fmt.Println("Incrementing total connections. It's now", len(m.ircConnections))
	}

	err := con.innerCon.Connect(m.bridge.Config.IRCServer)
	if err != nil {
		log.WithField("error", err).Errorln("error opening irc connection")
		return
	}

	go innerCon.Loop()
}

// Converts a nickname to a sanitised form.
// Does not check IRC or Discord existence, so don't use this method
// unless you're also checking IRC and Discord.
func sanitiseNickname(nick string) string {
	if nick == "" {
		fmt.Println(errors.WithStack(errors.New("trying to sanitise an empty nick")))
		return "_"
	}

	// Unidecode the nickname — we make sure it's not empty to prevent "🔴🔴" becoming ""
	if newnick := unidecode.Unidecode(nick); newnick != "" {
		nick = newnick
	}

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

	useFallback := len(newNick) > m.bridge.Config.MaxNickLength || m.bridge.ircListener.DoesUserExist(newNick)
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
		suffix = m.bridge.Config.Separator + discriminator + suffix

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

func (m *IRCManager) formatIRCMessage(message *DiscordMessage, content string) string {
	msg := m.bridge.Config.IRCFormat
	length := len(message.Author.Username)
	msg = strings.ReplaceAll(msg, "${USER}", message.Author.Username[:1]+"\u200B"+message.Author.Username[1:length])
	msg = strings.ReplaceAll(msg, "${DISCRIMINATOR}", message.Author.Discriminator)
	msg = strings.ReplaceAll(msg, "${CONTENT}", content)
	return msg
}

// SendMessage sends a broken down Discord Message to a particular IRC channel.
func (m *IRCManager) SendMessage(channel string, msg *DiscordMessage) {
	con, ok := m.ircConnections[msg.Author.ID]

	content := msg.Content

	channel = strings.Split(channel, " ")[0]

	// Person is appearing offline (or the bridge is running in Simple Mode)
	if !ok {
		// length := len(msg.Author.Username)
		for _, line := range strings.Split(content, "\n") {
			// m.bridge.ircListener.Privmsg(channel, fmt.Sprintf(
			// 	"<%s#%s> %s",
			// 	msg.Author.Username[:1]+"\u200B"+msg.Author.Username[1:length],
			// 	msg.Author.Discriminator,
			// 	line,
			// ))

			m.bridge.ircListener.Privmsg(channel, m.formatIRCMessage(msg, line))
		}
		return
	}

	// If there is a cooldown, reset the cooldown
	if con.cooldownTimer != nil {
		m.SetConnectionCooldown(con)
	}

	for _, line := range strings.Split(content, "\n") {
		ircMessage := IRCMessage{
			IRCChannel: channel,
			Message:    line,
			IsAction:   msg.IsAction,
		}

		if strings.HasPrefix(line, "/me ") && len(line) > 4 {
			ircMessage.IsAction = true
			ircMessage.Message = line[4:]
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
}

func (m *IRCManager) formatDiscordMessage(msgFormat string, e *irc.Event, content string, target string) string {
	msg := ""
	if format, ok := m.bridge.Config.DiscordFormat[strings.ToLower(msgFormat)]; ok && format != "" {
		msg = format
		msg = strings.ReplaceAll(msg, "${NICK}", e.Nick)
		msg = strings.ReplaceAll(msg, "${IDENT}", e.User)
		msg = strings.ReplaceAll(msg, "${HOST}", e.Host)
		msg = strings.ReplaceAll(msg, "${CONTENT}", content)
		msg = strings.ReplaceAll(msg, "${TARGET}", target)
		msg = strings.ReplaceAll(msg, "${SERVER}", e.Connection.Server)
		msg = strings.ReplaceAll(msg, "${DISCRIMINATOR}", m.bridge.Config.Discriminator)
	} // else {
	//  should we warn?
	//}

	return msg
}

// RequestChannels finds all the Discord channels this user belongs to,
// and then find pairings in the global pairings list
// Currently just returns all participating IRC channels
// TODO (?)
func (m *IRCManager) RequestChannels(userID string) []Mapping {
	return m.bridge.mappings
}

func (m *IRCManager) generateUsername(discordUser DiscordUser) string {
	if len(m.bridge.Config.PuppetUsername) > 0 {
		return m.bridge.Config.PuppetUsername
	}
	return sanitiseNickname(discordUser.Username)
}
