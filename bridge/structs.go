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
	Nick          string // still non-unique
	Bot           bool   // are they a bot?
	Online        bool
}

var quitMessages = []string{
	`We're no strangers to love`,
	`You know the rules and so do I`,
	`A full commitment's what I'm thinking of`,
	`You wouldn't get this from any other guy`,

	`I just want to tell you how I'm feeling`,
	`Gotta make you understand`,

	`Never gonna give you up, never gonna let you down`,
	`Never gonna run around and desert you`,
	`Never gonna make you cry, never gonna say goodbye`,
	`Never gonna tell a lie and hurt you`,

	`We've known each other for so long`,
	`Your heart's been aching but you're too shy to say it`,
	`Inside we both know what's been going on`,
	`We know the game and we're gonna play it`,

	`And if you ask me how I'm feeling`,
	`Don't tell me you're too blind to see`,

	`Never gonna give you up, never gonna let you down`,
	`Never gonna run around and desert you`,
	`Never gonna make you cry, never gonna say goodbye`,
	`Never gonna tell a lie and hurt you`,

	`Never gonna give you up, never gonna let you down`,
	`Never gonna run around and desert you`,
	`Never gonna make you cry, never gonna say goodbye`,
	`Never gonna tell a lie and hurt you`,

	`We've known each other for so long`,
	`Your heart's been aching but you're too shy to say it`,
	`Inside we both know what's been going on`,
	`We know the game and we're gonna play it`,

	`I just want to tell you how I'm feeling`,
	`Gotta make you understand`,

	`Never gonna give you up, never gonna let you down`,
	`Never gonna run around and desert you`,
	`Never gonna make you cry, never gonna say goodbye`,
	`Never gonna tell a lie and hurt you`,
}
