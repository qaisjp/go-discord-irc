package webhooks

import (
	"strings"

	"github.com/hashicorp/go-multierror"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

type webhook *discordgo.Webhook

// Package webhooks provides functionality for displaying arbitrary
// webhook messages on Discord.
//
// Existing webhooks are used for messages sent, and if necessary,
// new webhooks are created to ensure messages in multiple popular channels
// don't cause messages to be registered as new users.

// A Transmitter represents a message manager instance for a single guild.
type Transmitter struct {
	session *discordgo.Session
	guild   string
	prefix  string

	webhooks map[string]webhook
}

// NewTransmitter returns a new Transmitter given a Discord session, guild ID, and webhook prefix.
func NewTransmitter(session *discordgo.Session, guild string, prefix string) (*Transmitter, error) {
	// Get all existing webhooks
	hooks, err := session.GuildWebhooks(guild)

	// Check to make sure we have permissions
	if err != nil {
		restErr := err.(*discordgo.RESTError)
		if restErr.Message != nil && restErr.Message.Code == 50013 {
			return nil, errors.Wrap(err, "the 'Manage Webhooks' permission is required")
		}

		return nil, errors.Wrap(err, "could not get webhooks")
	}

	// Delete existing webhooks with the same prefix
	for _, wh := range hooks {
		if strings.HasPrefix(wh.Name, prefix) {
			if err := session.WebhookDelete(wh.ID); err != nil {
				return nil, errors.Errorf("Could not delete webhook %s (\"%s\")", wh.ID, wh.Name)
			}
		}
	}

	return &Transmitter{
		session: session,
		guild:   guild,
		prefix:  prefix,

		webhooks: make(map[string]webhook),
	}, nil
}

// Close immediately stops all active webhook timers and deletes webhooks.
func (t *Transmitter) Close() error {
	var result error

	// Delete all the webhooks
	for _, webhook := range t.webhooks {
		err := t.session.WebhookDelete(webhook.ID)
		if err != nil {
			multierror.Append(result, errors.Wrapf(err, "could not remove hook %s", webhook.ID))
		}
	}

	return result
}

// Message transmits a message to the given channel with the given username, avatarURL, and content.
//
// Note that this function will wait until Discord responds with an answer.
func (t *Transmitter) Message(channel string, username string, avatarURL string, content string) error {
	wh := t.webhooks[channel]

	if wh == nil {
		return errors.New("no webhook available")
	}

	params := discordgo.WebhookParams{
		Username:  username,
		AvatarURL: avatarURL,
		Content:   content,
	}

	err := t.session.WebhookExecute(wh.ID, wh.Token, true, &params)
	if err != nil {
		return errors.Wrap(err, "could not execute webhook")
	}

	return nil
}
