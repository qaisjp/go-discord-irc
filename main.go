package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/qaisjp/go-discord-irc/bridge"
)

func main() {
	discordBotToken := flag.String("discord_token", "", "Discord Bot User Token")
	channelMappings := flag.String("channel_mappings", "", "Discord:IRC mappings in format '#discord1:#irc1,#discord2:#irc2,...'")
	ircUsername := flag.String("irc_listener_name", "~d", "Name for IRC-side bot, for listening to messages.")
	ircServer := flag.String("irc_server", "", "Server address to use, example `irc.freenode.net:7000`.")
	ircNoTLS := flag.Bool("no_irc_tls", false, "Disable TLS for IRC bots?")
	guildID := flag.String("guild_id", "", "Guild to use")
	webIRCPass := flag.String("webirc_pass", "", "Password for WEBIRC")
	debugMode := flag.Bool("debug", false, "Debug mode?")
	insecure := flag.Bool("insecure", false, "Skip TLS verification? (INSECURE MODE)")
	suffix := flag.String("suffix", "~d", "The suffix to append to IRC connections (not in use when simple mode is on)")
	simple := flag.Bool("simple", false, "When in simple mode, the bridge will only spawn one IRC connection for listening and speaking")

	flag.Parse()

	if *webIRCPass == "" {
		fmt.Println("Warning: webirc_pass is empty")
	}

	mappingsMap := validateChannelMappings(*channelMappings)
	if mappingsMap == nil {
		return
	}

	dib, err := bridge.New(&bridge.Config{
		DiscordBotToken:    *discordBotToken,
		GuildID:            *guildID,
		IRCListenerName:    *ircUsername,
		IRCServer:          *ircServer,
		IRCUseTLS:          !*ircNoTLS, // exclamation mark is NOT a typo
		WebIRCPass:         *webIRCPass,
		Debug:              *debugMode,
		InsecureSkipVerify: *insecure,
		Suffix:             *suffix,
		SimpleMode:         *simple,
		ChannelMappings:    mappingsMap,
	})

	if err != nil {
		log.Printf("Go-Discord-IRC failed to start because: %s", err.Error())
		return
	}

	fmt.Println("Go-Discord-IRC is now running. Press Ctrl-C to exit.")

	// Create new signal receiver
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	err = dib.Open()
	if err != nil {
		panic(err)
	}

	// Watch for a signal
	<-sc

	fmt.Println("Shutting down Go-Discord-IRC...")

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
			fmt.Printf("Mapping `%s` must be in the format `discordChannelID:#ircChannel`.\n", mapping)
			valid = false
		}

		if valid {
			discordChannel := sides[0]
			ircChannel := sides[1]

			mappings[discordChannel] = ircChannel
		} else {
			invalidMappings++
		}
	}

	if invalidMappings != 0 {
		fmt.Printf("Channel mappings contains %d errors!\n", invalidMappings)
		return nil
	}

	return mappings
}
