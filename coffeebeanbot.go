// Package coffeebeanbot is a coffee bean inspired bot created to help me through my day.
// Its current focus is to handle "Pomodoro Technique"-style timeboxing notification.
package coffeebeanbot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	discordBotPrefix = "Bot "
	pomDuration      = time.Minute * 25
	voiceWaitTime    = time.Millisecond * 250 // The amount of time to sleep before speaking & leaving the voice channel
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
	started     time.Time
	cmdHandlers map[string]botCommand
	discord     *discordgo.Session

	helpMessage        string
	poms               channelPomMap
	workEndAudioBuffer [][]byte
}

// Config is the Bot's configuration data
type Config struct {
	AuthToken    string `json:"authToken"`
	CmdPrefix    string `json:"cmdPrefix"`
	WorkEndAudio string `json:"workEndAudio"`
}

// LoadConfigFile loads the config from the given path, returning the config or an error if one occurred.
// I generally prefer config files over environment variables, due to the ease of setting them up as secrets
// in Kubernetes.
func LoadConfigFile(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, err
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)

	return &cfg, err
}

// NewBot is how you should create a new Bot in order to assure that all initialization has been completed.
func NewBot(config Config) *Bot {
	bot := &Bot{
		Config: config,
		poms:   newChannelPomMap(),
	}

	bot.registerCmdHandlers()
	bot.helpMessage = bot.buildHelpMessage()
	bot.loadSounds()

	return bot
}

func (bot *Bot) loadSounds() {
	audioBuffer, err := LoadDiscordAudio(bot.Config.WorkEndAudio)
	if err != nil {
		log.Printf("Error loading audio: %v", err)
	} else {
		bot.workEndAudioBuffer = audioBuffer
	}
}

func (bot *Bot) registerCmdHandlers() {
	bot.cmdHandlers = map[string]botCommand{
		"ping":    {handler: bot.onCmdPing, desc: "Pings the bot to ensure it is currently running", exampleParams: ""},
		"started": {handler: bot.onCmdStarted, desc: "Shows when the current version of the bot started running", exampleParams: ""},
		"echo":    {handler: bot.onCmdEcho, desc: "Echoes the given message back", exampleParams: "Hello, world!"},
		"start":   {handler: bot.onCmdStartPom, desc: "Starts a Pomodoro work cycle on the channel", exampleParams: ""},
		"cancel":  {handler: bot.onCmdCancelPom, desc: "Cancels the current Pomodoro work cycle on the channel", exampleParams: ""},
		"help":    {handler: bot.onCmdHelp, desc: "Shows this help message", exampleParams: ""},
	}
}

func (bot *Bot) buildHelpMessage() string {
	helpBuf := bytes.Buffer{}
	helpBuf.WriteString("This bot was written by Sean A. Pfeifer to help him get more done.\n")

	// I don't really care about ordering right now - this is intentionally using the map iteration order,
	// which I am aware is pseudo-random.
	// TODO: Add a "group" attribute to the commands, and sort by group, then command.
	for cmdStr, cmd := range bot.cmdHandlers {
		helpBuf.WriteString(fmt.Sprintf("\n•  **%s**  -  %s\n", cmdStr, cmd.desc))
		helpBuf.WriteString(fmt.Sprintf("    Example: `%s%s %s`\n", bot.Config.CmdPrefix, cmdStr, cmd.exampleParams))
	}

	return helpBuf.String()
}

// Start will start the bot, blocking until completion
func (bot *Bot) Start() error {
	if bot.Config.AuthToken == "" {
		return errors.New("no auth token found in config")
	}

	var err error
	bot.discord, err = discordgo.New(discordBotPrefix + bot.Config.AuthToken)
	if err != nil {
		return err
	}

	bot.discord.AddHandler(bot.onReady)
	bot.discord.AddHandler(bot.onMessageReceived)

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
	bot.started = time.Now()
	log.Printf("Bot connected and ready as '%s#%s'!", event.User.Username, event.User.Discriminator)
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
				log.Printf("Error: nil handler for command '%s'", cmd)
				s.ChannelMessageSend(m.ChannelID, "Command error - please contact support.")
			}
		}
	}
}

func (bot *Bot) onCmdPing(s *discordgo.Session, m *discordgo.MessageCreate, extra string) {
	s.ChannelMessageSend(m.ChannelID, "Pong!")
}

func (bot *Bot) onCmdStarted(s *discordgo.Session, m *discordgo.MessageCreate, extra string) {
	s.ChannelMessageSend(m.ChannelID, "I started "+bot.started.String())
}

func (bot *Bot) onCmdEcho(s *discordgo.Session, m *discordgo.MessageCreate, extra string) {
	// Make sure the echoed text can't break out of our quote box.
	extra = strings.Replace(extra, "`", "", -1)
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Echo:  `%s`", extra))
}

func (bot *Bot) onCmdHelp(s *discordgo.Session, m *discordgo.MessageCreate, extra string) {
	s.ChannelMessageSend(m.ChannelID, bot.helpMessage)
}

func (bot *Bot) onCmdStartPom(s *discordgo.Session, m *discordgo.MessageCreate, extra string) {
	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		// Could not find the channel, so simply log and exit
		log.Printf("Could not find channel for ChannelID '%s'", m.ChannelID)
		return
	}

	notif := NotifyInfo{
		extra,
		m.Author.ID,
		channel.GuildID,
	}

	if bot.poms.CreateIfEmpty(m.ChannelID, pomDuration, func() { bot.onPomEnded(m.ChannelID) }, notif) {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Pomodoro started - **%.1f minutes** remaining!", pomDuration.Minutes()))
	} else {
		s.ChannelMessageSend(m.ChannelID, "A Pomodoro is already running on this channel.")
	}
}

func (bot *Bot) onCmdCancelPom(s *discordgo.Session, m *discordgo.MessageCreate, extra string) {
	if notif := bot.poms.RemoveIfExists(m.ChannelID); notif != nil {
		// TODO: Use the NotifyInfo here?
		s.ChannelMessageSend(m.ChannelID, "Pomodoro cancelled!")
	} else {
		s.ChannelMessageSend(m.ChannelID, "No Pomodoro running on this channel to cancel.")
	}
}

// onPomEnded performs the notification
func (bot *Bot) onPomEnded(channelID string) {
	notif := bot.poms.RemoveIfExists(channelID)
	message := "Pomodoro ended.  Time for a short break!"

	var toMention []string

	if notif != nil {
		user, err := bot.discord.User(notif.UserID)
		if err == nil {
			toMention = append(toMention, user.Mention())
		}
		go bot.playEndSound(*notif)
	} else {
		bot.discord.ChannelMessageSend(channelID, message)
	}

	if len(toMention) > 0 {
		mentions := strings.Join(toMention, " ")
		bot.discord.ChannelMessageSend(channelID, fmt.Sprintf("%s %s", message, mentions))
	} else {
		bot.discord.ChannelMessageSend(channelID, message)
	}
}
