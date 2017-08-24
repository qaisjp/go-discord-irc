# go-discord-irc [![Go Report Card](https://goreportcard.com/badge/github.com/qaisjp/go-discord-irc)](https://goreportcard.com/report/github.com/qaisjp/go-discord-irc)

[![Preview](https://i.imgur.com/he1euVW.gif)](https://i.imgur.com/he1euVW.webm)

This is an IRC to Discord bridge built just for [@compsoc-edinburgh](http://github.com/compsoc-edinburgh) and
[ImaginaryNet](http://imaginarynet.uk/).

- The `IRC -> Discord` side of things work as you would expect it to: messages on IRC send to Discord as the bot user,
as per usual.
- The `Discord -> IRC` side of things is a little different. On connect, this bot will join the server with the `~d`,
and spawn additional connections for each online person in the Discord.

## Gotchas

Things to keep in mind in terms of functionality:

- This does not work with private IRC/Discord channels (yet)

It's built with configuration in mind, but may need a little bit of tweaking for it to work for you:

- **Hardcoded**: Hostnames are hardcoded to follow the IPv6 IPs listed [here](https://github.com/qaisjp/go-discord-irc/issues/2).
- **Server mod**: Discord usernames will have `~d` tacked on at the end. This usually invalid nickname character required custom modifications to the IRC server code. This is configurable.
- **Server config**: This uses `WEBIRC` to give Discord users on IRC a distinct hostname. [See here](https://kiwiirc.com/docs/webirc).

## Configuration

Refer to `main.go` for the flag list. Here is my modified script for making things work:

```
./go-discord-irc \
        --discord_token "bot_token_here" \
        --channel_mappings "discordchannelid:#ircchannel,discordchannelid2:#ircchannel2,..." \
        --irc_server "irchost:6697" \
        --guild_id "guild_id_here" \
        --webirc_pass "verylongpassword"
```

This bot needs permissions to manage webhooks as it creates webhooks on the go.

```
https://discordapp.com/oauth2/authorize?&client_id=<YOUR_CLIENT_ID_HERE>&scope=bot&permissions=0x20000000
```
