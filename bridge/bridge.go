package bridge

import (
	"crypto/tls"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	irc "github.com/qaisjp/go-ircevent"
)

// Config to be passed to New
type Config struct {
	DiscordBotToken, GuildID string

	// Map from Discord to IRC
	ChannelMappings map[string]string

	IRCServer       string
	IRCUseTLS       bool
	IRCListenerName string // i.e, "DiscordBot", required to listen for messages in all cases
	WebIRCPass      string

	// InsecureSkipVerify controls whether a client verifies the
	// server's certificate chain and host name.
	// If InsecureSkipVerify is true, TLS accepts any certificate
	// presented by the server and any host name in that certificate.
	// In this mode, TLS is susceptible to man-in-the-middle attacks.
	// This should be used only for testing.
	InsecureSkipVerify bool

	// SimpleMode, when enabled, will ensure that IRCManager not spawn
	// an IRC connection for each of the online Discord users.
	SimpleMode bool

	Suffix string // Suffix is the suffix to append to Discord users on the IRC side.

	Debug bool
}

// A Bridge represents a bridging between an IRC server and channels in a Discord server
type Bridge struct {
	Config *Config

	discord     *discordBot
	ircListener *ircListener
	ircManager  *IRCManager

	mappings []*Mapping

	done chan bool

	discordMessagesChan      chan IRCMessage
	discordMessageEventsChan chan *DiscordMessage
	updateUserChan           chan DiscordUser
}

// Close the Bridge
func (b *Bridge) Close() {
	b.done <- true
	<-b.done
}

// TODO: Use errors package
func (b *Bridge) load(opts *Config) bool {
	if opts.IRCServer == "" {
		fmt.Println("Missing server name.")
		return false
	}

	return true
}

// New Bridge
func New(conf *Config) (*Bridge, error) {
	dib := &Bridge{
		Config: conf,
		done:   make(chan bool),

		discordMessagesChan:      make(chan IRCMessage),
		discordMessageEventsChan: make(chan *DiscordMessage),
		updateUserChan:           make(chan DiscordUser),
	}

	if !dib.load(conf) {
		return nil, errors.New("error with Config. TODO: More info here")
	}

	var err error

	dib.discord, err = NewDiscord(dib, conf.DiscordBotToken, conf.GuildID)
	if err != nil {
		return nil, errors.Wrap(err, "Could not create discord bot")
	}

	dib.ircListener = NewIRCListener(dib, conf.WebIRCPass)
	dib.ircManager = NewIRCManager(dib)

	go dib.loop()

	return dib, nil
}

// Open all the connections required to run the bridge
func (b *Bridge) Open() (err error) {

	// Open a websocket connection to Discord and begin listening.
	err = b.discord.Open()
	if err != nil {
		return errors.Wrap(err, "can't open discord")
	}

	err = b.ircListener.Connect(b.Config.IRCServer)
	if err != nil {
		return errors.Wrap(err, "can't open irc connection")
	}

	go b.ircListener.Loop()

	return
}

func (b *Bridge) SetupIRCConnection(con *irc.Connection, hostname, ip string) {
	con.UseTLS = true
	con.TLSConfig = &tls.Config{
		InsecureSkipVerify: b.Config.InsecureSkipVerify,
	}

	con.WebIRC = fmt.Sprintf("%s discord %s %s", b.Config.WebIRCPass, hostname, ip)
}

func (b *Bridge) GetIRCChannels() []string {
	channels := make([]string, len(b.mappings))
	for i, mapping := range b.mappings {
		channels[i] = mapping.IRCChannel
	}

	return channels
}

func (b *Bridge) GetMappingByIRC(channel string) *Mapping {
	for _, mapping := range b.mappings {
		if mapping.IRCChannel == channel {
			return mapping
		}
	}
	return nil
}

func (b *Bridge) GetMappingByDiscord(channel string) *Mapping {
	for _, mapping := range b.mappings {
		if mapping.ChannelID == channel {
			return mapping
		}
	}
	return nil
}

func (b *Bridge) loop() {
	for {
		select {

		// Messages from IRC to Discord
		case msg := <-b.discordMessagesChan:
			mapping := b.GetMappingByIRC(msg.IRCChannel)

			if mapping == nil {
				fmt.Println("Ignoring message sent from an unhandled IRC channel.")
				continue
			}

			avatar := b.discord.GetAvatar(mapping.GuildID, msg.Username)
			if avatar == "" {
				// If we don't have a Discord avatar, generate an adorable avatar
				avatar = "https://api.adorable.io/avatars/128/" + msg.Username
			}

			// Get current webhook
			webhook := mapping.Get(msg.Username)

			// TODO: What if it takes a long time? See wait=true below.
			err := b.discord.WebhookExecute(webhook.ID, webhook.Token, true, &discordgo.WebhookParams{
				Content:   msg.Message,
				Username:  msg.Username,
				AvatarURL: avatar,
			})

			if err != nil {
				fmt.Println("Message from IRC to Discord was unsuccessfully sent!", err.Error())
			}

		// Messages from Discord to IRC
		case msg := <-b.discordMessageEventsChan:
			mapping := b.GetMappingByDiscord(msg.ChannelID)

			// Do not do anything if we do not have a mapping for the channel
			if mapping == nil {
				fmt.Println("Ignoring message sent from an unhandled Discord channel.")
				continue
			}

			// Ignore messages sent from our webhooks
			fromHook := false
			for _, mapping := range b.mappings {
				if (mapping.ID == msg.Author.ID) || (mapping.AltHook.ID == msg.Author.ID) {
					fromHook = true
				}
			}
			if fromHook {
				continue
			}

			b.ircManager.SendMessage(mapping.IRCChannel, msg)

		// Notification to potentially update, or create, a user
		// We should not receive anything on this channel if we're in Simple Mode
		case user := <-b.updateUserChan:
			b.ircManager.HandleUser(user)

		// Done!
		case <-b.done:
			b.discord.Close()
			b.ircListener.Quit()
			b.ircManager.Close()
			close(b.done)

			return
		}

	}
}
