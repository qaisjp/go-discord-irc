package transmitter

import (
	"time"

	"github.com/matterbridge/discordgo"
	"github.com/pkg/errors"
)

// createWebhook creates a webhook for a specific channel.
func (t *Transmitter) createWebhook(channel string) (*discordgo.Webhook, error) {
	wh, err := t.session.WebhookCreate(channel, t.title+time.Now().Format(" 3:04:05PM"), "")

	if err != nil {
		return nil, errors.Wrap(err, "could not create webhook")
	}

	t.channelWebhooks[channel] = wh
	return wh, nil
}

func (t *Transmitter) getWebhook(channel string) *discordgo.Webhook {
	return t.channelWebhooks[channel]
}
