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
// Until the Android/Web bug is fixed, it will also handle swapping between
// two webhooks per user. It will automatically time out webhooks preserved
// for a certain user on a channel.
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

	// First find any existing webhooks targeting this channel
	channelWebhooks := make([]**Webhook, 0, 2)
	for i, webhook := range x.webhooks {
		if webhook.ChannelID == channelID {
			channelWebhooks = append(channelWebhooks, &x.webhooks[i])
		}
	}

	// The webhook to use
	var chosenWebhook **Webhook

	// Find a webhook of the same username and channel.
	for _, webhook := range channelWebhooks { // searching channel webhooks
		if (*webhook).Username == data.Username {
			chosenWebhook = webhook
			break
		}
	}

	// So we don't have an expired webhook from our channel.
	// Lets use the oldest webhook. The most recently active from
	// our channel is going to be the most recent speaker.
	// Only necessary for the Android/web bug workaround.
	if (chosenWebhook == nil) && (len(channelWebhooks) > 1) {
		chosenWebhook = channelWebhooks[0]
		for _, webhook := range channelWebhooks {
			// So if the chosen webhook is born after (younger than)
			// the current webhook. The make the current webhook
			// our chosen webhook.
			if (*chosenWebhook).LastUse.After((*webhook).LastUse) {
				chosenWebhook = webhook
			}
		}
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
	}

	// If we still haven't found a webhook, create one.
	var newWebhook *discordgo.Webhook
	if chosenWebhook == nil {
		log.WithField("params", data).Debugln("Creating a new webhook...")

		newWebhook, err = x.Discord.WebhookCreate(channelID, x.Discord.bridge.Config.WebhookPrefix+" IRC", "")
		if err != nil {
			log.Errorln("Could not create webhook. Stealing expired webhook.", err)

			// We couldn't create the webhook for some reason.
			// Lets steal an expired one from somewhere...
			if len(x.webhooks) > 0 {
				modifyWebhook := x.webhooks[0]
				wh, err := modifyWebhook.ModifyChannel(x, channelID)
				if err != nil {
					return errors.Wrap(err, "Could not modify existing webhook after webhook creation failure")
				}
				chosenWebhook = &wh
			} else {
				// ... if we can. But we can't. Because there aren't any webhooks to use.
				return errors.Wrap(err, "No webhooks available to fall back on after webhook creation failure")
			}
		}
	}

	// If we have created a new webhook
	if newWebhook != nil {
		log.Debugln("Created new webhook now, so creating wrapped webhook")
		// Create demux compatible webhook
		wh := &Webhook{
			Webhook: newWebhook,
			// Username and Expired fields set later
		}
		chosenWebhook = &wh

		// Add the newly created demux compatible webhook to our pool
		x.webhooks = append(x.webhooks, wh)
	}

	webhook := *chosenWebhook

	// Reset the expiry ticket for the webhook
	webhook.ResetExpiry()

	// Update the webook username field
	webhook.Username = data.Username

	log.Debugln("--------- done, executing webhook -------")

	// TODO: What if it takes a long time? See wait=true below.
	err = x.Discord.WebhookExecute(webhook.ID, webhook.Token, true, data)
	if err != nil {
		webhook.Close()
		webhook.Username = ""
		webhook.User = nil

		log.WithField("error", err).Errorln("Could not execute webhook")
		x.Discord.bridge.ircListener.Privmsg("qaisjp", "Check error log! "+err.Error())
	}

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
		// Stop all the webhooks expiry timers.
		log.Println("- Stopping hook timers...")
		for _, webhook := range x.webhooks {
			webhook.Close()
		}

		log.Println("- Removing hooks...")
		for _, webhook := range x.webhooks {
			err := x.WebhookDelete(webhook)
			if err != nil {
				log.Printf("-- Could not remove hook %s: %s", webhook.ID, err.Error())
			}
		}
		log.Println("- Hooks removed!")
	}
	log.Println("...WebhookDemuxer destroyed!")
}

// WebhookDelete destroys the given webhook
func (x *WebhookDemuxer) WebhookDelete(w *Webhook) error {
	err := x.Discord.WebhookDelete(w.ID)

	// Workaround for library bug: github.com/bwmarrin/discordgo/issues/429
	if err != discordgo.ErrJSONUnmarshal {
		return err
	}
	return nil
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

// Close closes the webhook expiry timer
func (w *Webhook) Close() {
	// w.Close()
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
