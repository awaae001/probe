package handlers

import (
	"discord-bot/bot"
	"log"

	"github.com/bwmarrin/discordgo"
)

// Register all handlers to the bot.
func Register(b *bot.Bot) {
	// Register event handlers
	b.Session.AddHandler(InteractionCreate(b))
	b.Session.AddHandler(ThreadCreateHandler)
	b.Session.AddHandler(ThreadDeleteHandler)

	// Add a ready handler to log when the bot is connected.
	b.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
}
