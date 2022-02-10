package coffeebeanbot

// Everything in this file is deprecated, since it is based off of reading chat messages

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/seanpfeifer/coffeebeanbot/pomodoro"
)

// onDeprecatedMessage is called when a message is received on a channel that the bot is listening on.
// It will dispatch known commands to the command handlers, passing along any extra string information.
func (bot *Bot) onDeprecatedMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
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

		rest := ""
		if len(cmd) > 1 {
			rest = cmd[1]
		}

		switch strings.ToLower(cmd[0]) {
		case "start":
			bot.onDeprecatedStart(s, m, rest)
		case "cancel":
			bot.onDeprecatedCancel(s, m, rest)
		}
	}
}

func (bot *Bot) onDeprecatedStart(s *discordgo.Session, m *discordgo.MessageCreate, extra string) {
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

func (bot *Bot) onDeprecatedCancel(s *discordgo.Session, m *discordgo.MessageCreate, extra string) {
	if exists := bot.poms.RemoveIfExists(m.ChannelID); !exists {
		s.ChannelMessageSend(m.ChannelID, "No Pomodoro running on this channel to cancel.")
	}
	// If this removal succeeds, then the onPomEnded callback will be called, so we don't need to do anything here.
}
