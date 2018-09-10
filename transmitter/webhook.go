package transmitter

import (
	"time"

	"github.com/pkg/errors"
	"github.com/qaisjp/discordgo"
)

type wrappedWebhook struct {
	*discordgo.Webhook
	lastUse time.Time
}

type webhook *wrappedWebhook

// executeWebhook executes a webhook for a specific channel, if it exists
func (t *Transmitter) executeWebhook(channel string, params *discordgo.WebhookParams) error {
	wh := t.webhooks.Get(channel)
	if wh == nil {
		return errors.New("webhook does not exist")
	}

	// Update the webhook's last use
	// and subsequently fix the heap
	wh.lastUse = time.Now()
	t.webhooks.Fix(channel)

	return t.session.WebhookExecute(wh.ID, wh.Token, true, params)
}

// getWebhook attempts to return a webhook for the channel, or
// repurposes an existing webhook to be used with that channel.
//
// An error will be returned if webhook repurposing failed.
//
// If no webhook is available, the webhook returned will be nil.
//
// TODO: Check if this function breaks when you remove webhooks
// TODO: Remove webhooks from heap automatically using the event
func (t *Transmitter) getWebhook(channel string) (webhook, error) {
	// Just stop if there are no webhooks
	if t.webhooks.Len() == 0 {
		return nil, nil
	}

	// Try and get a webhook that matches that channel
	if wh := t.webhooks.Get(channel); wh != nil {
		return wh, nil
	}

	// Peek at the heap pop
	wh := t.webhooks.Peak()

	// And repurpose if limit met OR is expired
	if !t.checkLimitOK() || time.Now().After(wh.lastUse.Add(time.Second*5)) {
		_, err := t.session.WebhookEdit(wh.ID, "", "", channel)
		if err == nil {
			// Webhooks don't maintain their own state, so we rely
			// on the old ChannelID here, and we update it later.
			t.webhooks.SwapChannel(wh.ChannelID, channel)
			wh.ChannelID = channel
		}
		return wh, errors.Wrap(err, "could not repurpose webhook")
	}

	return nil, nil
}

// createWebhook creates a webhook for a specific channel.
func (t *Transmitter) createWebhook(channel string) (webhook, error) {
	if !t.checkLimitOK() {
		panic(errors.New("webhook limit has been reached"))
	}

	wh, err := t.session.WebhookCreate(channel, t.prefix+time.Now().Format(" 3:04:05PM"), "")

	if err != nil {
		return nil, errors.Wrap(err, "could not create webhook")
	}

	wrapped := &wrappedWebhook{wh, time.Time{}}
	t.webhooks.Push(wrapped)

	return wrapped, nil
}

// checkAndDeleteWebhook checks to see if the webhook exists, and will delete accordingly.
//
// If the transmitter does not know about the webhook, false is returned.
// If the transmitter does know about the webhook:
// 		- false is returned if Discord doesn't know.
//		- true is returned if Discord does know it exists
// If Discord returns an error, this function will return an error for the second argument.
func (t *Transmitter) checkAndDeleteWebhook(channel string) (bool, error) {
	wh := t.webhooks.Get(channel)

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
			t.webhooks.Remove(channel)
			return false, nil
		}

		return false, err
	}
	return true, nil
}

// checkLimitOK returns true if the webhook limit has not been reached
func (t *Transmitter) checkLimitOK() bool {
	return t.webhooks.Len() < t.limit
}
