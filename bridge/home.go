package bridge

import (
	"fmt"
)

type home struct {
	dib         *Bridge
	discord     *discordBot
	ircListener *ircListener
	ircManager  *ircManager

	Mappings []*Mapping

	done chan bool

	discordMessagesChan      chan DiscordNewMessage
	discordMessageEventsChan chan DiscordMessageEvent
	updateUserChan           chan DiscordUser
}

func prepareHome(dib *Bridge, discord *discordBot, ircListener *ircListener, ircManager *ircManager) {
	dib.h = &home{
		dib:         dib,
		discord:     discord,
		ircListener: ircListener,
		ircManager:  ircManager,

		done: make(chan bool),

		discordMessagesChan:      make(chan DiscordNewMessage),
		discordMessageEventsChan: make(chan DiscordMessageEvent),
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
			mapping := h.GetMappingByIRC(msg.ircChannel)

			if mapping == nil {
				fmt.Println("Ignoring message sent from an unhandled IRC channel.")
				continue
			}

			_, err := h.discord.ChannelMessageSend(mapping.ChannelID, msg.str)
			if err != nil {
				fmt.Println("Message from IRC to Discord was unsuccessfully sent!", err.Error())
			}

		// Messages from Discord to IRC
		case msg := <-h.discordMessageEventsChan:
			mapping := h.GetMappingByDiscord(msg.channelID)

			if mapping == nil {
				fmt.Println("Ignoring message sent from an unhandled Discord channel.")
				continue
			}

			h.ircManager.SendMessage(msg.userID, mapping.IRCChannel, msg.message)

		// Notification to potentially update, or create, a user
		case user := <-h.updateUserChan:
			// if user.ID != "83386293446246400" {
			// 	continue
			// }

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
