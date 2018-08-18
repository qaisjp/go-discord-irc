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
		if restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeMissingPermissions {
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
func (t *Transmitter) Message(channel string, username string, avatarURL string, content string) (err error) {
	wh := t.webhooks[channel]

	if wh == nil {
		// todo: repurpose a webhook if there is an out of date one

		wh, err = t.createWebhook(channel)
		if err != nil {
			return errors.Wrap(err, "could not create webhook")
		}

		// todo: if we can't create a webhook, we want to repurpose a webhook
	}

	if wh == nil {
		return errors.New("no webhook available")
	}

	params := discordgo.WebhookParams{
		Username:  username,
		AvatarURL: avatarURL,
		Content:   content,
	}

	err = t.session.WebhookExecute(wh.ID, wh.Token, true, &params)
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

// createWebhook creates a webhook for a specific channel.
func (t *Transmitter) createWebhook(channel string) (webhook, error) {
	wh, err := t.session.WebhookCreate(channel, t.prefix+" IRC", "")

	if err != nil {
		return nil, errors.Wrap(err, "could not create webhook")
	}

	t.webhooks[channel] = wh

	return wh, nil
}

// checkAndDeleteWebhook checks to see if the webhook exists, and will delete accordingly.
//
// If the transmitter does not know about the webhook, false is returned.
// If the transmitter does know about the webhook:
// 		- false is returned if Discord doesn't know.
//		- true is returned if Discord does know it exists
// If Discord returns an error, this function will return an error for the second argument.
func (t *Transmitter) checkAndDeleteWebhook(channel string) (bool, error) {
	wh := t.webhooks[channel]

	// If no webhook, return false
	if wh == nil {
		return false, nil
	}

	_, err := t.session.Webhook(wh.ID)
	if err != nil {
		// Check if the error is a known REST error (UnknownWebhook)
		err, ok := err.(*discordgo.RESTError)
		if ok && err.Message != nil && err.Message.Code == 10015 { // todo: in next discordgo version use discordgo.ErrCodeUnknownWebhook
			// Retry the message because the webhook is dead
			delete(t.webhooks, channel)
			return false, nil
		}

		return false, err
	}
	return true, nil
}
