package bridge

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type home struct {
	dib         *Bridge
	discord     *discordBot
	ircListener *ircListener
	ircManager  *ircManager

	Mappings []*Mapping

	done chan bool

	discordMessagesChan      chan IRCMessage
	discordMessageEventsChan chan *DiscordMessage
	updateUserChan           chan DiscordUser
}

func prepareHome(dib *Bridge, discord *discordBot, ircListener *ircListener, ircManager *ircManager) {
	dib.h = &home{
		dib:         dib,
		discord:     discord,
		ircListener: ircListener,
		ircManager:  ircManager,

		done: make(chan bool),

		discordMessagesChan:      make(chan IRCMessage),
		discordMessageEventsChan: make(chan *DiscordMessage),
		updateUserChan:           make(chan DiscordUser),
	}

	go dib.h.loop()
}

func (h *home) GetIRCChannels() []string {
	channels := make([]string, len(h.Mappings))
	for i, mapping := range h.Mappings {
		channels[i] = mapping.IRCChannel
	}

	return channels
}

func (h *home) GetMappingByIRC(channel string) *Mapping {
	for _, mapping := range h.Mappings {
		if mapping.IRCChannel == channel {
			return mapping
		}
	}
	return nil
}

func (h *home) GetMappingByDiscord(channel string) *Mapping {
	for _, mapping := range h.Mappings {
		if mapping.ChannelID == channel {
			return mapping
		}
	}
	return nil
}

func (h *home) loop() {
	for {
		select {

		// Messages from IRC to Discord
		case msg := <-h.discordMessagesChan:
			mapping := h.GetMappingByIRC(msg.IRCChannel)

			if mapping == nil {
				fmt.Println("Ignoring message sent from an unhandled IRC channel.")
				continue
			}

			avatar := h.discord.GetAvatar(mapping.GuildID, msg.Username)
			if avatar == "" {
				// If we don't have a Discord avatar, generate an adorable avatar
				avatar = "https://api.adorable.io/avatars/128/" + msg.Username
			}

			// Get current webhook
			webhook := mapping.Get(msg.Username)

			// TODO: What if it takes a long time? wait=true below.
			err := h.discord.WebhookExecute(webhook.ID, webhook.Token, true, &discordgo.WebhookParams{
				Content:   msg.Message,
				Username:  msg.Username,
				AvatarURL: avatar,
			})

			if err != nil {
				fmt.Println("Message from IRC to Discord was unsuccessfully sent!", err.Error())
			}

		// Messages from Discord to IRC
		case msg := <-h.discordMessageEventsChan:
			mapping := h.GetMappingByDiscord(msg.ChannelID)

			// Do not do anything if we do not have a mapping for the channel
			if mapping == nil {
				fmt.Println("Ignoring message sent from an unhandled Discord channel.")
				continue
			}

			// Ignore messages sent from our webhooks
			fromHook := false
			for _, mapping := range h.Mappings {
				if (mapping.ID == msg.Author.ID) || (mapping.AltHook.ID == msg.Author.ID) {
					fromHook = true
				}
			}
			if fromHook {
				continue
			}

			h.ircManager.SendMessage(mapping.IRCChannel, msg)

		// Notification to potentially update, or create, a user
		case user := <-h.updateUserChan:
			h.ircManager.HandleUser(user)

		// Done!
		case <-h.done:
			h.discord.Close()
			h.ircListener.Quit()
			h.ircManager.Close()
			close(h.done)

			return
		}

	}
}
