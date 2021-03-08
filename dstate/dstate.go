// Package dstate provides helpers for discordgo that first tries the State, and then falls back on an endpoint request.
package dstate

import "github.com/matterbridge/discordgo"

var nilState = discordgo.ErrNilState

func ChannelMessage(s *discordgo.Session, channelID string, messageID string) (*discordgo.Message, error) {
	if msg, err := s.State.Message(channelID, messageID); err == nil {
		return msg, err
	}

	return s.ChannelMessage(channelID, messageID)
}
