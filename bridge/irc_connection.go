package bridge

import (
	"fmt"
	"strings"
	"time"

	"github.com/qaisjp/go-discord-irc/irc/varys"
	irc "github.com/qaisjp/go-ircevent"
	log "github.com/sirupsen/logrus"
)

// An ircConnection should only ever communicate with its manager
// Refer to `(m *ircManager) CreateConnection` to see how these are spawned
type ircConnection struct {
	discord DiscordUser
	nick    string

	quitMessage string

	messages      chan IRCMessage
	cooldownTimer *time.Timer

	manager *IRCManager

	// channel ID for their discord channel for PMs
	pmDiscordChannel string

	// Tell users this feature is in beta
	pmNoticed        bool
	pmNoticedSenders map[string]struct{}
}

func (i *ircConnection) GetNick() string {
	nick, err := i.manager.varys.GetNick(i.discord.ID)
	if err != nil {
		panic(err.Error())
	}
	return nick
}

func (i *ircConnection) Connected() bool {
	connected, err := i.manager.varys.Connected(i.discord._ID)
	if err != nil {
		panic(err.Error())
	}
	return connected
}

func (i *ircConnection) OnWelcome(e *irc.Event) {
	// execute puppet prejoin commands
	err := i.manager.varys.SendRaw(i.discord.ID, varys.InterpolationParams{Nick: true}, i.manager.bridge.Config.IRCPuppetPrejoinCommands...)
	if err != nil {
		panic(err.Error())
	}

	i.JoinChannels()

	// just in case NickServ, Q:Lines, or otherwise force our nick to be not what we expect!
	i.manager.puppetNicks[i.GetNick()] = i

	go func(i *ircConnection) {
		for m := range i.messages {
			msg := m.Message
			if m.IsAction {
				msg = fmt.Sprintf("\001ACTION %s\001", msg)
			}
			i.Privmsg(m.IRCChannel, msg)
		}
	}(i)
}

func (i *ircConnection) JoinChannels() {
	i.SendRaw(i.manager.bridge.GetJoinCommand(i.manager.RequestChannels(i.discord.ID)))
}

func (i *ircConnection) UpdateDetails(discord DiscordUser) {
	if i.discord.Username != discord.Username {
		i.quitMessage = fmt.Sprintf("Changing real name from %s to %s", i.discord.Username, discord.Username)
		i.manager.CloseConnection(i)

		// After one second make the user reconnect.
		// This should be enough time for the nick tracker to update.
		time.AfterFunc(time.Second, func() {
			i.manager.HandleUser(discord)
		})
		return
	}

	// if their details haven't changed, don't do anything
	if (i.discord.Nick == discord.Nick) && (i.discord.Discriminator == discord.Discriminator) {
		return
	}

	i.discord = discord
	delete(i.manager.puppetNicks, i.nick)
	i.nick = i.manager.generateNickname(i.discord)
	i.manager.puppetNicks[i.nick] = i

	if err := i.manager.varys.Nick(i.discord.ID, i.nick); err != nil {
		panic(err.Error())
	}
}

func (i *ircConnection) introducePM(nick string) {
	d := i.manager.bridge.discord

	if i.pmDiscordChannel == "" {
		c, err := d.UserChannelCreate(i.discord.ID)
		if err != nil {
			// todo: sentry
			log.Warnln("Could not create private message room", i.discord, err)
			return
		}
		i.pmDiscordChannel = c.ID
	}

	if !i.pmNoticed {
		i.pmNoticed = true
		_, err := d.ChannelMessageSend(
			i.pmDiscordChannel,
			fmt.Sprintf("To reply type: `%s@%s, your message here`", nick, i.manager.bridge.Config.Discriminator))
		if err != nil {
			log.Warnln("Could not send pmNotice", i.discord, err)
			return
		}
	}

	nick = strings.ToLower(nick)
	if _, ok := i.pmNoticedSenders[nick]; !ok {
		i.pmNoticedSenders[nick] = struct{}{}
	}
}

func (i *ircConnection) OnPrivateMessage(e *irc.Event) {
	// Ignored hostmasks
	if i.manager.isIgnoredHostmask(e.Source) {
		return
	}

	// Alert private messages
	if string(e.Arguments[0][0]) != "#" {
		if e.Message() == "help" {
			i.Privmsg(e.Nick, "Commands: help, who")
		} else if e.Message() == "who" {
			i.Privmsg(e.Nick, fmt.Sprintf("I am: %s#%s with ID %s", i.discord.Nick, i.discord.Discriminator, i.discord.ID))
		}

		d := i.manager.bridge.discord

		i.introducePM(e.Nick)

		msg := fmt.Sprintf(
			"%s,%s - %s@%s: %s", e.Connection.Server, e.Source,
			e.Nick, i.manager.bridge.Config.Discriminator, e.Message())
		_, err := d.ChannelMessageSend(i.pmDiscordChannel, msg)
		if err != nil {
			log.Warnln("Could not send PM", i.discord, err)
			return
		}
		return
	}

	// GTANet does not support deafness so the below logmsg has been disabled
	// log.Println("Non listener IRC connection received PRIVMSG from channel. Something went wrong.")
}

func (i *ircConnection) SendRaw(message string) {
	if err := i.manager.varys.SendRaw(i.discord.ID, varys.InterpolationParams{}, message); err != nil {
		panic(err.Error())
	}
}

func (i *ircConnection) SetAway(status string) {
	i.SendRaw(fmt.Sprintf("AWAY :%s", status))
}

func (i *ircConnection) Privmsg(target, message string) {
	i.SendRaw(fmt.Sprintf("PRIVMSG %s :%s\r\n", target, message))
}
