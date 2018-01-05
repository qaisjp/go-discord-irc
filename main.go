package main

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/qaisjp/go-discord-irc/bridge"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	config := flag.String("config", "", "Config file to read configuration stuff from")
	simple := flag.Bool("simple", false, "When in simple mode, the bridge will only spawn one IRC connection for listening and speaking")
	debugMode := flag.Bool("debug", false, "Debug mode? (false = use value from settings)")
	insecure := flag.Bool("insecure", false, "Skip TLS verification? (INSECURE MODE) (false = use value from settings)")

	flag.Parse()

	if *config == "" {
		log.Fatalln("--config argument is required!")
		return
	}

	if *simple {
		log.Println("Running in simple mode.")
	}

	viper := viper.New()
	ext := filepath.Ext(*config)
	configName := strings.TrimSuffix(filepath.Base(*config), ext)
	configType := ext[1:]
	configPath := filepath.Dir(*config)
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	viper.AddConfigPath(configPath)

	log.WithFields(log.Fields{
		"ConfigName": configName,
		"ConfigType": configType,
		"ConfigPath": configPath,
	}).Infoln("Loading configuration...")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalln(errors.Wrap(err, "could not read config"))
	}

	discordBotToken := viper.GetString("discord_token")             // Discord Bot User Token
	channelMappings := viper.GetStringMapString("channel_mappings") // Discord:IRC mappings in format '#discord1:#irc1,#discord2:#irc2,...'
	ircServer := viper.GetString("irc_server")                      // Server address to use, example `irc.freenode.net:7000`.
	guildID := viper.GetString("guild_id")                          // Guild to use
	webIRCPass := viper.GetString("webirc_pass")                    // Password for WEBIRC
	//
	if !*debugMode {
		*debugMode = viper.GetBool("debug")
	}
	//
	if !*insecure {
		*insecure = viper.GetBool("insecure")
	}
	//
	viper.SetDefault("irc_listener_name", "~d")
	ircUsername := viper.GetString("irc_listener_name") // Name for IRC-side bot, for listening to messages.
	//
	viper.SetDefault("suffix", "~d")
	suffix := viper.GetString("suffix") // The suffix to append to IRC connections (not in use when simple mode is on)
	//
	webhookPrefix := viper.GetString("webhook_prefix") // the unique prefix for this bottiful bot

	if webIRCPass == "" {
		log.Warnln("webirc_pass is empty")
	}

	// Validate mappings
	if channelMappings == nil || len(channelMappings) == 0 {
		log.Warnln("Channel mappings are missing!")
	}

	dib, err := bridge.New(&bridge.Config{
		DiscordBotToken:    discordBotToken,
		GuildID:            guildID,
		IRCListenerName:    ircUsername,
		IRCServer:          ircServer,
		WebIRCPass:         webIRCPass,
		Debug:              *debugMode,
		InsecureSkipVerify: *insecure,
		Suffix:             suffix,
		SimpleMode:         *simple,
		ChannelMappings:    channelMappings,
		WebhookPrefix:      webhookPrefix,
	})

	if err != nil {
		log.WithField("error", err).Fatalln("Go-Discord-IRC failed to start.")
		return
	}

	log.Infoln("Go-Discord-IRC is now running. Press Ctrl-C to exit.")

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Println("Configuration file has changed!")
		if newUsername := viper.GetString("irc_listener_name"); ircUsername != newUsername {
			log.Printf("Changed irc_listener_name from '%s' to '%s'", ircUsername, newUsername)
			// Listener name has changed
			ircUsername = newUsername
			dib.SetIRCListenerName(ircUsername)
		}

		if debug := viper.GetBool("debug"); *debugMode != debug {
			log.Printf("Debug changed from %+v to %+v", *debugMode, debug)
			*debugMode = debug
			dib.SetDebugMode(*debugMode)
		}

		chans := viper.GetStringMapString("channel_mappings")
		equalChans := reflect.DeepEqual(chans, channelMappings)
		if !equalChans {
			log.Println("Channel mappings updated!")
			if chans == nil || len(chans) == 0 {
				log.Println("Channel mappings are missing!")
			}

			if err := dib.SetChannelMappings(chans); err != nil {
				channelMappings = chans
			} else {
				log.WithField("error", err).Errorln("could not set channel mappings")
			}
		}
	})

	// Create new signal receiver
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	err = dib.Open()
	if err != nil {
		panic(err)
	}

	// Watch for a signal
	<-sc

	log.Infoln("Shutting down Go-Discord-IRC...")

	// Cleanly close down the Discord session.
	dib.Close()
}
