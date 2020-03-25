package transmitter

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

var webhookExpiry = time.Second * 30

// createWebhook creates a webhook for a specific channel.
func (t *Transmitter) createWebhook(channel string) error {
	wh, err := t.session.WebhookCreate(channel, t.prefix+time.Now().Format(" 3:04:05PM"), "")

	if err != nil {
		return errors.Wrap(err, "could not create webhook")
	}

	t.webhook = wh
	return nil
}

// checkAndDeleteWebhook checks to see if the webhook exists, and will delete accordingly.
//
// If the transmitter does not know about the webhook, false is returned.
// If the transmitter does know about the webhook:
// 		- false is returned if Discord doesn't know.
//		- true is returned if Discord does know it exists
// If Discord returns an error, this function will return an error for the second argument.
func (t *Transmitter) checkAndDeleteWebhook(channel string) (bool, error) {
	wh := t.webhook

	// If no webhook, return false
	if wh == nil {
		return false, nil
	}

	_, err := t.session.Webhook(wh.ID)
	if err != nil {
		// Check if the error is a known REST error (UnknownWebhook)
		err, ok := err.(*discordgo.RESTError)
		if ok && err.Message != nil && err.Message.Code == discordgo.ErrCodeUnknownWebhook {
			// Retry the message because the webhook is dead
			t.webhook = nil
			return false, nil
		}

		return false, err
	}
	return true, nil
}
