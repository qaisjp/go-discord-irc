package transmitter

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/qaisjp/discordgo"
)

var webhookExpiry = time.Second * 30

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
	if !t.checkLimitOK() || time.Now().After(wh.lastUse.Add(webhookExpiry)) {
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

// onWebhookUpdate responds to webhook deletions and edits and responds accordingly
// TODO: USES A NAIVE AND SLOW SOLUTION. WATCH OUT FOR UPDATES TO THIS EVENT.
func (t *Transmitter) onWebhookUpdate(s *discordgo.Session, e *discordgo.WebhooksUpdate) {
	// Important facts:
	// - INFO: we are only told the guild and the (new) channel.
	// - WHAT: we do not know if the webhook was created, deletion or edited
	// - WHO: we do not know if we ordered the creation, deletion or edit
	//
	// EVERYTHING BELOW THIS IS NAIVE AND SLOW. WHEN THIS EVENT IS UPDATED, REWRITE AND USE INFO AT BOTTOM OF FUNC

	// Naive solution:
	// - Ask Discord for all the webhooks
	// - Compare our state (of webhooks) with Discord's copy (using ID as pivot)
	// - Remove any webhooks (on both sides) where the channel is not as expected.
	// - Remove any webhooks (on our side) if it's not in their list.
	webhooks, err := t.session.GuildWebhooks(t.guild)
	if err != nil {
		logrus.Warnln(errors.Wrap(err, "cannot get guild webhooks in response to edit"))
		return
	}

OurWebhooks:
	for _, ours := range t.webhooks.list {
		oursExists := false

		for _, theirs := range webhooks {
			// Don't check for inconsistencies if our their webhook is not also our webhook
			if theirs.ID != ours.ID {
				continue
			}

			// Since the ID is the same, we can confirm they still have our webhook!
			oursExists = true

			// The webhook is consistent if the ChannelID matches.
			// If it matches, then we continue the outerloop
			if theirs.ChannelID == ours.ChannelID {
				continue OurWebhooks
			}

			// Remove the webhook from our side
			t.webhooks.Remove(ours.ChannelID)

			// Remove the webhook from Discord, thus triggering this event yet again.
			// We can't use an ignore flag here as we'll inevitably run into a race condition.
			t.session.WebhookDelete(theirs.ID)

			// Continue the OUTER loop as Discord will not have another webhook with the same ID
			continue OurWebhooks
		}

		// Delete our webhook if they don't have the webhook
		if !oursExists {
			t.webhooks.Remove(ours.ChannelID)
		}
	}

	// EVERYTHING BELOW THIS IS WRONG. COME BACK AND USE THIS INFORMATION WHEN THE EVENT IS UPDATED.

	// Syncing webhook deletions:
	// - Ask discord if the webhook at that ID exists <<<CANT DO THIS, NO ID>>>
	// - If the webhook does not exist, then we know it was deleted.
	// 	- If the webhook was DELETED, sync the deletion on our side, and return.
	// - If the webhook was NOT DELETED, it could have been a creation or edit.

	// Syncing webhook edits:
	// - We now know it was an edit or creation. We know the new channel and id.
	// ...
}
