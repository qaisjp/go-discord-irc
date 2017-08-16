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
- **Dependency mod**: Right now one of the dependencies ([github.com/thoj/irc-event](https://github.com/thoj/irc-event)) needs to be modified.
This is not yet included as one day I hope to submit a proper pull request supporting WebIRC.
- **Server mod**: Discord usernames contain `~`. This usually invalid nickname character required custom modifications to the IRC server code.
- **Server config**: This uses `WEBIRC` to give Discord users on IRC a distinct hostname. [See here](https://kiwiirc.com/docs/webirc).

## Configuration

Refer to `main.go` for the flag list. Here is my modified script for making things work:

```
./go-discord-irc \
        --discord_token "bot_token_here" \
        --irc_server "irchost:6697" \
        --guild_id "guild_id_here" \
        --webirc_pass "verylongpassword"
```

This bot needs permissions to manage webhooks.

```
https://discordapp.com/oauth2/authorize?&client_id=<YOUR_CLIENT_ID_HERE>&scope=bot&permissions=0x20000000
```

To tell the bot what Discord channels map IRC channels, you need to create webhooks.

- Server Settings > Webhooks
- Use the `Create Webhook` to create a webhook. You need to create a webhook for each channel you bridge.
- Set "name" to `IRC: <channel>`, where `<channel>` is the IRC channel with the leading `#`
- Set "channel" to the Discord channel you want to track.

Example, to map "#general" on Discord to "#compsoc" on IRC, you need to create a webhook with:
- Name: `IRC: #compsoc`
- Channel: `#general`

The bridge only reads the list when it is started, so restart the bridge if need be. Undefined behaviour if you modify the webhook whilst the bridge is running. In the future support for reloading whilst the bridge is running may be added.
