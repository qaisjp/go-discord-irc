package bridge

import (
	"fmt"
)

// The brains of the operation.
// We coordinate concurrency between all connections and data stores.
// I also write a GUI interface in Visual Basic to track your IP address.
// TODO: Rename to something less comfortable
type home struct {
	discordMessagesChan chan DiscordMessage
	dib                 *Bridge
}

func prepareHome(dib *Bridge) {
	h := &home{
		discordMessagesChan: make(chan DiscordMessage),
		dib:                 dib,
	}

	dib.h = h

	go h.loop()
}

func (h *home) GetIRCChannels() []string {
	return h.dib.chanIRC
}

func (h *home) SendDiscordMessage(msg DiscordMessage) {
	h.discordMessagesChan <- msg
}

func (h *home) loop() {
	fmt.Println("Loop")

	for {
		select {
		case msg := <-h.discordMessagesChan:
			fmt.Println("Received, sending to", h.dib.chanMapToDiscord[msg.ircChannel])
			_, err := h.dib.dg.ChannelMessageSend(h.dib.chanMapToDiscord[msg.ircChannel], msg.str)
			if err != nil {
				fmt.Println("Message from IRC to Discord was unsuccessfully sent!", err.Error())
			}
		default:
		}

	}
}

type DiscordMessage struct {
	ircChannel string
	str        string
}
