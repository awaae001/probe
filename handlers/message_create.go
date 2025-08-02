package handlers

import (
	"discord-bot/bot"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// MessageCreate handles regular message commands.
func MessageCreate(b *bot.Bot) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		prefix := viper.GetString("bot.prefix")
		if prefix == "" {
			prefix = "!" // Default prefix
		}

		if !strings.HasPrefix(m.Content, prefix) {
			return
		}

		content := strings.TrimPrefix(m.Content, prefix)
		parts := strings.Fields(content)
		if len(parts) == 0 {
			return
		}

		cmdName := parts[0]
		if cmd, ok := b.Commands[cmdName]; ok {
			cmd.MessageHandler(s, m)
		}
	}
}
