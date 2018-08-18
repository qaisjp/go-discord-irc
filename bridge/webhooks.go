package bridge

import (
	"encoding/json"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// A WebhookDemuxer automatically keeps track of the webhooks
// being used. If webhooks are taking too long to modify, it will
// automatically create a new one to speed up the process.
//
// It also detects removed webhooks and removes them from the system,
// guaranteeing message delivery. Messages may not be in the right order.
//
// WebhookDemuxer does not need to keep track of all the currently bridged
// channels. It only needs to know the target channel as it is being used.
//
// TODO: It does not currently delete webhooks. It may infinitely create them.
//		 Removals need to take into consideration whether all other hooks are
//		 expired, and maybe even how frequently hooks are being created.
type WebhookDemuxer struct {
	Discord  *discordBot
	webhooks []*Webhook
}

// NewWebhookDemuxer creates a new WebhookDemuxer
func NewWebhookDemuxer(bot *discordBot) *WebhookDemuxer {
	return &WebhookDemuxer{
		Discord:  bot,
		webhooks: make([]*Webhook, 0, 2),
	}
}

// Execute executes a webhook, keeping track of the username provided in WebhookParams.
func (x *WebhookDemuxer) Execute(channelID string, data *discordgo.WebhookParams) (err error) {
Retry:
	// Remove any niled items
	a := x.webhooks
	for i := 0; i < len(a); i++ {
		if a[i] == nil {
			a = append(a[:i], a[i+1:]...)
			i-- // Since we just deleted a[i], we must redo that index
		}
	}
	x.webhooks = a

	// The webhook to use
	var chosenWebhook **Webhook

	// First find any existing webhooks targeting this channel
	channelWebhooks := make([]**Webhook, 0, 2)
	for i, webhook := range x.webhooks {
		if webhook.ChannelID == channelID {
			channelWebhooks = append(channelWebhooks, &x.webhooks[i])
		}
	}

	if len(channelWebhooks) > 1 {
		log.WithField("webhooks", channelWebhooks).Warn("Investigate: multiple webhooks for a channel")
	}

	// Set chosen webhook to the first channelWebhook (if exists)
	if len(channelWebhooks) > 0 {
		chosenWebhook = channelWebhooks[0]
	}

	// Make sure that webhook actually exists!
	if chosenWebhook != nil {
		_, err := x.Discord.bridge.discord.Webhook((*chosenWebhook).ID)
		if err != nil {
			*chosenWebhook = nil

			err = errors.Wrap(err, "chosen webhook does not actually exist")
			log.WithField("error", err).Warnln("Retrying webhook execution")
			goto Retry
		}

		log.WithField("params", data).Debugln("Webhook passed validity test.")
	}

	// If we still haven't found a webhook, create one.
	var newWebhook *discordgo.Webhook
	if chosenWebhook == nil {
		log.WithField("params", data).Debugln("Creating a new webhook...")

		newWebhook, err = x.Discord.WebhookCreate(channelID, x.Discord.bridge.Config.WebhookPrefix+" IRC", "")
		if err != nil {
			// An error could occur here if we run out of remaining webhooks to use.
			log.WithField("params", data).Errorln("Could not create webhook. Attempting to use fallback webhooks.", err)

			// Do you we have any existing webhooks?
			if len(x.webhooks) == 0 {
				return errors.Wrap(err, "webhook creation failed, and no fallback webhooks")
			}

			// Lets try to reuse an existing webhook
			wh, err := x.webhooks[0].ModifyChannel(x, channelID)
			if err != nil {
				return errors.Wrap(err, "webhook creation failed, and could not modify webhook")
			}

			x.webhooks[0] = wh
			chosenWebhook = &wh

			log.WithField("params", data).Debugln("Fallback webhook successfully used.")
		}
	}

	// If we have created a new webhook
	if newWebhook != nil {
		// Create demux compatible webhook
		wh := &Webhook{
			Webhook: newWebhook,
			// Username and Expired fields set later
		}
		chosenWebhook = &wh

		// Add the newly created demux compatible webhook to our pool
		x.webhooks = append(x.webhooks, wh)

		log.WithField("params", data).Debugln("New webhook successfully created.")
	}

	webhook := *chosenWebhook

	// Reset the expiry ticket for the webhook
	webhook.ResetExpiry()

	// Update the webook username field
	webhook.Username = data.Username

	log.WithField("params", data).Debugln("Executing webhook now...")

	// TODO: What if it takes a long time? See wait=true below.
	err = x.Discord.WebhookExecute(webhook.ID, webhook.Token, true, data)
	if err != nil {
		webhook.Username = ""
		webhook.User = nil

		log.WithField("error", err).WithField("params", data).Errorln("Could not execute webhook,")
		x.Discord.bridge.ircListener.Privmsg("qaisjp", "Check error log! "+err.Error())
	}

	log.WithField("params", data).Debugln("Webhook successfully executed.")

	return nil
}

// WebhookEdit updates an existing Webhook.
// This method is a copy of discordgo.WebhookEdit, but with added support for channelID.
// See github.com/bwmarrin/discordgo/issues/434.
//
// webhookID: The ID of a webhook.
// name     : The name of the webhook.
// avatar   : The avatar of the webhook.
func (x *WebhookDemuxer) WebhookEdit(webhookID, name, avatar, channelID string) (st *discordgo.Role, err error) {
	data := struct {
		Name      string `json:"name,omitempty"`
		Avatar    string `json:"avatar,omitempty"`
		ChannelID string `json:"channel_id,omitempty"`
	}{name, avatar, channelID}

	body, err := x.Discord.RequestWithBucketID("PATCH", discordgo.EndpointWebhook(webhookID), data, discordgo.EndpointWebhooks)
	if err != nil {
		return
	}

	// err = unmarshal(body, &st)
	// (above statement pseudo-transcluded below)
	err = json.Unmarshal(body, &st)
	if err != nil {
		return nil, discordgo.ErrJSONUnmarshal
	}

	return
}

// ContainsWebhook checks whether the pool contains the given webhookID
func (x *WebhookDemuxer) ContainsWebhook(webhookID string) (contains bool) {
	for _, webhook := range x.webhooks {
		if webhook.ID == webhookID {
			contains = true
			break
		}
	}

	return
}

// Destroy destroys the webhook demultiplexer
func (x *WebhookDemuxer) Destroy() {
	log.Println("Destroying WebhookDemuxer...")
	// Delete all the webhooks
	if len(x.webhooks) > 0 {
		log.Println("- Removing hooks...")
		for _, webhook := range x.webhooks {
			err := x.Discord.WebhookDelete(webhook.ID)
			if err != nil {
				log.Printf("-- Could not remove hook %s: %s", webhook.ID, err.Error())
			}
		}
		log.Println("- Hooks removed!")
	}
	log.Println("...WebhookDemuxer destroyed!")
}

// Webhook is a wrapper around discordgo.Webhook,
// tracking whether it was the last webhook that spoke in a channel.
// This struct is only necessary for the swapping functionality that
// works around the Android/Web bug.
type Webhook struct {
	*discordgo.Webhook
	Username string
	LastUse  time.Time
}

// ResetExpiry resets the expiry of the webhook
func (w *Webhook) ResetExpiry() {
	w.LastUse = time.Now()
}

// ModifyChannel changes the channel of a webhook
func (w *Webhook) ModifyChannel(x *WebhookDemuxer, channelID string) (*Webhook, error) {
	_, err := x.WebhookEdit(
		w.ID,
		"", "",
		channelID,
	)

	if err != nil {
		// Ah crap, so we couldn't edit the webhook to be for that channel.
		// Let's reset our chosen webhook and resort to creating a new one.
		return nil, errors.Wrap(err, "Could not modify webhook channel")
	}

	// Our library doesn't track this for us,
	// so lets update the channel ID.
	w.ChannelID = channelID
	return w, nil
}
