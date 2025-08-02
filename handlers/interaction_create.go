package handlers

import (
	"discord-bot/bot"

	"github.com/bwmarrin/discordgo"
)

// InteractionCreate handles slash command interactions.
func InteractionCreate(b *bot.Bot) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			CommandDispatcher(s, i)
		case discordgo.InteractionApplicationCommandAutocomplete:
			HandleAutocomplete(s, i)
		}
	}
}
