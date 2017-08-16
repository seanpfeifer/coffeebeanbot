package coffeebeanbot

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) playEndSound(notif NotifyInfo) error {
	// Find the user in the voice chat for the guild
	voiceChannelID := findUserVoiceChannelID(bot.discord, notif.GuildID, notif.UserID)
	if voiceChannelID == "" {
		return nil
	}

	return playSound(bot.discord, notif.GuildID, voiceChannelID, bot.workEndAudioBuffer)
}

func findUserVoiceChannelID(s *discordgo.Session, guildID, userID string) string {
	channelID := ""
	guild, err := s.Guild(guildID)
	if err != nil {
		return channelID
	}

	for _, voiceState := range guild.VoiceStates {
		if voiceState.UserID == userID {
			channelID = voiceState.ChannelID
			break
		}
	}

	return channelID
}

func playSound(s *discordgo.Session, guildID, channelID string, audioBuffer [][]byte) error {
	// Simply don't play the audio if the buffer is nil.
	if audioBuffer == nil {
		return nil
	}

	voice, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}

	time.Sleep(voiceWaitTime)

	voice.Speaking(true)
	for _, buf := range audioBuffer {
		voice.OpusSend <- buf
	}

	voice.Speaking(false)

	time.Sleep(voiceWaitTime)
	return voice.Disconnect()
}

// LoadDiscordAudio will load a DCA file, returning the data and/or any error that occurred.
func LoadDiscordAudio(filename string) ([][]byte, error) {
	audioBuffer := make([][]byte, 0)

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	// Ensure we close the file and log the potential regardless of what happens
	defer func() {
		fErr := file.Close()
		if fErr != nil {
			log.Printf("Error closing audio file '%s': %v", filename, fErr)
		}
	}()

	var opusLen int16
	for {
		// Read the length of the next packet of Opus audio data
		err = binary.Read(file, binary.LittleEndian, &opusLen)

		// EOF errors are perfectly normal - we're done
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return audioBuffer, nil
		}

		// Otherwise, we don't expect this error and should exit
		if err != nil {
			return nil, err
		}

		pcmBuf := make([]byte, opusLen)
		err = binary.Read(file, binary.LittleEndian, &pcmBuf)
		// No end of file errors should occur at this point - we expect to have the data that was promised.
		if err != nil {
			return nil, err
		}
		// Otherwise we add the data to our slice of slices
		audioBuffer = append(audioBuffer, pcmBuf)
	}
}
