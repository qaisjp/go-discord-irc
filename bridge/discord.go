package bridge

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/42wim/matterbridge/bridge/discord/transmitter"
	"github.com/qaisjp/go-discord-irc/dstate"
	ircnick "github.com/qaisjp/go-discord-irc/irc/nick"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type discordBot struct {
	Session *discordgo.Session
	bridge  *Bridge

	guildID string

	transmitter *transmitter.Transmitter
}

func newDiscord(bridge *Bridge, botToken, guildID string) (*discordBot, error) {

	// Create a new Discord session using the provided bot token.
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, errors.Wrap(err, "discord, could not create new session")
	}
	session.StateEnabled = true

	discord := &discordBot{
		Session: session,
		bridge:  bridge,

		guildID: guildID,
	}

	// These events are all fired in separate goroutines
	discord.Session.AddHandler(discord.OnReady)
	discord.Session.AddHandler(discord.onMessageCreate)
	discord.Session.AddHandler(discord.onMessageUpdate)
	discord.Session.AddHandler(discord.onGuildEmojiUpdate)

	if !bridge.Config.SimpleMode {
		discord.Session.AddHandler(discord.onMemberListChunk)
		discord.Session.AddHandler(discord.onMemberUpdate)
		discord.Session.AddHandler(discord.onMemberLeave)
		discord.Session.AddHandler(discord.OnPresencesReplace)
		discord.Session.AddHandler(discord.OnPresenceUpdate)
		discord.Session.AddHandler(discord.OnTypingStart)
		discord.Session.AddHandler(discord.OnMessageReactionAdd)
	}

	return discord, nil
}

func (d *discordBot) Open() error {
	d.transmitter = transmitter.New(d.Session, d.guildID, "irc-bridge", true)
	d.transmitter.Log = logrus.NewEntry(logrus.StandardLogger())
	if err := d.transmitter.RefreshGuildWebhooks(nil); err != nil {
		return fmt.Errorf("failed to refresh guild webhooks: %w", err)
	}

	d.Session.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAll)
	err := d.Session.Open()
	if err != nil {
		return errors.Wrap(err, "discord, could not open session")
	}

	return nil
}

func (d *discordBot) Close() error {
	return errors.Wrap(d.Session.Close(), "closing discord session")
}

// Returns `<@uid>` if a discord user or just `name` if a bot
func userToMention(u *discordgo.User) (mention string) {
	mention = u.Username
	if !u.Bot {
		mention = u.Mention()
	}
	return
}

// For spoiler colouring:
var spoilerPattern = regexp.MustCompile(`\|\|(.*?)\|\|`)
var colorCode = string(rune(3))

func (d *discordBot) publishMessage(s *discordgo.Session, m *discordgo.Message, wasEdit bool) {
	// Fix crash if these fields don't exist
	if m.Author == nil || s.State.User == nil {
		// todo: add sentry logging
		return
	}

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Ignore messages sent from our webhooks
	if d.transmitter.HasWebhook(m.Author.ID) {
		return
	}

	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		_, err := s.ChannelMessageSend(m.ChannelID, "Pong!")
		if err != nil {
			log.Warningln("Could not respond to Discord ping message", err.Error())
		}
	}

	// HACK: this is before d.ParseText so that the existing <@uid> translation logic can be used
	if m.MessageReference != nil && m.MessageReference.ChannelID == m.ChannelID {
		prefix := "[reply]"
		msg, err := dstate.ChannelMessage(d.Session, m.MessageReference.ChannelID, m.MessageReference.MessageID)
		if err == nil {
			prefix = userToMention(msg.Author) + ":"
			if !msg.Author.Bot {
				// HACK: theoretically could already be there, thereotically not a big problem
				m.Mentions = append(m.Mentions, msg.Author)
			}
		}
		m.Content = prefix + " " + m.Content
	}

	content := d.ParseText(m)

	// The content is an action if it matches "_(.+)_"
	isAction := len(content) > 2 &&
		content[0] == '_' &&
		content[len(content)-1] == '_'

	// If it is an action, remove the enclosing underscores
	if isAction {
		content = content[1 : len(content)-1]
	}

	if wasEdit {
		if isAction {
			content = "/me " + content
		}

		content = "[edit] " + content
	}

	if strings.Count(content, "||") >= 2 {
		content = spoilerPattern.ReplaceAllString(content, colorCode+"1,1$1"+colorCode)
	}

	pmTarget := ""
	// Blank guild means that it's a PM
	if m.GuildID == "" {
		pmTarget, content = pmTargetFromContent(content, d.bridge.Config.Discriminator)
		// if the target could not be deduced. tell them this.
		switch pmTarget {
		case "":
			_, _ = d.Session.ChannelMessageSend(
				m.ChannelID,
				fmt.Sprintf(
					"Don't know who that is. Can't PM. Try 'name@%s, message here'",
					d.bridge.Config.Discriminator))
			return
		case "*UNKNOWN*":
			return
		default:
			break
		}
	}

	d.bridge.discordMessageEventsChan <- &DiscordMessage{
		Message:  m,
		Content:  content,
		IsAction: isAction,
		PmTarget: pmTarget,
	}

	for _, attachment := range m.Attachments {
		d.bridge.discordMessageEventsChan <- &DiscordMessage{
			Message:  m,
			Content:  attachment.URL,
			IsAction: isAction,
			PmTarget: pmTarget,
		}
	}
}

func (d *discordBot) publishReaction(s *discordgo.Session, r *discordgo.MessageReaction) {
	if s.State.User == nil {
		return
	}

	user, err := s.User(r.UserID)
	if err != nil {
		log.Errorln(err)
		return
	}

	// Bridge needs these for mapping
	m := &discordgo.Message{
		ChannelID: r.ChannelID,
		Author:    user,
		GuildID:   r.GuildID,
	}

	originalMessage, err := dstate.ChannelMessage(d.Session, r.ChannelID, r.MessageID)
	reactionTarget := ""
	if err == nil {
		// TODO 1: could add extra logic to figure out what length is needed to disambiguate
		// TODO 2: length should not cause command to exceed the max command length

		// HACK: this is before d.ParseText so that the existing <@uid> translation logic can be used
		username := userToMention(originalMessage.Author)
		if !originalMessage.Author.Bot {
			// HACK: theoretically could already be there, thereotically not a big problem
			originalMessage.Mentions = append(originalMessage.Mentions, originalMessage.Author)
		}
		originalMessage.Content = fmt.Sprintf(
			" to <%s> %s",
			username,
			// Truncate messages to just 40 characters so reactions to long messages
			// don't pollute the IRC log. Similarly, replace newlines with spaces
			// so that any reactions to messages with a newline within the first 40
			// characters don't cause multiple IRC messages to be sent.
			strings.ReplaceAll(TruncateString(40, originalMessage.Content), "\n", " "),
		)

		reactionTarget = d.ParseText(originalMessage)
	}

	emoji := r.Emoji.Name
	if r.Emoji.ID != "" {
		// Custom emoji
		emoji = fmt.Sprint(":", emoji, ":")
	}
	content := fmt.Sprint("reacted with ", emoji, reactionTarget)

	d.bridge.discordMessageEventsChan <- &DiscordMessage{
		Message:  m,
		Content:  content,
		IsAction: true,
		PmTarget: "",
	}
}

// Up to date as of https://git.io/v5kJg
var channelMention = regexp.MustCompile(`<#(\d+)>`)
var roleMention = regexp.MustCompile(`<@&(\d+)>`)

var patternChannels = regexp.MustCompile("<#[^>]*>")
var emoteRegex = regexp.MustCompile(`<a?(:\w+:)\d+>`)

// Up to date as of https://git.io/v5kJg
func (d *discordBot) ParseText(m *discordgo.Message) string {
	// Replace @user mentions with name~d mentions
	content := m.Content

	// Convert embeds to plaintext
	for _, embed := range m.Embeds {
		if embed.Title != "" {
			content += fmt.Sprintf("--- %s\n", embed.Title)
		}
		if embed.Description != "" {
			content += fmt.Sprintf("%s\n", embed.Description)
		}
		if embed.Footer != nil && embed.Footer.Text != "" {
			content += fmt.Sprintf("%s\n", embed.Footer.Text)
		}
		if embed.Author != nil && embed.Author.Name != "" {
			content += fmt.Sprintf("Author: %s\n", embed.Author.Name)
		}
		if embed.URL != "" {
			content += fmt.Sprintf("URL: %s\n", embed.URL)
		}
		if embed.Timestamp != "" {
			content += fmt.Sprintf("Timestamp: %s\n", embed.Timestamp)
		}
	}

	for _, user := range m.Mentions {
		// Find the irc username with the discord ID in irc connections
		username := ""
		for _, u := range d.bridge.ircManager.ircConnections {
			if u.discord.ID == user.ID {
				username = u.nick
			}
		}

		if username == "" {
			// Nickname is their username by default
			nick := user.Username

			// If we can get their member + nick, set nick to the real nick
			member, err := d.Session.State.Member(d.guildID, user.ID)
			if err == nil && member.Nick != "" {
				nick = member.Nick
			}

			username = d.bridge.ircManager.generateNickname(DiscordUser{
				ID:            user.ID,
				Username:      user.Username,
				Discriminator: user.Discriminator,
				Nick:          nick,
				Bot:           user.Bot,
				Online:        false,
			})

			log.WithFields(log.Fields{
				"discord-username": user.Username,
				"irc-username":     username,
				"discord-id":       user.ID,
			}).Infoln("Could not convert mention using existing IRC connection")
		} else {
			log.WithFields(log.Fields{
				"discord-username": user.Username,
				"irc-username":     username,
				"discord-id":       user.ID,
			}).Infoln("Converted mention using existing IRC connection")
		}

		content = strings.NewReplacer(
			"<@"+user.ID+">", username,
			"<@!"+user.ID+">", username,
		).Replace(content)
	}

	// Copied from message.go ContentWithMoreMentionsReplaced(s)
	for _, roleID := range m.MentionRoles {
		role, err := d.Session.State.Role(d.guildID, roleID)
		if err != nil || !role.Mentionable {
			continue
		}

		content = strings.Replace(content, "<&"+role.ID+">", "@"+role.Name, -1)
	}

	// Also copied from message.go ContentWithMoreMentionsReplaced(s)
	content = patternChannels.ReplaceAllStringFunc(content, func(mention string) string {
		channel, err := d.Session.State.Channel(mention[2 : len(mention)-1])
		if err != nil || channel.Type == discordgo.ChannelTypeGuildVoice {
			return mention
		}

		return "#" + channel.Name
	})

	// Break down malformed newlines
	content = strings.Replace(content, "\r\n", "\n", -1) // replace CRLF with LF
	content = strings.Replace(content, "\r", "\n", -1)   // replace CR with LF

	// Replace <#xxxxx> channel mentions
	content = channelMention.ReplaceAllStringFunc(content, func(str string) string {
		// Strip enclosing identifiers
		channelID := str[2 : len(str)-1]

		channel, err := d.Session.State.Channel(channelID)
		if err == nil {
			return "#" + channel.Name
		} else if err == discordgo.ErrStateNotFound {
			return "#deleted-channel"
		}

		panic(errors.Wrap(err, "Channel mention failed for "+str))
	})

	// Replace <@&xxxxx> role mentions
	content = roleMention.ReplaceAllStringFunc(content, func(str string) string {
		// Strip enclosing identifiers
		roleID := str[3 : len(str)-1]

		role, err := d.Session.State.Role(d.bridge.Config.GuildID, roleID)
		if err == nil {
			return "@" + role.Name
		} else if err == discordgo.ErrStateNotFound {
			return "@deleted-role"
		}

		panic(errors.Wrap(err, "Channel mention failed for "+str))
	})

	// Replace emotes
	content = emoteRegex.ReplaceAllString(content, "$1")

	return content
}

func (d *discordBot) handlePresenceUpdate(uid string, status discordgo.Status, forceOnline bool) {
	// If they are offline, just deliver a mostly empty struct with the ID and online state
	if !forceOnline && !isStatusOnline(status) {
		if d.bridge.Config.DebugPresence {
			log.WithField("id", uid).Debugln("PRESENCE", status, "(handlePresenceUpdate - Online: false)")
		}
		d.sendUpdateUserChan(DiscordUser{
			ID:     uid,
			Online: false,
		})
		return
	}

	if d.bridge.Config.DebugPresence {
		log.WithField("id", uid).Debugln("PRESENCE", status, "(handlePresenceUpdate)")
	}

	// Otherwise get their GuildMember object...
	user, err := d.Session.State.Member(d.guildID, uid)
	if err != nil {
		log.Println(errors.Wrap(err, "get member from state in handlePresenceUpdate failed"))
		return
	}

	// .. and handle as per usual
	d.handleMemberUpdate(user, forceOnline)
}

func isStatusOnline(status discordgo.Status) bool {
	return status != discordgo.StatusOffline
}

func (d *discordBot) sendUpdateUserChan(user DiscordUser) bool {
	// Only log this for online events, because offline events won't have this
	if (user.Username == "" || user.Discriminator == "") && user.Online {
		log.WithFields(log.Fields{
			"err":                errors.WithStack(errors.New("Username or Discriminator is empty")).Error(),
			"user.Username":      user.Username,
			"user.Discriminator": user.Discriminator,
			"user.ID":            user.ID,
		}).Println("sendUpdateUserChan called with empty Username and Discriminator (see stack below)")
		debug.PrintStack()
	}

	d.bridge.updateUserChan <- user
	return true
}

// See https://github.com/reactiflux/discord-irc/pull/230/files#diff-7202bb7fb017faefd425a2af32df2f9dR357
func (d *discordBot) GetAvatar(guildID, username string) (_ string) {
	// First get all members
	guild, err := d.Session.State.Guild(guildID)
	if err != nil {
		panic(err)
	}

	// Matching members
	var foundMember *discordgo.Member

	// Try and find an exact case-sensitive match
	for _, member := range guild.Members {
		if (username != member.Nick) && (username != member.User.Username) {
			continue
		}

		// If there are multiple matches, return an empty string
		if foundMember == nil {
			foundMember = member
		} else {
			return
		}
	}

	// If no member found, check case-insensitively
	if foundMember == nil {
		for _, member := range guild.Members {
			if !strings.EqualFold(username, member.Nick) && !strings.EqualFold(username, member.User.Username) {
				continue
			}

			// If there are multiple matches, return an empty string
			if foundMember == nil {
				foundMember = member
			} else {
				return
			}
		}
	}

	// Do not provide an avatar if there is no matching user
	if foundMember == nil {
		return
	}

	return discordgo.EndpointUserAvatar(foundMember.User.ID, foundMember.User.Avatar)
}

// GetMemberNick returns the real display name for a Discord GuildMember
func GetMemberNick(m *discordgo.Member) string {
	if m.Nick == "" {
		return m.User.Username
	}

	return m.Nick
}

// pmTargetFromContent returns an irc nick given a message sent to an IRC user via Discord
//
// Returns empty string if the nick could not be deduced.
// Also returns the content without the nick
func pmTargetFromContent(content string, discriminator string) (nick, newContent string) {
	// Pull out substrings
	// "qais,come on, i need this!" gives []string{"qais", "come on, i need this!"}
	subs := strings.SplitN(content, ",", 2)

	if len(subs) != 2 {
		return "", ""
	}

	nick = subs[0]
	newContent = strings.TrimPrefix(subs[1], " ")

	nickParts := strings.Split(nick, "@")

	// we were given an invalid nick if we can't split it into 2 parts
	if len(nickParts) < 2 {
		return "", ""
	}

	if nickParts[1] != discriminator {
		return "*UNKNOWN*", ""
	}

	nick = nickParts[0]

	// check if name is a valid nick
	for _, c := range []byte(nick) {
		if !ircnick.IsNickChar(c) {
			return "", ""
		}
	}

	return
}
