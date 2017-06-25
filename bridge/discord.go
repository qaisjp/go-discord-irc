package bridge

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
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
		fmt.Println("error creating Discord session,", err)
		return nil, err
	}

	discord := &discordBot{session, nil, guildID}

	// These events are all fired in separate goroutines
	discord.AddHandler(discord.onMessageCreate)
	discord.AddHandler(discord.onMemberListChunk)
	discord.AddHandler(discord.onMemberUpdate)

	return discord, nil
}

func (d *discordBot) Open() error {
	err := d.Session.Open()
	if err != nil {
		return err
	}

	d.RequestGuildMembers(d.guildID, "", 0)
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

	d.h.discordMessageEventsChan <- DiscordMessageEvent{
		userID:    m.Author.ID,
		channelID: m.ChannelID,
		message:   m.Content,
	}
}

func (d *discordBot) onMemberListChunk(s *discordgo.Session, m *discordgo.GuildMembersChunk) {
	fmt.Println("Chunk received.")

	for _, m := range m.Members {
		d.handleMemberUpdate(m)
	}
}

func (d *discordBot) onMemberUpdate(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	fmt.Println("Member updated", m.User.Username, m.Nick)
	d.handleMemberUpdate(m.Member)
}

func (d *discordBot) handleMemberUpdate(m *discordgo.Member) {
	nickname := m.Nick
	if nickname == "" {
		nickname = m.User.Username
	}

	d.h.updateUserChan <- DiscordUser{
		Nick:          nickname,
		Discriminator: m.User.Discriminator,
		ID:            m.User.ID,
		Bot:           m.User.Bot, // this should never change, we can't handle it when this changes, but it's ok
	}
}
