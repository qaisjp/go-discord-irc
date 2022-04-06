# go-discord-irc

[![Go Report Card](https://goreportcard.com/badge/github.com/qaisjp/go-discord-irc)](https://goreportcard.com/report/github.com/qaisjp/go-discord-irc)
[![GoDoc](https://godoc.org/github.com/qaisjp/go-discord-irc?status.svg)](https://godoc.org/github.com/qaisjp/go-discord-irc)

[![Preview](https://i.imgur.com/YpCqzdn.gif)](https://i.imgur.com/YpCqzdn.webm)

**Is this being maintained?** Yes. But I want to merge all this functionality
into the much superior
[matterbridge by 42wim](https://github.com/42wim/matterbridge).

This is IRC to Discord bridge was originally built for
[@compsoc-edinburgh](http://github.com/compsoc-edinburgh) and
[ImaginaryNet](http://imaginarynet.uk/), but now it looks like more people are
using it!

- The `IRC -> Discord` side of things work as you would expect it to: messages
  on IRC send to Discord as the bot user, as per usual.
- The `Discord -> IRC` side of things is a little different. On connect, this
  bot will join the server with the `~d`, and spawn additional connections for
  each online person in the Discord.
- Supports bidirectional PMs. (Not user friendly, but it works.)

**Features**

(not a full list)

- Every Discord user in your server will join your channel. Messages come from those "puppets", not from a single chat bridge user.
- Saying the puppet username will @ that person on Discord.
- When a Discord user's presence is "offline" or "idle", their irc puppet will
  have their AWAY status set.
- A Discord user offline for will disconnect from IRC after 24 hours (or
  whatever `cooldown_duration` you set).
- Join/Quit/Part/Kick messages are sent to Discord (configurable!)
- Replying to someone on Discord will prefix that someone's name, e.g. replying to Alex with "yes that's fine" will show up as `<you> Alex: yes, that's fine` on IRC.
- IRC users can send (custom!) emoji to Discord, just do `:somename:`. Discord emoji shows up like that on IRC.
- Reacting to a Discord message will send a CTCP ACTION (`/me`) on IRC.

## Gotchas

Things to keep in mind in terms of functionality:

- This does not work with private Discord channels properly (all discord users
  are added to the channel)
- **DO NOT USE THE SAME DISCORD BOT (API KEY) ACROSS MULTIPLE GUILDS
  (SERVERS).**

It's built with configuration in mind, but may need a little bit of tweaking for
it to work for you:

- **Hardcoded**: Hostnames are hardcoded to follow the IPv6 IPs listed
  [here](https://github.com/qaisjp/go-discord-irc/issues/2).
- **Defaults aren't usable**: You should set the `suffix` and `separator` config
  options. The default options require custom modifications to the IRC server.
- **Server config**: This uses `WEBIRC` to give Discord users on IRC a distinct
  hostname. [See here](https://kiwiirc.com/docs/webirc).

## Configuration

The binary takes three flags:

- `--config filename.yaml`: to pass along a configuration file containing things
  like passwords and channel options
- `--simple`: to only spawn one connection (the listener will send across
  messages from Discord) instead of a connection per online Discord user
- `--debug`: provide this flag to print extra debug info. Setting this flag to
  false (or not providing this flag) will take the value from the config file
  instead
- `--insecure`: used to skip TLS verification (false = use value from settings)
- `--no-tls`: turns off TLS

The config file is a yaml formatted file with the following fields:

| name                            | requires restart | default                                        | optional                     | description                                                                                                                                                              |
| ------------------------------- | ---------------- | ---------------------------------------------- | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `avatar_url`                    | No               | `https://ui-avatars.com/api/?name=${USERNAME}` | Yes                          | The URL for the API to use to tell Discord what Avatar to use for a User when the user's avatar cannot be found at Discord already.                                      |
| `discord_token`                 | Yes              |                                                | No                           | [The bot user token](https://github.com/reactiflux/discord-irc/wiki/Creating-a-discord-bot-&-getting-a-token)                                                            |
| `discord_message_filter`        | No               |                                                | Yes                          | Filters messages from Discord to IRC when they match.                                                                                                                    |
| `irc_message_filter`            | No               |                                                | Yes                          | Filters messages from IRC to Discord when they match.                                                                                                                    |
| `irc_server`                    | Yes              |                                                | No                           | IRC server address                                                                                                                                                       |
| `irc_server_name`               | Yes              |                                                | No                           | Used as a reference when PMing from Discord to IRC. Try to use short, simple one-word names like `freenode` or `swift`                                                   |
| `channel_mappings`              | No               |                                                | No                           | a dict with irc channel as key (prefixed with `#`) and Discord channel ID as value                                                                                       |
| `guild_id`                      | No               |                                                | No                           | the Discord guild (server) id                                                                                                                                            |
| `irc_pass`                      | Yes              |                                                | Yes                          | password for connecting to the IRC server                                                                                                                                |
| `suffix`                        | No               | `~d`                                           | Yes                          | appended to each Discord user's nickname when they are connected to IRC. If set to `_d2`, if the name will be `bob_d2`                                                   |
| `separator`                     | No               | `_`                                            | Yes                          | used in fallback situations. If set to `-`, the **fallback name** will be like `bob-7247_d2` (where `7247` is the discord user's discriminator, and `_d2` is the suffix) |
| `irc_listener_name`             | Yes              | `~d`                                           | The name of the irc listener |                                                                                                                                                                          |
| `ignored_discord_ids`           | Sometimes        |                                                | Yes                          | A list of Discord IDs to not relay to IRC                                                                                                                                |
| `allowed_discord_ids`           | Sometimes        | `null`                                         | Yes                          | A list of Discord IDs to relay to IRC. `null` allows all Discord users to be relayed to IRC. Hot reload: IDs added to the list require a presence change to take effect. |
| `puppet_username`               | No               | username of discord account being puppeted     | Yes                          | username to connect to irc with                                                                                                                                          |
| `webirc_pass`                   | No               |                                                | Yes                          | optional, but recommended for regular (non-simple) usage. this must be obtained by the IRC sysops                                                                        |
| `irc_listener_prejoin_commands` | Yes              |                                                | Yes                          | list of commands for the listener IRC connection to execute (right before joining channels)                                                                              |
| `irc_puppet_prejoin_commands`   | Yes              |                                                | Yes                          | list of commands for each Puppet IRC connection to execute (right before joining channels)                                                                               |
| `debug`                         | Yes              | false                                          | Yes                          | debug mode                                                                                                                                                               |
| `insecure`,                     | Yes              | false                                          | Yes                          | TLS will skip verification (but still uses TLS)                                                                                                                          |
| `no_tls`,                       | Yes              | false                                          | Yes                          | turns off TLS                                                                                                                                                            |
| `cooldown_duration`             | No               | 86400 (24 hours)                               | Yes                          | time in seconds for a discord user to be offline before it's puppet disconnects from irc                                                                                 |
| `show_joinquit`                 | No               | false                                          | yes                          | displays JOIN, PART, QUIT, KICK on discord                                                                                                                               |
| `max_nick_length`               | No               | 30                                             | yes                          | Maximum allowed nick length                                                                                                                                              |
| `ignored_irc_hostmasks`         | No               |                                                | Yes                          | A list of IRC users identified by hostmask to not relay to Discord, uses matching syntax as in [glob](https://github.com/gobwas/glob)                                    |
| `connection_limit`              | Yes              | 0                                              | Yes                          | How many connections to IRC (including our listener) to spawn (limit of 0 or less means unlimited)                                                                       |

**The filename.yaml file is continuously read from and many changes will
automatically update on the bridge. This means you can add or remove channels
without restarting the bot.**

An example configuration file can be seen in [`config.yml`](./config.yml). Those
marked as `requires restart` definitely require restart, but others may not
currently be configured to automatically update.

This bot needs permissions to manage webhooks as it creates webhooks on the go.

```
https://discordapp.com/oauth2/authorize?&client_id=<YOUR_CLIENT_ID_HERE>&scope=bot&permissions=0x20000000
```

**NEW IN 2020**

Make sure you also give the bot application these intents too:

![](https://user-images.githubusercontent.com/923242/97645553-23c34f00-1a45-11eb-95f0-76130261f0ab.png)

**NEW IN 2022**

Make sure you give the message content intent too:

![](https://user-images.githubusercontent.com/923242/161871952-e8e4a1c0-2751-4d42-9f30-64666ce87120.png)

## Docker

First edit `config.yml` file to your needs.
Then launch `docker build -t go-discord-irc .` in the repository root folder.
And then `docker run -d go-discord-irc` to run the bot in background.

## Development

A Makefile is provided to make getting started easier.

To build a binary run `make build` this will produce a binary of `go-discord-irc` in the root dir.

To build and run the binary run `make run`, this will use the `config.yaml` and start in debug.

To Execute tests run `make test`

Dependencies will be updated and installed with all the above commands or by running `make dev`
