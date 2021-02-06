// Package transmitter provides functionality for transmitting
// arbitrary webhook messages to Discord.
//
// The package provides the following functionality:
// - Creating new webhooks, whenever necessary
// - Loading webhooks that we have previously created
// - Sending new messages
// - Editing messages, via message ID
// - Deleting messages, via message ID
//
// The package has been designed for matterbridge, but with other
// Go bots in mind. The public API should be matterbridge-agnostic.
package transmitter

import (
	"fmt"

	"github.com/matterbridge/discordgo"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// A Transmitter represents a message manager for a single guild.
type Transmitter struct {
	session    *discordgo.Session
	guild      string
	title      string
	autoCreate bool

	// channelWebhooks maps from a channel ID to a webhook instance
	channelWebhooks map[string]*discordgo.Webhook
}

// ErrWebhookNotFound is returned when a valid webhook for this channel/message combination does not exist
var ErrWebhookNotFound = errors.New("webhook for this channel and message does not exist")

// New returns a new Transmitter given a Discord session, guild ID, and title.
func New(session *discordgo.Session, guild string, title string, autoCreate bool) (*Transmitter, error) {
	channelWebhooks := make(map[string]*discordgo.Webhook)

	// Get all existing webhooks
	hooks, err := session.GuildWebhooks(guild)

	// Check to make sure we have permissions
	if err != nil {
		restErr := err.(*discordgo.RESTError)

		if restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeMissingPermissions {
			// Only propagate the error if we are in autoCreate mode
			if autoCreate {
				return nil, errors.Wrap(err, "the 'Manage Webhooks' permission is required")
			}
		} else {
			return nil, errors.Wrap(err, "could not get webhooks")
		}
	}

	// Get own user ID from state, and fallback on API request
	var botID string
	if user := session.State.User; user != nil {
		botID = user.ID
	} else {
		user, err := session.User("@me")
		if err != nil {
			return nil, errors.Wrap(err, "could not get current user")
		}
		botID = user.ID
	}

	// Pick up existing webhooks with the same name, created by us
	// This is still used when autoCreate is disabled
	for _, wh := range hooks {
		// If there are multiple webhooks, it will just take
		chosen := wh.ApplicationID == botID
		if chosen {
			channelWebhooks[wh.ChannelID] = wh
			log.WithFields(log.Fields{
				"id":      wh.ID,
				"name":    wh.Name,
				"channel": wh.ChannelID,
			}).Println("Picking up webhook")
		}
	}

	t := &Transmitter{
		session:    session,
		guild:      guild,
		title:      title,
		autoCreate: autoCreate,

		channelWebhooks: channelWebhooks,
	}

	return t, nil
}

// Message transmits a message to the given channel with the provided webhook data.
//
// Note that this function will wait until Discord responds with an answer.
func (t *Transmitter) Message(channelID string, params *discordgo.WebhookParams) (msg *discordgo.Message, err error) {
	wh := t.getWebhook(channelID)

	// If we don't have a webhook for this channel...
	if wh == nil {
		// Early exit, if we don't want to automatically create one
		if !t.autoCreate {
			return nil, ErrWebhookNotFound
		}

		// Try and create one
		fmt.Printf("Creating a webhook for %s\n", channelID)
		wh, err = t.createWebhook(channelID)
		if err != nil {
			return
		}
	}

	msg, err = t.session.WebhookExecute(wh.ID, wh.Token, true, params)
	if err != nil {
		err = errors.Wrap(err, "could not execute existing webhook")
		return nil, err
	}

	return msg, nil
}

// Edit will edit a message in a channel, if possible.
func (t *Transmitter) Edit(channelID string, messageID string, params *discordgo.WebhookParams) (err error) {
	wh := t.getWebhook(channelID)

	if wh == nil {
		err = ErrWebhookNotFound
		return
	}

	uri := discordgo.EndpointWebhookToken(wh.ID, wh.Token) + "/messages/" + messageID
	_, err = t.session.RequestWithBucketID("PATCH", uri, params, discordgo.EndpointWebhookToken("", ""))
	if err != nil {
		return
	}

	return
}

// HasWebhook checks whether the transmitter is using a particular webhook.
func (t *Transmitter) HasWebhook(id string) bool {
	for _, wh := range t.channelWebhooks {
		if wh.ID == id {
			return true
		}
	}

	return false
}

// AddWebhook allows you to register a channel's webhook with the transmitter.
func (t *Transmitter) AddWebhook(channelID string, webhook *discordgo.Webhook) (replaced bool) {
	_, replaced = t.channelWebhooks[channelID]
	t.channelWebhooks[channelID] = webhook
	return
}
