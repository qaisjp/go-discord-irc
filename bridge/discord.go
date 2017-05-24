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

	// Register the messageCreate func as a callback for MessageCreate events.
	discord.AddHandler(discord.messageCreate)
	discord.AddHandler(discord.typingStart)

	return discord, nil
}

// func (d *discordBot) AddHandler(handler interface{}) {
// 	d.Session.AddHandler(func() {
// 		go func() {

// 		}()
// 	})
// }

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (d *discordBot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	if !testingChannels(m.ChannelID) {
		return
	}

	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}

	// If the message is "pong" reply with "Ping!"
	if m.Content == "pong" {
		s.ChannelMessageSend(m.ChannelID, "Ping!")
	}

	d.h.OnDiscordMessage(DiscordMessageEvent{
		channelID: m.ChannelID,
		userID:    m.Author.ID,
		message:   m.Content,
	})
}

// This function will be called (due to AddHandler above) every time a discord user
// starts typing
func (d *discordBot) typingStart(s *discordgo.Session, m *discordgo.TypingStart) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.UserID == s.State.User.ID {
		return
	}

	if !testingChannels(m.ChannelID) {
		return
	}

	d.h.SendDiscordUserPulse(DiscordUserPulse{
		channelID: m.ChannelID,
		userID:    m.UserID,
	})
}
