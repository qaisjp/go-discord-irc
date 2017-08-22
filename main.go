package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/qaisjp/go-discord-irc/bridge"
)

func main() {
	discordBotToken := flag.String("discord_token", "", "Discord Bot User Token")
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
