// Package transmitter provides functionality for transmitting
// arbitrary webhook messages on Discord.
//
// Existing webhooks are used for messages sent, and if necessary,
// new webhooks are created to ensure messages in multiple popular channels
// don't cause messages to be registered as new users.
package transmitter

import (
	"strings"

	"github.com/hashicorp/go-multierror"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

// A Transmitter represents a message manager instance for a single guild.
type Transmitter struct {
	session *discordgo.Session
	guild   string
	prefix  string
	limit   int // max number of webhooks

	webhooks webhookHeap
}

// New returns a new Transmitter given a Discord session, guild ID, and webhook prefix.
func New(session *discordgo.Session, guild string, prefix string, limit int) (*Transmitter, error) {
	// Get all existing webhooks
	hooks, err := session.GuildWebhooks(guild)

	// Check to make sure we have permissions
	if err != nil {
		restErr := err.(*discordgo.RESTError)
		if restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeMissingPermissions {
			return nil, errors.Wrap(err, "the 'Manage Webhooks' permission is required")
		}

		return nil, errors.Wrap(err, "could not get webhooks")
	}

	// Delete existing webhooks with the same prefix
	for _, wh := range hooks {
		if strings.HasPrefix(wh.Name, prefix) {
			if err := session.WebhookDelete(wh.ID); err != nil {
				return nil, errors.Wrapf(err, "could not remove hook %s", wh.ID)
			}
		}
	}

	return &Transmitter{
		session: session,
		guild:   guild,
		prefix:  prefix,
		limit:   limit,

		webhooks: newWebhookHeap(),
	}, nil
}

// Close immediately stops all active webhook timers and deletes webhooks.
func (t *Transmitter) Close() error {
	var result error

	// Delete all the webhooks
	for _, wh := range t.webhooks.list {
		err := t.session.WebhookDelete(wh.ID)
		if err != nil {
			result = multierror.Append(result, errors.Wrapf(err, "could not remove hook %s", wh.ID)).ErrorOrNil()
		}
	}

	return result
}

// Message transmits a message to the given channel with the given username, avatarURL, and content.
//
// Note that this function will wait until Discord responds with an answer.
//
// This will use an existing webhook if it exists.
// If an existing webhook doesn't exist then it will try to repurpose a webhook.
// If there is space to create a new webhook then it will do that.
func (t *Transmitter) Message(channel string, username string, avatarURL string, content string) (err error) {
	wh, err := t.getWebhook(channel)
	if err != nil {
		return err
	}

	// webhook will be nil if there was none to repurpose
	if wh == nil {
		wh, err = t.createWebhook(channel)
		if err != nil {
			return err // this error is already wrapped by us
		}
	}

	params := discordgo.WebhookParams{
		Username:  username,
		AvatarURL: avatarURL,
		Content:   content,
	}

	err = t.executeWebhook(channel, &params)
	if err != nil {
		exists, checkErr := t.checkAndDeleteWebhook(channel)

		// If there was error performing the check, compose the list
		if checkErr != nil {
			err = multierror.Append(err, checkErr).ErrorOrNil()
		}

		// If the webhook exists OR there was an error performing the check
		// return the error to the caller
		if exists || checkErr != nil {
			return errors.Wrap(err, "could not execute existing webhook")
		}

		// Otherwise just try and send the message again
		return t.Message(channel, username, avatarURL, content)
	}

	return nil
}

// HasWebhook checks whether the transmitter is using a particular webhook
func (t *Transmitter) HasWebhook(id string) bool {
	for _, wh := range t.webhooks.list {
		if wh.ID == id {
			return true
		}
	}

	return false
}
