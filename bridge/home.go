package bridge

import (
	"fmt"
)

type home struct {
	dib         *Bridge
	discord     *discordBot
	ircListener *ircListener
	ircManager  *ircManager

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
	return h.dib.chanIRC
}

func (h *home) loop() {
	for {
		select {

		// Messages from IRC to Discord
		case msg := <-h.discordMessagesChan:
			if h.dib.chanMapToDiscord[msg.ircChannel] == "" {
				fmt.Println("Ignoring message sent from an unhandled IRC channel.")
				continue
			}
			_, err := h.discord.ChannelMessageSend(h.dib.chanMapToDiscord[msg.ircChannel], msg.str)
			if err != nil {
				fmt.Println("Message from IRC to Discord was unsuccessfully sent!", err.Error())
			}

		// Messages from Discord to IRC
		case msg := <-h.discordMessageEventsChan:
			ircChan := h.dib.chanMapToIRC[msg.channelID]
			if ircChan == "" {
				fmt.Println("Ignoring message sent from an unhandled Discord channel.")
				continue
			}

			h.ircManager.SendMessage(msg.userID, ircChan, msg.message)

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
