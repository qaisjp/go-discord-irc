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
