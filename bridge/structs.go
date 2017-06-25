package bridge

// DiscordMessageEvent is a chat message sent to IRC (from Discord)
type DiscordMessageEvent struct {
	channelID string
	userID    string
	message   string
}

// DiscordNewMessage is a chat message sent to Discord (from IRCListener)
type DiscordNewMessage struct {
	ircChannel string
	str        string
}

// DiscordUser is information that IRC needs to know about a user
type DiscordUser struct {
	ID            string // globally unique id
	Discriminator string // locally unique ID
	Username      string // non-unique nickname
	Nick          string // still non-unique
	Bot           bool   // are they a bot?
}
