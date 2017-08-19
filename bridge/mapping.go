package bridge

import "github.com/bwmarrin/discordgo"

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
