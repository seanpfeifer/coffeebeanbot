// Package coffeebeanbot is a coffee bean inspired bot created to help me through my day.
// Its current focus is to handle "Pomodoro Technique"-style timeboxing notification.
package coffeebeanbot

import (
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
	cmdPrefix        = "!bb "
	cmdPrefixLen     = len(cmdPrefix)
)

// cmdHandler is the type for our functions that will be called upon receiving commands from a user.
type cmdHandler func(s *discordgo.Session, m *discordgo.MessageCreate, extra string)

// Bot contains the information needed to run the Discord bot
type Bot struct {
	Config      Config
	started     time.Time
	cmdHandlers map[string]cmdHandler
}

// Config is the Bot's configuration data
type Config struct {
	AuthToken string `json:"authToken"`
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

func (bot *Bot) registerCmdHandlers() {
	bot.cmdHandlers = map[string]cmdHandler{
		"ping":    bot.onCmdPing,
		"started": bot.onCmdStarted,
		"echo":    bot.onCmdEcho,
	}
}

// Start will start the bot, blocking until completion
func (bot *Bot) Start() error {
	if bot.Config.AuthToken == "" {
		return errors.New("no auth token found in config")
	}

	discord, err := discordgo.New(discordBotPrefix + bot.Config.AuthToken)
	if err != nil {
		return err
	}

	bot.registerCmdHandlers()

	discord.AddHandler(bot.onReady)
	discord.AddHandler(bot.onMessageReceived)

	err = discord.Open()
	if err != nil {
		return err
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	return discord.Close()
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

	// Dispatch the command iff we have our prefix (case-insensitive).
	if len(msg) > cmdPrefixLen && cmdPrefix == strings.ToLower(msg[0:cmdPrefixLen]) {
		afterPrefix := msg[cmdPrefixLen:]
		cmd := strings.SplitN(afterPrefix, " ", 2)

		if f, ok := bot.cmdHandlers[strings.ToLower(cmd[0])]; ok {
			rest := ""
			if len(cmd) > 1 {
				rest = cmd[1]
			}

			f(s, m, rest)
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
