package handlers

import (
	"discord-bot/bot"

	"github.com/bwmarrin/discordgo"
)

// InteractionCreate handles slash command interactions.
func InteractionCreate(b *bot.Bot) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			if cmd, ok := b.Commands[i.ApplicationCommandData().Name]; ok {
				cmd.Handler(s, i)
			}
		}
	}
}
