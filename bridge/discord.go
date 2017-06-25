package bridge

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type discordBot struct {
	*discordgo.Session
	h *home
}

func prepareDiscord(dib *Bridge, botToken string) (*discordBot, error) {

	// Create a new Discord session using the provided bot token.
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return nil, err
	}

	discord := &discordBot{session, nil}

	// These events are all fired in separate goroutines
	discord.AddHandler(discord.onMessageCreate)

	return discord, nil
}

func (d *discordBot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// TOOD: Check valid channel

	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}

	d.h.OnDiscordMessage(m.Author.ID, m.ChannelID, m.Content)
}
