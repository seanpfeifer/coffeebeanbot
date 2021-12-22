// Package coffeebeanbot is a coffee bean inspired bot created to help me through my day.
// Its current focus is to handle "Pomodoro Technique"-style timeboxing notification.
package coffeebeanbot

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/seanpfeifer/coffeebeanbot/metrics"
	"github.com/seanpfeifer/coffeebeanbot/pomodoro"
)

const (
	discordBotPrefix    = "Bot "
	pomDuration         = time.Minute * 25
	voiceWaitTime       = time.Millisecond * 250 // The amount of time to sleep before speaking & leaving the voice channel
	baseAuthURLTemplate = "https://discordapp.com/api/oauth2/authorize?client_id=%s&scope=bot"
)

// cmdHandler is the type for our functions that will be called upon receiving commands from a user.
type cmdHandler func(s *discordgo.Session, m *discordgo.MessageCreate, extra string)

type botCommand struct {
	handler       cmdHandler
	desc          string
	exampleParams string
}

// Bot contains the information needed to run the Discord bot
type Bot struct {
	Config      Config
	secrets     Secrets
	cmdHandlers map[string]botCommand
	discord     *discordgo.Session
	logger      Logger
	metrics     metrics.Recorder

	helpMessage        string
	poms               pomodoro.ChannelPomMap
	workEndAudioBuffer [][]byte
}

// NewBot is how you should create a new Bot in order to assure that all initialization has been completed.
func NewBot(config Config, secrets Secrets, logger Logger, recorder metrics.Recorder) *Bot {
	bot := &Bot{
		Config:  config,
		secrets: secrets,
		logger:  logger.Named("bot"),
		metrics: recorder,
		poms:    pomodoro.NewChannelPomMap(),
	}

	bot.registerCmdHandlers()
	bot.helpMessage = bot.buildHelpMessage()
	bot.loadSounds()

	return bot
}

func (bot *Bot) loadSounds() {
	audioBuffer, err := LoadDiscordAudio(bot.Config.WorkEndAudio)
	if !LogIfError(bot.logger, err, "Error loading audio") {
		bot.workEndAudioBuffer = audioBuffer
	}
}

func (bot *Bot) registerCmdHandlers() {
	bot.cmdHandlers = map[string]botCommand{
		"start":  {handler: bot.onCmdStartPom, desc: "Starts a Pomodoro work cycle on the channel. You can optionally specify the task you are working on", exampleParams: "Create a new notification sound, add an example"},
		"cancel": {handler: bot.onCmdCancelPom, desc: "Cancels the current Pomodoro work cycle on the channel", exampleParams: ""},
		"help":   {handler: bot.onCmdHelp, desc: "Shows this help message", exampleParams: ""},
	}
}

func (bot *Bot) buildHelpMessage() string {
	helpBuf := bytes.Buffer{}
	helpBuf.WriteString("This bot was written by Sean Pfeifer to help him get more done.\n")

	// I don't really care about ordering right now - this is intentionally using the map iteration order,
	// which I am aware is pseudo-random.
	for cmdStr, cmd := range bot.cmdHandlers {
		helpBuf.WriteString(fmt.Sprintf("\n•  **%s**  -  %s\n", cmdStr, cmd.desc))
		helpBuf.WriteString(fmt.Sprintf("    Example: `%s%s %s`\n", bot.Config.CmdPrefix, cmdStr, cmd.exampleParams))
	}

	helpBuf.WriteString("\nTo have me join your server, click on my name and 'Add to Server'.")

	return helpBuf.String()
}

// Start will start the bot, blocking until completion
func (bot *Bot) Start() error {
	if bot.secrets.AuthToken == "" {
		return errors.New("no auth token found in config")
	}

	var err error
	bot.discord, err = discordgo.New(discordBotPrefix + bot.secrets.AuthToken)
	if err != nil {
		return err
	}

	bot.discord.AddHandler(bot.onReady)
	bot.discord.AddHandler(bot.onMessageReceived)
	// Simply for keeping track of how many guilds we're a part of (to monitor bot health)
	bot.discord.AddHandler(bot.onGuildCreate)
	bot.discord.AddHandler(bot.onGuildDelete)

	err = bot.discord.Open()
	if err != nil {
		return err
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	return bot.discord.Close()
}

func (bot *Bot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	numGuilds := int64(len(s.State.Guilds))
	bot.logger.Info("Bot connected and ready", "userName", event.User.Username+"#"+event.User.Discriminator, "numGuilds", numGuilds)
	bot.metrics.RecordConnectedServers(numGuilds)
}

// onMessageReceived is called when a message is received on a channel that the bot is listening on.
// It will dispatch known commands to the command handlers, passing along any extra string information.
func (bot *Bot) onMessageReceived(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages created by this bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	msg := m.Content

	cmdPrefixLen := len(bot.Config.CmdPrefix)

	// Dispatch the command iff we have our prefix (case-insensitive).
	if len(msg) > cmdPrefixLen && strings.EqualFold(bot.Config.CmdPrefix, msg[0:cmdPrefixLen]) {
		afterPrefix := msg[cmdPrefixLen:]
		cmd := strings.SplitN(afterPrefix, " ", 2)

		if f, ok := bot.cmdHandlers[strings.ToLower(cmd[0])]; ok {
			rest := ""
			if len(cmd) > 1 {
				rest = cmd[1]
			}

			if f.handler != nil {
				f.handler(s, m, rest)
			} else {
				bot.logger.Error("nil handler for command", "command", cmd)
				s.ChannelMessageSend(m.ChannelID, "Command error - please contact support.")
			}
		}
	}
}

func (bot *Bot) onCmdHelp(s *discordgo.Session, m *discordgo.MessageCreate, extra string) {
	s.ChannelMessageSend(m.ChannelID, bot.helpMessage)
}

func (bot *Bot) onCmdStartPom(s *discordgo.Session, m *discordgo.MessageCreate, extra string) {
	channel, err := s.State.Channel(m.ChannelID)
	if LogIfError(bot.logger, err, "Could not find channel", "channelID", m.ChannelID) {
		// Could not find the channel, so simply log and exit
		return
	}

	// Make sure the user's text can't break out of our quote box.
	extra = strings.Replace(extra, "`", "", -1)
	extra = strings.TrimSpace(extra)

	notif := pomodoro.NotifyInfo{
		Title:     extra,
		UserID:    m.Author.ID,
		GuildID:   channel.GuildID,
		ChannelID: m.ChannelID,
	}

	if bot.poms.CreateIfEmpty(pomDuration, bot.onPomEnded, notif) {
		taskStr := "Started task  -  "
		if len(notif.Title) > 0 {
			taskStr = fmt.Sprintf("```md\n%s\n```", notif.Title)
		}

		msg := fmt.Sprintf("%s**%.1f minutes** remaining!", taskStr, pomDuration.Minutes())
		s.ChannelMessageSend(m.ChannelID, msg)
		bot.metrics.RecordStartPom()
		bot.metrics.RecordRunningPoms(int64(bot.poms.Count()))
	} else {
		s.ChannelMessageSend(m.ChannelID, "A Pomodoro is already running on this channel.")
	}
}

func (bot *Bot) onCmdCancelPom(s *discordgo.Session, m *discordgo.MessageCreate, extra string) {
	if exists := bot.poms.RemoveIfExists(m.ChannelID); !exists {
		s.ChannelMessageSend(m.ChannelID, "No Pomodoro running on this channel to cancel.")
	}
	// If this removal succeeds, then the onPomEnded callback will be called, so we don't need to do anything here.
}

// onPomEnded performs the notification
func (bot *Bot) onPomEnded(notif pomodoro.NotifyInfo, completed bool) {
	if completed {
		message := "Work cycle complete.  Time for a short break!"
		var toMention []string

		if len(notif.Title) > 0 {
			message = fmt.Sprintf("```md\n%s\n```%s", notif.Title, message)
		}

		user, err := bot.discord.User(notif.UserID)
		if err == nil {
			toMention = append(toMention, user.Mention())
		}
		// Doing this in a goroutine so we don't wait until the audio has been played to send the text notification.
		// This isn't required, but is my preference.
		go bot.playEndSound(notif)

		if len(toMention) > 0 {
			mentions := strings.Join(toMention, " ")
			message = fmt.Sprintf("%s\n%s", message, mentions)
		}

		bot.discord.ChannelMessageSend(notif.ChannelID, message)
	} else {
		bot.discord.ChannelMessageSend(notif.ChannelID, "Pomodoro cancelled!")
	}

	bot.metrics.RecordRunningPoms(int64(bot.poms.Count()))
}

// onGuildCreate is called when a Guild adds the bot.
func (bot *Bot) onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	bot.metrics.RecordConnectedServers(int64(len(s.State.Guilds)))
}

// onGuildDelete is called when a Guild removes the bot.
func (bot *Bot) onGuildDelete(s *discordgo.Session, event *discordgo.GuildDelete) {
	bot.metrics.RecordConnectedServers(int64(len(s.State.Guilds)))
}
