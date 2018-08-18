package transmitter

import (
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

type webhook *discordgo.Webhook

// getWebhook attempts to return a webhook for the channel, or
// repurposes an existing webhook to be used with that channel.
//
// An error will be returned if webhook repurposing failed
//
// If no webhook is available, the webhook returned will be nil.
func (t *Transmitter) getWebhook(channel string) (webhook, error) {
	if wh := t.webhooks[channel]; wh != nil {
		return wh, nil
	}

	// errors.Wrap(err, "failed to repurpose webhook")

	return nil, nil
}

// createWebhook creates a webhook for a specific channel.
func (t *Transmitter) createWebhook(channel string) (webhook, error) {
	wh, err := t.session.WebhookCreate(channel, t.prefix, "")

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
