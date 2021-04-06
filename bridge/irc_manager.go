package bridge

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mozillazg/go-unidecode"
	"github.com/pkg/errors"

	ircnick "github.com/qaisjp/go-discord-irc/irc/nick"
	"github.com/qaisjp/go-discord-irc/irc/varys"
	irc "github.com/qaisjp/go-ircevent"
	log "github.com/sirupsen/logrus"
)

// DevMode is a hack
var DevMode = false

// IRCManager should only be used from one thread.
type IRCManager struct {
	ircConnections map[string]*ircConnection
	puppetNicks    map[string]*ircConnection

	bridge *Bridge
	varys  varys.Client
}

// NewIRCManager creates a new IRCManager
func newIRCManager(bridge *Bridge) (*IRCManager, error) {
	conf := bridge.Config
	m := &IRCManager{
		ircConnections: make(map[string]*ircConnection),
		puppetNicks:    make(map[string]*ircConnection),
		bridge:         bridge,
	}

	// Set up varys
	if conf.VarysServer == "" {
		m.varys = varys.NewMemClient()
	} else {
		log.Infoln("Connecting to varys host:", conf.VarysServer)
		m.varys = varys.NewNetClient(conf.VarysServer, m.HandleVarysCallback)
		log.Infoln("Connected to varys!")
	}
	err := m.varys.Setup(varys.SetupParams{
		UseTLS:             !conf.NoTLS,
		InsecureSkipVerify: conf.InsecureSkipVerify,

		Server:         conf.IRCServer,
		ServerPassword: conf.IRCServerPass,
		WebIRCPassword: conf.WebIRCPass,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set up params: %w", err)
	}

	// Sync back state, if there is any
	discordToNicks, err := m.varys.GetUIDToNicks()
	if err != nil {
		return nil, fmt.Errorf("failed to get discordToNicks: %w", err)
	}
	m.ircConnections = make(map[string]*ircConnection, len(discordToNicks))
	m.puppetNicks = make(map[string]*ircConnection, len(discordToNicks))
	for discord, nick := range discordToNicks {
		m.ircConnections[discord] = &ircConnection{
			discord:          DiscordUser{ID: discord},
			nick:             nick,
			messages:         make(chan IRCMessage),
			manager:          m,
			pmNoticedSenders: make(map[string]struct{}),
		}
	}

	return m, nil
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
	delete(m.puppetNicks, i.nick)
	close(i.messages)

	if DevMode {
		fmt.Println("Decrementing total connections. It's now", len(m.ircConnections))
	}

	if err := m.varys.QuitIfConnected(i.discord.ID, i.quitMessage); err != nil {
		log.WithError(err).WithFields(log.Fields{"discord": i.discord.ID}).Errorln("failed to quit")
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

// HandleVarysCallback responds to callbacks sent by varys
func (m *IRCManager) HandleVarysCallback(uid string, e *irc.Event) {
	fmt.Println("[HandleVarysCallback] Pre-lookup")
	conn, ok := m.ircConnections[uid]
	// todo: what if a callback comes back after ircManager thinks it's gone?
	if !ok {
		panic(fmt.Sprintf("[HandleVarysCallback] uid %#v missing", uid))
	}

	fmt.Println("[HandleVarysCallback] Post-lookup")
	if e.Code == "001" {
		conn.OnWelcome(e)
	} else if e.Code == "PRIVMSG" {
		conn.OnPrivateMessage(e)
	}
	fmt.Println("[HandleVarysCallback] Post-fanout")
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

func (m *IRCManager) ircIgnoredDiscord(user string) bool {
	_, ret := m.bridge.Config.DiscordIgnores[user]
	return ret
}

// HandleUser deals with messages sent from a DiscordUser
//
// When `user.Online == false`, we make `user.ID` the only other data present in discord.handlePresenceUpdate
func (m *IRCManager) HandleUser(user DiscordUser) {
	if m.ircIgnoredDiscord(user.ID) {
		return
	}
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

	// Don't connect them if we're over our configured connection limit! (Includes our listener)
	if m.bridge.Config.ConnectionLimit > 0 && len(m.ircConnections)+1 >= m.bridge.Config.ConnectionLimit {
		return
	}

	nick := m.generateNickname(user)
	username := m.generateUsername(user)

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

	con := &ircConnection{
		discord:          user,
		nick:             nick,
		messages:         make(chan IRCMessage),
		manager:          m,
		pmNoticedSenders: make(map[string]struct{}),
		quitMessage:      fmt.Sprintf("Offline for %s", m.bridge.Config.CooldownDuration),
	}

	m.ircConnections[user.ID] = con
	m.puppetNicks[nick] = con

	if DevMode {
		fmt.Println("Incrementing total connections. It's now", len(m.ircConnections))
	}

	connectParams := varys.ConnectParams{}
	{
		connectParams.UID = user.ID

		connectParams.Nick = nick
		connectParams.Username = username
		connectParams.RealName = user.Username

		connectParams.WebIRCSuffix = fmt.Sprintf("discord %s %s", hostname, ip)

		connectParams.Callbacks = []string{"001", "PRIVMSG"}
	}

	err := m.varys.Connect(connectParams)
	if err != nil {
		log.WithError(err).Errorln("error opening irc connection")
		return
	}
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
		guild, err := m.bridge.discord.Session.State.Guild(m.bridge.Config.GuildID)
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

			if strings.EqualFold(sanitiseNickname(name), nick) {
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

// SendMessage sends a broken down Discord Message to a particular IRC channel.
func (m *IRCManager) SendMessage(channel string, msg *DiscordMessage) {
	if m.ircIgnoredDiscord(msg.Author.ID) {
		return
	}

	con, ok := m.ircConnections[msg.Author.ID]

	content := msg.Content

	channel = strings.Split(channel, " ")[0]

	// Person is appearing offline (or the bridge is running in Simple Mode)
	if !ok {
		length := len(msg.Author.Username)
		for _, line := range strings.Split(content, "\n") {
			m.bridge.ircListener.Privmsg(channel, fmt.Sprintf(
				"<%s#%s> %s",
				msg.Author.Username[:1]+"\u200B"+msg.Author.Username[1:length],
				msg.Author.Discriminator,
				line,
			))
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

		if m.isFilteredDiscordMessage(line) {
			continue
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

// RequestChannels finds all the Discord channels this user belongs to,
// and then find pairings in the global pairings list
// Currently just returns all participating IRC channels
// TODO (?)
func (m *IRCManager) RequestChannels(userID string) []Mapping {
	return m.bridge.mappings
}

func (m *IRCManager) isIgnoredHostmask(mask string) bool {
	for _, ban := range m.bridge.Config.IRCIgnores {
		if ban.Match(mask) {
			return true
		}
	}
	return false
}

func (m *IRCManager) isFilteredIRCMessage(txt string) bool {
	for _, ban := range m.bridge.Config.IRCFilteredMessages {
		if ban.Match(txt) {
			return true
		}
	}
	return false
}

func (m *IRCManager) isFilteredDiscordMessage(txt string) bool {
	for _, ban := range m.bridge.Config.DiscordFilteredMessages {
		if ban.Match(txt) {
			return true
		}
	}
	return false
}

func (m *IRCManager) generateUsername(discordUser DiscordUser) string {
	if len(m.bridge.Config.PuppetUsername) > 0 {
		return m.bridge.Config.PuppetUsername
	}
	return sanitiseNickname(discordUser.Username)
}
