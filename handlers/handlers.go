package handlers

import (
	"discord-bot/bot"
	"discord-bot/handlers/thread"
	"log"

	"github.com/bwmarrin/discordgo"
)

// Register all handlers to the bot.
func Register(b *bot.Bot) {
	// Initialize message collector
	if err := InitMessageCollector(); err != nil {
		log.Printf("Failed to initialize message collector: %v", err)
	}

	// Register event handlers
	b.Session.AddHandler(InteractionCreate(b))
	b.Session.AddHandler(thread.ThreadCreateHandler)
	b.Session.AddHandler(thread.ThreadDeleteHandler)
	b.Session.AddHandler(MessageCreateHandler(b))
	// b.Session.AddHandler(MemberAddHandler)
	// b.Session.AddHandler(MemberRemoveHandler)
	// b.Session.AddHandler(MemberUpdateHandler)

	// Add a ready handler to log when the bot is connected.
	b.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
}
