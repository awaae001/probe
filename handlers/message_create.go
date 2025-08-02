package handlers

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// MessageCreate will be called every time a new message is created on any channel that the authenticated bot has access to.
func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	prefix := viper.GetString("bot.prefix")
	if prefix == "" {
		prefix = "!" // Default prefix
	}

	if strings.HasPrefix(m.Content, prefix) {
		command := strings.TrimPrefix(m.Content, prefix)

		switch command {
		case "ping":
			s.ChannelMessageSend(m.ChannelID, "Pong!")
		case "pong":
			s.ChannelMessageSend(m.ChannelID, "Ping!")
		}
	}
}
