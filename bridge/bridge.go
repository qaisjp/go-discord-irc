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

	Debug bool
}

// A Bridge represents a bridging between an IRC server and channels in a Discord server
type Bridge struct {
	Config *Config
	h      *home
}

type Mapping struct {
	*discordgo.Webhook
	AltHook     *discordgo.Webhook
	UsingAlt    bool
	CurrentUser string
	IRCChannel  string
}

// Get the webhook given a user
func (m *Mapping) Get(user string) *discordgo.Webhook {
	if m.CurrentUser == user {
		return m.Current()
	}

	m.UsingAlt = !m.UsingAlt
	m.CurrentUser = user
	return m.Current()
}

// Get the current webhook
func (m *Mapping) Current() *discordgo.Webhook {
	if m.UsingAlt {
		return m.AltHook
	}

	return m.Webhook
}

// Close the Bridge
func (b *Bridge) Close() {
	b.h.done <- true
	<-b.h.done
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
	}

	if !dib.load(conf) {
		return nil, errors.New("error with Config. TODO: More info here")
	}

	discord, err := prepareDiscord(dib, conf.DiscordBotToken, conf.GuildID)
	ircPrimary := prepareIRCListener(dib, conf.WebIRCPass)
	ircManager := NewIRCManager()

	if err != nil {
		return nil, err
	}

	prepareHome(dib, discord, ircPrimary, ircManager)

	discord.h = dib.h
	ircPrimary.h = dib.h
	ircManager.h = dib.h

	return dib, nil
}

// Open all the connections required to run the bridge
func (b *Bridge) Open() (err error) {

	// Open a websocket connection to Discord and begin listening.
	err = b.h.discord.Open()
	if err != nil {
		return errors.Wrap(err, "can't open discord")
	}

	err = b.h.ircListener.Connect(b.Config.IRCServer)
	if err != nil {
		return errors.Wrap(err, "can't open irc connection")
	}

	go b.h.ircListener.Loop()

	return
}

func (b *Bridge) SetupIRCConnection(con *irc.Connection, hostname, ip string) {
	con.UseTLS = true
	con.TLSConfig = &tls.Config{
		InsecureSkipVerify: b.Config.InsecureSkipVerify,
	}

	con.WebIRC = fmt.Sprintf("%s discord %s %s", b.Config.WebIRCPass, hostname, ip)
}
