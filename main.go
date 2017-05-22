package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/qaisjp/go-discord-irc/bridge"
)

func main() {
	discordBotToken := flag.String("discord_token", "", "Discord Bot User Token")
	channelMappings := flag.String("channel_mappings", "", "Discord:IRC mappings in format '#discord1:#irc1,#discord2:#irc2,...'")

	flag.Parse()

	mappingsMap := validateChannelMappings(*channelMappings)
	if mappingsMap == nil {
		return
	}

	dib, err := bridge.New(bridge.Options{
		DiscordBotToken: *discordBotToken,
		ChannelMappings: mappingsMap,
	})

	if err != nil {
		return
	}

	fmt.Println("Go-Discord-IRC is now running. Press Ctrl-C to exit.")

	// Create new signal receiver
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	err = dib.Open()
	if err != nil {
		return
	}

	// Watch for a signal
	<-sc

	// Cleanly close down the Discord session.
	dib.Close()
}

func validateChannelMappings(rawMappings string) map[string]string {
	mappings := make(map[string]string)

	// Validate mappings
	splitMappings := strings.Split(rawMappings, ",")
	if len(splitMappings) == 1 && splitMappings[0] == "" {
		fmt.Println("Channel mappings are missing!")
		return nil
	}

	invalidMappings := 0
	for _, mapping := range splitMappings {
		sides := strings.Split(mapping, ":")
		valid := true

		if len(sides) != 2 {
			fmt.Printf("Mapping `%s` must be in the format `#discordChannel:#ircChannel`.\n", mapping)
			valid = false
		}

		if valid {
			discordChannel := sides[0]
			ircChannel := sides[1]

			mappings[discordChannel] = ircChannel
		} else {
			invalidMappings += 1
		}
	}

	if invalidMappings != 0 {
		fmt.Printf("Channel mappings contains %d errors!\n", invalidMappings)
		return nil
	}

	return mappings
}
