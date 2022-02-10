// Package coffeebeanbot is a coffee bean inspired bot created to help me through my day.
// Its current focus is to handle "Pomodoro Technique"-style timeboxing notification.
package coffeebeanbot

import (
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
	discordBotPrefix = "Bot "
	pomDuration      = time.Minute * 25
	voiceWaitTime    = time.Millisecond * 250 // The amount of time to sleep before speaking & leaving the voice channel
	startCmdName     = "pomstart"
	cancelCmdName    = "pomcancel"
	flagEphemeral    = 1 << 6 // The flag that specifies that a message is "ephemeral". ie, only visible to the caller
)

// cmdHandler is the type for our functions that will be called upon receiving commands from a user.
type cmdHandler func(s *discordgo.Session, m *discordgo.MessageCreate, extra string)

type appCmdHandler func(*discordgo.Session, *discordgo.Interaction)

// Bot contains the information needed to run the Discord bot
type Bot struct {
	Config  Config
	secrets Secrets
	discord *discordgo.Session
	logger  Logger
	metrics metrics.Recorder

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

	bot.loadSounds()

	return bot
}

func (bot *Bot) loadSounds() {
	audioBuffer, err := LoadDiscordAudio(bot.Config.WorkEndAudio)
	if !LogIfError(bot.logger, err, "Error loading audio") {
		bot.workEndAudioBuffer = audioBuffer
	}
}

func (bot *Bot) registerAppCmds() error {
	// Intentionally not using the returned commands - we have no use for them, just the const names that we already have,
	// which we'll use to determine which command a person has triggered.
	_, err := bot.discord.ApplicationCommandBulkOverwrite(bot.secrets.AppID, "", []*discordgo.ApplicationCommand{
		{
			Name:        startCmdName,
			Description: "Starts a Pomodoro work cycle on the channel. You can optionally specify the task you are working on.",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:         "task",
					Type:         discordgo.ApplicationCommandOptionString,
					Description:  "The task you are working on",
					ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
				},
			},
		},
		{
			Name:        cancelCmdName,
			Description: "Cancels the current Pomodoro work cycle on the channel",
			Type:        discordgo.ChatApplicationCommand,
		},
	})

	return err
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
	bot.discord.AddHandler(bot.onDeprecatedMessage) // Deprecated - will be removed when this functionality is disabled by Discord
	// Our app command handler, which dispatches all incoming commands
	bot.discord.AddHandler(bot.onAppCmd)
	// Simply for keeping track of how many guilds we're a part of (to monitor bot health)
	bot.discord.AddHandler(bot.onGuildCreate)
	bot.discord.AddHandler(bot.onGuildDelete)

	err = bot.discord.Open()
	if err != nil {
		return err
	}

	defer bot.discord.Close()
	if err := bot.registerAppCmds(); err != nil {
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

func (bot *Bot) onAppCmd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Ignore anything that's not an app cmd
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	bot.logger.Info("appCmd", fmt.Sprintf("%+v", *i))

	data := i.ApplicationCommandData()
	switch data.Name {
	case startCmdName:
		bot.onAppCmdStart(s, i.Interaction)
	case cancelCmdName:
		bot.onAppCmdCancel(s, i.Interaction)
	}
}

func (bot *Bot) onAppCmdStart(s *discordgo.Session, i *discordgo.Interaction) {
	data := i.ApplicationCommandData()
	var task string
	if len(data.Options) > 0 {
		task = data.Options[0].StringValue()
	}

	channel, err := s.State.Channel(i.ChannelID)
	if LogIfError(bot.logger, err, "Could not find channel", "channelID", i.ChannelID) {
		// Could not find the channel, so simply log and exit
		return
	}

	notif := pomodoro.NotifyInfo{
		Title:     task,
		UserID:    i.Member.User.ID,
		GuildID:   channel.GuildID,
		ChannelID: i.ChannelID,
	}

	if bot.poms.CreateIfEmpty(pomDuration, bot.onPomEnded, notif) {
		taskStr := "Started task  -  "
		if len(notif.Title) > 0 {
			taskStr = fmt.Sprintf("```md\n%s\n```", notif.Title)
		}

		msg := fmt.Sprintf("%s**%.1f minutes** remaining!", taskStr, pomDuration.Minutes())
		s.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
			},
		})
		bot.metrics.RecordStartPom()
		bot.metrics.RecordRunningPoms(int64(bot.poms.Count()))
	} else {
		s.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "A Pomodoro is already running on this channel.",
				Flags:   flagEphemeral,
			},
		})
	}
}

func (bot *Bot) onAppCmdCancel(s *discordgo.Session, i *discordgo.Interaction) {
	if exists := bot.poms.RemoveIfExists(i.ChannelID); !exists {
		s.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No Pomodoro running on this channel to cancel.",
				Flags:   flagEphemeral,
			},
		})
	} else {
		s.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Pomodoro cancelled!",
			},
		})
	}
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
	}
	// Otherwise this was cancelled, and the reply will already be sent by the app cmd

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
