package bridge

// DiscordMessageEvent is a chat message sent to IRC (from Discord)
type DiscordMessageEvent struct {
	channelID string
	userID    string
	message   string
}

// DiscordNewMessage is a chat message sent to Discord (from IRCListener)
type DiscordNewMessage struct {
	IRCChannel string
	Username   string
	Message    string
}

// DiscordUser is information that IRC needs to know about a user
type DiscordUser struct {
	ID            string // globally unique id
	Discriminator string // locally unique ID
	Nick          string // still non-unique
	Bot           bool   // are they a bot?
	Online        bool
}
