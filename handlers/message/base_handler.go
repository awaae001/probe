package message

import (
	"discord-bot/database/message/basedb"
	"discord-bot/models"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/bwmarrin/discordgo"
)

// BaseHandler handles message events for the "base" mode.
type BaseHandler struct {
	db     *basedb.BaseDB
	config models.BaseGuildConfig
}

// NewBaseHandler creates a new handler for the base mode.
func NewBaseHandler(config models.BaseGuildConfig) (*BaseHandler, error) {
	db, err := basedb.NewBaseDB(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize baseDB for guild %s: %w", config.GuildsID, err)
	}
	return &BaseHandler{db: db, config: config}, nil
}

// HandleCreate processes new messages for the base mode.
func (h *BaseHandler) HandleCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot || m.GuildID != h.config.GuildsID || slices.Contains(h.config.Exclude, m.ChannelID) {
		return
	}

	messageID, userID, guildID, channelID, err := parseIDs(m.ID, m.Author.ID, m.GuildID, m.ChannelID)
	if err != nil {
		log.Printf("BaseHandler: Error parsing IDs: %v", err)
		return
	}

	message := models.Message{
		MessageID: messageID,
		UserID:    userID,
		GuildID:   guildID,
		ChannelID: channelID,
		Timestamp: time.Now().Unix(),
	}

	if err := h.db.SaveMessage(message); err != nil {
		log.Printf("BaseHandler: Error saving message for guild %s: %v", h.config.GuildsID, err)
	}
}

// HandleUpdate does nothing for the base mode.
func (h *BaseHandler) HandleUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) {
	// Base mode does not track edits.
}

// HandleDelete does nothing for the base mode.
func (h *BaseHandler) HandleDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	// Base mode does not track deletions.
}

// Close closes the database connection.
func (h *BaseHandler) Close() error {
	return h.db.Close()
}
