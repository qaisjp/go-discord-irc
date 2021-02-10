package main

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/qaisjp/go-discord-irc/bridge"
	ircnick "github.com/qaisjp/go-discord-irc/irc/nick"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	config := flag.String("config", "", "Config file to read configuration stuff from")
	simple := flag.Bool("simple", false, "When in simple mode, the bridge will only spawn one IRC connection for listening and speaking")
	debugMode := flag.Bool("debug", false, "Debug mode? (false = use value from settings)")
	notls := flag.Bool("no-tls", false, "Avoids using TLS att all when connecting to IRC server ")
	insecure := flag.Bool("insecure", false, "Skip TLS certificate verification? (INSECURE MODE) (false = use value from settings)")

	// Secret devmode
	devMode := flag.Bool("dev", false, "")
	debugPresence := flag.Bool("debug-presence", false, "Include presence in debug output")

	flag.Parse()
	bridge.DevMode = *devMode

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
	ircPassword := viper.GetString("irc_pass")                      // Optional password for connecting to the IRC server
	ircGlobalPerform := viper.GetStringSlice("global_perform")      // IRC Messages to send @OnWelcome
	guildID := viper.GetString("guild_id")                          // Guild to use
	webIRCPass := viper.GetString("webirc_pass")                    // Password for WEBIRC
	identify := viper.GetString("nickserv_identify")                // NickServ IDENTIFY for Listener
	//
	if !*debugMode {
		*debugMode = viper.GetBool("debug")
	}
	//
	if !*notls {
		*notls = viper.GetBool("no_tls")
	}
	if !*insecure {
		*insecure = viper.GetBool("insecure")
	}
	//
	viper.SetDefault("avatar_url", "https://ui-avatars.com/api/?name=${USERNAME}")
	avatarURL := viper.GetString("avatar_url")
	//
	viper.SetDefault("irc_listener_name", "~d")
	ircUsername := viper.GetString("irc_listener_name") // Name for IRC-side bot, for listening to messages.
	// Name to Connect to IRC puppet account with
	viper.SetDefault("puppet_username", "")
	puppetUsername := viper.GetString("puppet_username")
	//
	viper.SetDefault("suffix", "~d")
	suffix := viper.GetString("suffix") // The suffix to append to IRC connections (not in use when simple mode is on)
	//
	viper.SetDefault("separator", "~")
	separator := viper.GetString("separator")
	//
	webhookPrefix := viper.GetString("webhook_prefix") // the unique prefix for this bottiful bot
	//
	viper.SetDefault("cooldown_duration", int64((time.Hour * 24).Seconds()))
	cooldownDuration := viper.GetInt64("cooldown_duration")
	//
	viper.SetDefault("show_joinquit", false)
	showJoinQuit := viper.GetBool("show_joinquit")
	// Maximum length of user nicks aloud
	viper.SetDefault("max_nick_length", ircnick.MAXLENGTH)
	maxNickLength := viper.GetInt("max_nick_length")

	if webIRCPass == "" {
		log.Warnln("webirc_pass is empty")
	}

	// Validate mappings
	if len(channelMappings) == 0 {
		log.Warnln("Channel mappings are missing!")
	}

	SetLogDebug(*debugMode)

	dib, err := bridge.New(&bridge.Config{
		AvatarURL:          avatarURL,
		DiscordBotToken:    discordBotToken,
		GuildID:            guildID,
		IRCListenerName:    ircUsername,
		IRCServer:          ircServer,
		IRCServerPass:      ircPassword,
		IRCGlobalPerform:   ircGlobalPerform,
		PuppetUsername:     puppetUsername,
		NickServIdentify:   identify,
		WebIRCPass:         webIRCPass,
		NoTLS:              *notls,
		InsecureSkipVerify: *insecure,
		Suffix:             suffix,
		Separator:          separator,
		SimpleMode:         *simple,
		ChannelMappings:    channelMappings,
		WebhookPrefix:      webhookPrefix,
		CooldownDuration:   time.Second * time.Duration(cooldownDuration),
		ShowJoinQuit:       showJoinQuit,
		MaxNickLength:      maxNickLength,

		Debug:         *debugMode,
		DebugPresence: *debugPresence,
	})

	if err != nil {
		log.WithField("error", err).Fatalln("Go-Discord-IRC failed to initialise.")
		return
	}

	log.Infoln("Cooldown duration for IRC puppets is", dib.Config.CooldownDuration)

	// Create new signal receiver
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	// Open the bot
	err = dib.Open()
	if err != nil {
		log.WithField("error", err).Fatalln("Go-Discord-IRC failed to start.")
		return
	}

	// Inform the user that things are happening!
	log.Infoln("Go-Discord-IRC is now running. Press Ctrl-C to exit.")

	// Start watching for live changes...
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Println("Configuration file has changed!")
		if newUsername := viper.GetString("irc_listener_name"); ircUsername != newUsername {
			log.Printf("Changed irc_listener_name from '%s' to '%s'", ircUsername, newUsername)
			// Listener name has changed
			ircUsername = newUsername
			dib.SetIRCListenerName(ircUsername)
		}

		avatarURL := viper.GetString("avatar_url")
		dib.Config.AvatarURL = avatarURL

		if debug := viper.GetBool("debug"); *debugMode != debug {
			log.Printf("Debug changed from %+v to %+v", *debugMode, debug)
			*debugMode = debug
			dib.SetDebugMode(debug)
			SetLogDebug(debug)
		}

		chans := viper.GetStringMapString("channel_mappings")
		equalChans := reflect.DeepEqual(chans, channelMappings)
		if !equalChans {
			log.Println("Channel mappings updated!")
			if len(chans) == 0 {
				log.Println("Channel mappings are missing!")
			}

			if err := dib.SetChannelMappings(chans); err != nil {
				log.WithField("error", err).Errorln("could not set channel mappings")
			} else {
				channelMappings = chans
			}
		}
	})

	// Watch for a shutdown signal
	<-sc

	log.Infoln("Shutting down Go-Discord-IRC...")

	// Cleanly close down the bridge.
	dib.Close()
}

func SetLogDebug(debug bool) {
	logger := log.StandardLogger()
	if debug {
		logger.SetLevel(log.DebugLevel)
	} else {
		logger.SetLevel(log.InfoLevel)
	}
}
