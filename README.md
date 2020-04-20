# go-discord-irc

[![Go Report Card](https://goreportcard.com/badge/github.com/qaisjp/go-discord-irc)](https://goreportcard.com/report/github.com/qaisjp/go-discord-irc)
[![GoDoc](https://godoc.org/github.com/qaisjp/go-discord-irc?status.svg)](https://godoc.org/github.com/qaisjp/go-discord-irc)

[![Preview](https://i.imgur.com/YpCqzdn.gif)](https://i.imgur.com/YpCqzdn.webm)

**Is this being maintained?** Sort of. I'm here. I'll read your issues & accept pull requests. But eventually I want to merge all this functionality into the much superior [matterbridge by 42wim](https://github.com/42wim/matterbridge).

This is an IRC to Discord bridge built just for [@compsoc-edinburgh](http://github.com/compsoc-edinburgh) and
[ImaginaryNet](http://imaginarynet.uk/).

- The `IRC -> Discord` side of things work as you would expect it to: messages on IRC send to Discord as the bot user,
as per usual.
- The `Discord -> IRC` side of things is a little different. On connect, this bot will join the server with the `~d`,
and spawn additional connections for each online person in the Discord.
- Supports bidirectional PMs. (Not user friendly, but it works.)

## Gotchas

Things to keep in mind in terms of functionality:

- This does not work with private Discord channels properly (all discord users are added to the channel)
- **DO NOT USE THE SAME DISCORD BOT (API KEY) ACROSS MULTIPLE GUILDS (SERVERS).**

It's built with configuration in mind, but may need a little bit of tweaking for it to work for you:

- **Hardcoded**: Hostnames are hardcoded to follow the IPv6 IPs listed [here](https://github.com/qaisjp/go-discord-irc/issues/2).
- **Server mod**: Discord usernames will have `~d` tacked on at the end. This usually invalid nickname character required custom modifications to the IRC server code. This is configurable.
- **Server config**: This uses `WEBIRC` to give Discord users on IRC a distinct hostname. [See here](https://kiwiirc.com/docs/webirc).

## Configuration

The binary takes three flags:

- `--config filename.yaml`: to pass along a configuration file containing things like passwords and channel options
- `--simple`: to only spawn one connection (the listener will send across messages from Discord) instead of a connection per online Discord user
- `--debug`: provide this flag to print extra debug info. Setting this flag to false (or not providing this flag) will take the value from the config file instead
- `--insecure`: used to skip TLS verification (false = use value from settings)
- `--no-tls`: turns off TLS

The config file is a yaml formatted file with the following fields:

- `discord_token`, [the bot user token](https://github.com/reactiflux/discord-irc/wiki/Creating-a-discord-bot-&-getting-a-token)
- `irc_server`, IRC server address
- `irc_pass`, optional password for connecting to the IRC server
- `channel_mappings`, a dict with irc channel as key (prefixed with `#`) and Discord channel ID as value
- `suffix`, appended to each Discord user's nickname when they are connected to IRC
- `irc_listener_name`, the name of the irc listener
- `guild_id`, the Discord guild (server) id
- `webirc_pass`, optional, but recommended for regular (non-simple) usage. this must be obtained by the IRC sysops
- `debug`, debug mode
- `insecure`, TLS will skip verification (but still uses TLS)
- `no_tls`, turns off TLS
- `webhook_prefix`, a prefix for webhooks, so we know which ones to keep and which ones to delete
- `webhook_limit`, integer limit for the maximum number of webhooks to create
- `nickserv_identify`, optional, on connect this message will be sent: `PRIVMSG nickserv IDENTIFY <value>`, you can provide both a username and password if your ircd supports it

**The filename.yaml file is continuously read from and many changes will automatically update on the bridge. This means you can add or remove channels without restarting the bot.**

An example configuration file (those marked as `requires restart` definitely require restart, but others may not currently be configured to automatically update):

```
discord_token: abc.def.ghi
irc_server: localhost:6697
guild_id: 315277951597936640
nickserv_identify: password123
channel_mappings:
  "#bottest chanKey": 316038111811600387
  "#bottest2": 318327329044561920
suffix: "_d2"
irc_listener_name: "_d2"
webirc_pass: abcdef.ghijk.lmnop
insecure: true # this requires restart
no_tls: false # requires restart
debug: false
webhook_prefix: "(auto-test)" # this probably requires restart
webhook_limit: 3
#simple: true # this requires restart
```

This bot needs permissions to manage webhooks as it creates webhooks on the go.

```
https://discordapp.com/oauth2/authorize?&client_id=<YOUR_CLIENT_ID_HERE>&scope=bot&permissions=0x20000000
```

## Docker

First edit `config.yml` file to your needs.
Then launch `docker build -t go-discord-irc .` in the repository root folder.
And then `docker run -d go-discord-irc` to run the bot in background.

