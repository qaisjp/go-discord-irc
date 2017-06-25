package bridge

import (
	"fmt"
)

// The brains of the operation.
// We coordinate concurrency between all connections and data stores.
// I also write a GUI interface in Visual Basic to track your IP address.
// TODO: Rename to something less comfortable
type home struct {
	dib         *Bridge
	discord     *discordBot
	ircListener *ircListener
	ircManager  *ircManager

	done chan interface{}

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

		done: make(chan interface{}),

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
			_, err := h.discord.ChannelMessageSend(h.dib.chanMapToDiscord[msg.ircChannel], msg.str)
			if err != nil {
				fmt.Println("Message from IRC to Discord was unsuccessfully sent!", err.Error())
			}

		// Messages from Discord to IRC
		case msg := <-h.discordMessageEventsChan:
			ircChan := h.dib.chanMapToIRC[msg.channelID]
			if ircChan == "" {
				fmt.Println("Ignoring message sent from an unhandled channel.")
				continue
			}

			h.ircManager.SendMessage(msg.userID, ircChan, msg.message)

		// Notification to potentially update, or create, a user
		case user := <-h.updateUserChan:
			if user.ID != "83386293446246400" {
				continue
			}

			h.ircManager.CreateConnection(user.ID, user.Discriminator, user.Nick, user.Bot)
		// Done!
		case <-h.done:
			fmt.Println("Closing all connections!")
			h.discord.Close()
			h.ircListener.Disconnect()
			h.ircManager.DisconnectAll()

		default:
		}

	}
}
