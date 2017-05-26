package bridge

import (
	"errors"
	"fmt"
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
	ircServerAddress string
	ircPrimaryName   string

	chanMapToIRC     map[string]string
	chanMapToDiscord map[string]string
	chanIRC          []string
	chanDiscord      []string

	h *home
}

func (b *Bridge) Close() {
	close(b.h.done)
}

// TODO: Use errors package
func (b *Bridge) load(opts Options) bool {
	if opts.IRCServer == "" {
		fmt.Println("Missing server name.")
		return false
	}

	b.ircServerAddress = opts.IRCServer
	b.ircPrimaryName = opts.IRCPrimaryName

	b.chanMapToIRC = opts.ChannelMappings

	ircChannels := make([]string, len(b.chanMapToIRC))
	discordChannels := make([]string, len(b.chanMapToIRC))

	i := 0
	for discord, irc := range opts.ChannelMappings {
		ircChannels[i] = irc
		discordChannels[i] = discord
		i += 1
	}

	chanMapToDiscord := make(map[string]string)
	for k, v := range b.chanMapToIRC {
		chanMapToDiscord[v] = k
	}
	b.chanMapToDiscord = chanMapToDiscord

	b.chanIRC = ircChannels
	b.chanDiscord = discordChannels

	return true
}

func New(opts Options) (*Bridge, error) {
	dib := &Bridge{}
	if !dib.load(opts) {
		return nil, errors.New("Options error. TODO: More info here!")
	}

	discord, err := prepareDiscord(dib, opts.DiscordBotToken)
	ircPrimary := prepareIRCPrimary(dib)
	ircManager := prepareIRCManager(opts.IRCServer)

	if err != nil {
		return nil, err
	}

	prepareHome(dib, discord, ircPrimary, ircManager)

	discord.h = dib.h
	ircPrimary.h = dib.h
	ircManager.h = dib.h

	return dib, nil
}

func (b *Bridge) Open() (err error) {
	// Open a websocket connection to Discord and begin listening.
	err = b.h.discord.Open()
	if err != nil {
		fmt.Println("error opening discord connection,", err)
		return err
	}

	err = b.h.ircPrimary.Connect(b.ircServerAddress)
	if err != nil {
		fmt.Println("error opening irc connection,", err)
		return err
	}

	go b.h.ircPrimary.Loop()

	return
}

func testingChannels(id string) bool {
	// inf1, bottest
	return /*(id == "315278744572919809") ||*/ (id == "316038111811600387")
}
