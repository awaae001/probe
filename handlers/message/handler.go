package message

import (
	"github.com/bwmarrin/discordgo"
)

// MessageHandler defines the interface for handling Discord message events.
// Each mode (e.g., "base", "plus") will have its own implementation of this interface.
type MessageHandler interface {
	// HandleCreate is called when a new message is created.
	HandleCreate(s *discordgo.Session, m *discordgo.MessageCreate)

	// HandleUpdate is called when a message is updated (edited).
	HandleUpdate(s *discordgo.Session, m *discordgo.MessageUpdate)

	// HandleDelete is called when a message is deleted.
	HandleDelete(s *discordgo.Session, m *discordgo.MessageDelete)

	// Close is called to release any resources held by the handler, such as database connections.
	Close() error
}
