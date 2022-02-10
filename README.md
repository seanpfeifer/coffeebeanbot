# CoffeeBeanBot

[![Discord Invite](https://img.shields.io/badge/Invite%20Bot-Discord-blue.svg)](https://discordapp.com/api/oauth2/authorize?client_id=347286461252370432&scope=bot) [![GoDoc](https://godoc.org/github.com/seanpfeifer/coffeebeanbot?status.svg)](https://godoc.org/github.com/seanpfeifer/coffeebeanbot) [![Go Report Card](https://goreportcard.com/badge/github.com/seanpfeifer/coffeebeanbot)](https://goreportcard.com/report/github.com/seanpfeifer/coffeebeanbot) ![Build Status](https://github.com/seanpfeifer/coffeebeanbot/workflows/Tests/badge.svg)

`coffeebeanbot` is a coffee bean inspired Discord bot created to help me through my day. Its current focus is to handle "Pomodoro Technique"-style timeboxing notification.

If you simply want to use the bot, and not run your own or customize it, you can [invite it to your Discord server using this link](https://discordapp.com/api/oauth2/authorize?client_id=347286461252370432&scope=bot).

Use `!cbb help` to show the list of available commands.

## Getting Started

### Running using Docker

For Linux, assuming your `discord.toml` lives at `./secrets`:

```sh
docker run -v $(pwd)/secrets:/secrets docker.pkg.github.com/seanpfeifer/coffeebeanbot/cbb:1.0.5
```

For Windows PowerShell, assuming your `discord.toml` lives at `./secrets`:

```powershell
docker run -v ${PWD}\secrets:/secrets docker.pkg.github.com/seanpfeifer/coffeebeanbot/cbb:1.0.5
```

Metrics are disabled by default (see `Metrics` below). If you want your container to report to Stackdriver, you need to override the Docker container's parameters to add `-stackdriver`:

```sh
docker run -v $(pwd)/secrets:/secrets docker.pkg.github.com/seanpfeifer/coffeebeanbot/cbb:1.0.5 -cfg /bot/cfg.toml -secrets /secrets/discord.toml -stackdriver
```

### Installation

If you simply want to build + install the `cbb` binary on your own, run the following:
```sh
go get github.com/seanpfeifer/coffeebeanbot/cmd/cbb
```

Retrieve the package using:
```sh
go get github.com/seanpfeifer/coffeebeanbot
```

Build and install the `cbb` bot binary using:
```sh
go install github.com/seanpfeifer/coffeebeanbot/cmd/cbb
```

### Configuration

Two files are used for configuration:

* `cfg.toml` - general bot config
* `discord.toml` - bot secrets that shouldn't be shared with others

Create a `cfg.toml` file that exists wherever you want to run the bot from.

Sample `cfg.toml`:
```toml
cmdPrefix = "!cbb "
workEndAudio =  "audio/airhorn.dca"
```

Sample `discord.toml`:
```toml
authToken = "PASTE_AUTH_TOKEN_HERE"
appID = "PASTE_APPLICATION_ID_HERE"
```

The `authToken` and `appID` values can be found at https://discordapp.com/developers/applications/me

* Create your App and copy the `Application ID` into your `discord.toml`.
* Click `Create a Bot User`.
* Under "App Bot User" click "click to reveal" on `Token` and copy the value into your `discord.toml`.
* Save Changes.
* Ensure your bot's `Public Bot` setting is what you want it to be.

### Usage

Run the bot's `cbb` executable from the directory containing your `cfg.toml` and `./secrets/discord.toml`. See the output of `cbb -h` for details.

Invite the bot to one of your servers via the URL `https://discordapp.com/api/oauth2/authorize?client_id=CLIENT_ID_HERE&scope=bot`, replacing `CLIENT_ID_HERE` with your client ID shown in your config.

To show the current list of commands (and your bot's invite link), use the following command after you've invited the bot to one of your servers:
```
!cbb help
```

### Metrics

The following aggregated metrics can be recorded so you can tell how your service is performing:

* `connected_servers` - the current number of connected servers (Discord Guilds)
* `pomodoros_running` - the current number of Pomodoros actively running
* `pomodoros_started` - the count of Pomodoros started

Aggregated metrics for your running servers are only ever sent to either standard output, or to your Stackdriver if you have it configured. No personal information is ever sent from this service.

To enable sending metrics to stdout, use the command-line parameter `-stdoutMetrics`

To enable sending metrics to your configured Stackdriver, use the command-line parameter `-stackdriver`
