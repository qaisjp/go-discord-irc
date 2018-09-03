package bridge

import (
	"github.com/qaisjp/discordgo"
)

// DiscordMessage is a chat message sent to IRC (from Discord)
type DiscordMessage struct {
	*discordgo.Message
	Content  string
	IsAction bool
}

// IRCMessage is a chat message sent to Discord (from IRCListener)
type IRCMessage struct {
	IRCChannel string
	Username   string
	Message    string
	IsAction   bool
}

// DiscordUser is information that IRC needs to know about a user
type DiscordUser struct {
	ID            string // globally unique id
	Username      string
	Discriminator string
	Nick          string // still non-unique
	Bot           bool   // are they a bot?
	Online        bool
}

// Mapping is a mapping between a Discord channel and an IRC channel (essentially a tuple).
type Mapping struct {
	DiscordChannel string
	IRCChannel     string
}
