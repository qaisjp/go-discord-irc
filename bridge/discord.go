package bridge

import (
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

type discordBot struct {
	*discordgo.Session
	h *home

	guildID string
}

func prepareDiscord(dib *Bridge, botToken, guildID string) (*discordBot, error) {

	// Create a new Discord session using the provided bot token.
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, errors.Wrap(err, "discord, could not create new session")
	}
	session.StateEnabled = true

	discord := &discordBot{session, nil, guildID}

	// These events are all fired in separate goroutines
	discord.AddHandler(discord.onMessageCreate)
	discord.AddHandler(discord.onMemberListChunk)
	discord.AddHandler(discord.onMemberUpdate)
	discord.AddHandler(discord.OnPresencesReplace)
	discord.AddHandler(discord.OnPresenceUpdate)
	discord.AddHandler(discord.OnReady)

	return discord, nil
}

func (d *discordBot) Open() error {
	err := d.Session.Open()
	if err != nil {
		return errors.Wrap(err, "discord, could not open session")
	}

	wh, err := d.GuildWebhooks(d.h.dib.Config.GuildID)
	if err != nil {
		restErr := err.(*discordgo.RESTError)
		if restErr.Message != nil && restErr.Message.Code == 50013 {
			fmt.Println("ERROR: The bot does not have the 'Manage Webhooks' permission.")
			os.Exit(1)
		}

		panic(err)
	}

	mappings := []*Mapping{}
	for _, hook := range wh {
		if strings.HasPrefix(hook.Name, "IRC: #") {
			mappings = append(mappings, &Mapping{
				Webhook:    hook,
				IRCChannel: strings.TrimPrefix(hook.Name, "IRC: "),
			})
		}
	}

	// Check for duplicate channels
	for i, mapping := range mappings {
		for j, check := range mappings {
			if (mapping.ChannelID == check.ChannelID) || (mapping.IRCChannel == check.IRCChannel) {
				if i != j {
					fmt.Printf("Check channel %s or %s for duplicate webhook entries.\n", check.ChannelID, check.IRCChannel)
					os.Exit(1)
				}
			}
		}
	}

	d.h.Mappings = mappings

	return nil
}

func (d *discordBot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}

	isAction := len(m.Content) > 2 &&
		m.Content[0] == '_' &&
		m.Content[len(m.Content)-1] == '_'
	content := m.Content
	if isAction {
		content = content[1 : len(m.Content)-1]
	}

	d.h.discordMessageEventsChan <- &DiscordMessage{
		Message:  m.Message,
		Content:  content,
		IsAction: isAction,
	}
}

func (d *discordBot) onMemberListChunk(s *discordgo.Session, m *discordgo.GuildMembersChunk) {
	for _, m := range m.Members {
		d.handleMemberUpdate(m)
	}
}

func (d *discordBot) onMemberUpdate(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	d.handleMemberUpdate(m.Member)
}

// What does this do? Probably what it sounds like.
func (d *discordBot) OnPresencesReplace(s *discordgo.Session, m *discordgo.PresencesReplace) {
	for _, p := range *m {
		d.handlePresenceUpdate(p)
	}
}

// Handle when presence is updated
func (d *discordBot) OnPresenceUpdate(s *discordgo.Session, m *discordgo.PresenceUpdate) {
	d.handlePresenceUpdate(&m.Presence)
}

func (d *discordBot) handlePresenceUpdate(p *discordgo.Presence) {
	// If they are offline, just deliver a mostly empty struct with the ID and online state
	if p.Status == "offline" {
		d.h.updateUserChan <- DiscordUser{
			ID:     p.User.ID,
			Online: false,
		}
		return
	}

	// Otherwise get their GuildMember object...
	user, err := d.State.Member(d.guildID, p.User.ID)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// .. and handle as per usual
	d.handleMemberUpdate(user)
}

func (d *discordBot) OnReady(s *discordgo.Session, m *discordgo.Ready) {
	d.RequestGuildMembers(d.guildID, "", 0)
}

func (d *discordBot) handleMemberUpdate(m *discordgo.Member) {
	// This error is usually triggered on first run because it represents offline
	presence, err := d.State.Presence(d.guildID, m.User.ID)
	if err != nil {
		// TODO: Determine the type of the error, and handle non-offline situations
		return
	}

	if presence.Status == "offline" {
		return
	}

	d.h.updateUserChan <- DiscordUser{
		ID:            m.User.ID,
		Discriminator: m.User.Discriminator,
		Nick:          GetMemberNick(m),
		Bot:           m.User.Bot,
		Online:        presence.Status != "offline",
	}
}

// See https://github.com/reactiflux/discord-irc/pull/230/files#diff-7202bb7fb017faefd425a2af32df2f9dR357
func (d *discordBot) GetAvatar(guildID, username string) (_ string) {
	// First get all members
	guild, err := d.State.Guild(guildID)
	if err != nil {
		panic(err)
	}

	// Matching members
	var foundMember *discordgo.Member

	// First check an exact match, aborting on multiple
	for _, member := range guild.Members {
		if (username != member.Nick) && (username != member.User.Username) {
			continue
		}

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

			if foundMember == nil {
				foundMember = member
			} else {
				return
			}
		}
	}

	// Do not provide an avatar if:
	// - no matching user OR
	// - multiple matching users
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
