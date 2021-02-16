package bridge

import (
	"strings"

	"github.com/matterbridge/discordgo"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (d *discordBot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	d.publishMessage(s, m.Message, false)
}

func (d *discordBot) onMessageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) {
	d.publishMessage(s, m.Message, true)
}

func (d *discordBot) OnMessageReactionAdd(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	d.publishReaction(s, m.MessageReaction)
}

// onMemberListChunk is fired in response to our GuildMembers request in OnReady
func (d *discordBot) onMemberListChunk(s *discordgo.Session, m *discordgo.GuildMembersChunk) {
	for _, m := range m.Members {
		d.handleMemberUpdate(m, false)
	}
}

func (d *discordBot) onMemberUpdate(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	d.handleMemberUpdate(m.Member, false)
}

// onMemberLeave is triggered when a user is removed from a guild (leave/kick/ban).
func (d *discordBot) onMemberLeave(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	d.bridge.removeUserChan <- m.User.ID
}

// What does this do? Probably what it sounds like.
func (d *discordBot) OnPresencesReplace(s *discordgo.Session, m *discordgo.PresencesReplace) {
	for _, p := range *m {
		d.handlePresenceUpdate(p.User.ID, p.Status, false)
	}
}

// Handle when presence is updated
func (d *discordBot) OnPresenceUpdate(s *discordgo.Session, m *discordgo.PresenceUpdate) {
	d.handlePresenceUpdate(m.Presence.User.ID, m.Presence.Status, false)
}

func (d *discordBot) OnTypingStart(s *discordgo.Session, m *discordgo.TypingStart) {
	status := discordgo.StatusOffline

	p, err := d.State.Presence(d.guildID, m.UserID)
	if err != nil {
		log.Println(errors.Wrap(err, "get presence from in OnTypingStart failed"))
		// return
	} else {
		status = p.Status
	}

	// .. and handle as per usual
	d.handlePresenceUpdate(m.UserID, status, true)
}

func (d *discordBot) OnReady(s *discordgo.Session, m *discordgo.Ready) {
	// Fires a GuildMembersChunk event
	err := d.RequestGuildMembers(d.guildID, "", 0, true)
	if err != nil {
		log.Warningln(errors.Wrap(err, "could not request guild members").Error())
		return
	}

	emoji, err := d.GuildEmojis(d.guildID)
	if err == nil {
		d.setGuildEmoji(d.guildID, emoji)
	}
}

func (d *discordBot) onGuildEmojiUpdate(s *discordgo.Session, m *discordgo.GuildEmojisUpdate) {
	d.setGuildEmoji(m.GuildID, m.Emojis)
}

func (d *discordBot) setGuildEmoji(guild string, emoji []*discordgo.Emoji) {
	d.bridge.emoji = make(map[string]*discordgo.Emoji)
	for _, e := range emoji {
		d.bridge.emoji[strings.ToLower(e.Name)] = e
	}
}

func (d *discordBot) handleMemberUpdate(m *discordgo.Member, forceOnline bool) {
	status := discordgo.StatusOnline

	if !forceOnline {
		presence, err := d.State.Presence(d.guildID, m.User.ID)
		if err != nil {
			// This error is usually triggered on first run because it represents offline
			if err != discordgo.ErrStateNotFound {
				log.WithField("error", err).Errorln("presence retrieval failed")
			}
			return
		}

		if !isStatusOnline(presence.Status) {
			return
		}

		status = presence.Status
	}

	d.sendUpdateUserChan(DiscordUser{
		ID:            m.User.ID,
		Username:      m.User.Username,
		Discriminator: m.User.Discriminator,
		Nick:          GetMemberNick(m),
		Bot:           m.User.Bot,
		Online:        isStatusOnline(status),
	})
}
