package bridge

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/thoj/go-ircevent"
)

type Options struct {
	DiscordBotToken string
	ChannelMappings map[string]string

	IRCServer      string
	IRCUseTLS      bool
	IRCPrimaryName string // i.e, "DiscordBot", required to listen for messages in all cases
	UsePrimaryOnly bool   // set to "true" to only echo messages, instead of creating a new connection per user
}

type Bridge struct {
	opts       Options
	dg         *discordgo.Session
	ircPrimary *irc.Connection

	chanMappings map[string]string
	chanIRC      []string
	chanDiscord  []string
}

func (b *Bridge) Close() {
	b.dg.Close()
}

// TODO: Use errors package
func (b *Bridge) load(opts Options) bool {
	if opts.IRCServer == "" {
		fmt.Println("Missing server name.")
		return false
	}

	b.chanMappings = opts.ChannelMappings

	ircChannels := make([]string, len(b.chanMappings))
	discordChannels := make([]string, len(b.chanMappings))

	i := 0
	for discord, irc := range opts.ChannelMappings {
		ircChannels[i] = irc
		discordChannels[i] = discord
		i += 1
	}

	b.chanIRC = ircChannels
	b.chanDiscord = discordChannels

	return true
}

func New(opts Options) (*Bridge, error) {
	dib := &Bridge{opts: opts}
	if !dib.load(opts) {
		return nil, errors.New("Options error. TODO: More info here!")
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + opts.DiscordBotToken)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return nil, err
	}

	dib.dg = dg

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)
	dg.AddHandler(typingStart)

	ircnick := opts.IRCPrimaryName
	irccon := irc.IRC(ircnick, "BetterDiscordBot")
	// irccon.VerboseCallbackHandler = true
	irccon.Debug = true
	irccon.UseTLS = true
	irccon.TLSConfig = &tls.Config{InsecureSkipVerify: true} // TODO: Insecure TLS!

	// Welcome event
	irccon.AddCallback("001", func(e *irc.Event) {
		// Join all channels
		e.Connection.SendRaw("JOIN " + strings.Join(dib.chanIRC, ","))
	})

	// Called when received channel names... essentially OnJoinChannel
	irccon.AddCallback("366", func(e *irc.Event) { fmt.Printf("Joined IRC channel %s.", e.Arguments[1]) })

	dib.ircPrimary = irccon

	return dib, nil
}

func (b *Bridge) Open() (err error) {
	// Open a websocket connection to Discord and begin listening.
	err = b.dg.Open()
	if err != nil {
		fmt.Println("error opening discord connection,", err)
		return err
	}

	err = b.ircPrimary.Connect(b.opts.IRCServer)
	if err != nil {
		fmt.Println("error opening irc connection,", err)
		return err
	}

	go b.ircPrimary.Loop()

	return
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}

	// If the message is "pong" reply with "Ping!"
	if m.Content == "pong" {
		s.ChannelMessageSend(m.ChannelID, "Ping!")
	}
}

// This function will be called (due to AddHandler above) every time a discord user
// starts typing
func typingStart(s *discordgo.Session, m *discordgo.TypingStart) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.UserID == s.State.User.ID {
		return
	}

	if !testingChannels(m.ChannelID) {
		return
	}

	// TODO: Catch username changes, and cache UserID:Username mappings somewhere
	u, err := s.User(m.UserID)
	if err != nil {
		return
	}

	s.ChannelMessageSend(m.ChannelID, "Send global pulse for IRC user `["+u.Username+"]`")

}

func testingChannels(id string) bool {
	// inf1, bottest
	return /*(id == "315278744572919809") ||*/ (id == "316038111811600387")
}
