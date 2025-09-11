package message

import (
	"discord-bot/database/message/plusdb"
	"discord-bot/models"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// PlusHandler handles message events for the "plus" mode, including complex edit handling.
type PlusHandler struct {
	db           *plusdb.PlusDB
	config       models.PlusGuildConfig
	recentEdits  map[int64]time.Time
	editsMutex   sync.RWMutex
	lastCleanup  time.Time
	cleanupMutex sync.Mutex
}

// NewPlusHandler creates a new handler for the plus mode.
func NewPlusHandler(config models.PlusGuildConfig) (*PlusHandler, error) {
	db, err := plusdb.NewPlusDB(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize plusDB for guild %s: %w", config.GuildsID, err)
	}
	return &PlusHandler{
		db:          db,
		config:      config,
		recentEdits: make(map[int64]time.Time),
	}, nil
}

// HandleCreate processes new messages for the plus mode.
func (h *PlusHandler) HandleCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot || m.GuildID != h.config.GuildsID || slices.Contains(h.config.Exclude, m.ChannelID) {
		return
	}

	messageID, userID, guildID, channelID, err := parseIDs(m.ID, m.Author.ID, m.GuildID, m.ChannelID)
	if err != nil {
		log.Printf("PlusHandler: Error parsing IDs: %v", err)
		return
	}

	attachmentsJSON := getAttachmentsJSON(m.Attachments)
	message := models.Message{
		MessageID:      messageID,
		UserID:         userID,
		GuildID:        guildID,
		ChannelID:      channelID,
		Timestamp:      time.Now().Unix(),
		MessageContent: m.Content,
		Attachments:    attachmentsJSON,
		IsEdited:       false,
	}

	if err := h.db.InsertMessage(message); err != nil {
		log.Printf("PlusHandler: Error saving message for guild %s: %v", h.config.GuildsID, err)
	}
}

// HandleUpdate processes message edits for the plus mode, including duplicate prevention and original content fetching.
func (h *PlusHandler) HandleUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) {
	if (m.Author != nil && m.Author.Bot) || m.GuildID != h.config.GuildsID || slices.Contains(h.config.Exclude, m.ChannelID) {
		return
	}

	messageID, _, guildID, channelID, err := parseIDs(m.ID, "", m.GuildID, m.ChannelID)
	if err != nil {
		log.Printf("PlusHandler: Error parsing IDs for update: %v", err)
		return
	}

	// Duplicate edit event prevention logic
	h.editsMutex.RLock()
	if lastEdit, exists := h.recentEdits[messageID]; exists && time.Since(lastEdit) < 5*time.Second {
		h.editsMutex.RUnlock()
		return
	}
	h.editsMutex.RUnlock()

	h.editsMutex.Lock()
	h.recentEdits[messageID] = time.Now()
	h.editsMutex.Unlock()

	// Cleanup old edit records
	h.cleanupMutex.Lock()
	if time.Since(h.lastCleanup) > 60*time.Second {
		go func() {
			defer h.cleanupMutex.Unlock()
			h.editsMutex.Lock()
			for id, timestamp := range h.recentEdits {
				if time.Since(timestamp) > 30*time.Second {
					delete(h.recentEdits, id)
				}
			}
			h.editsMutex.Unlock()
			h.lastCleanup = time.Now()
		}()
	} else {
		h.cleanupMutex.Unlock()
	}

	editedAttachmentsJSON := getAttachmentsJSON(m.Attachments)
	originalContent, originalAttachments := h.getOriginalMessageContent(s, messageID, m.ChannelID, m.Content, editedAttachmentsJSON)

	edit := models.MessageEdit{
		MessageID:           messageID,
		GuildID:             guildID,
		ChannelID:           channelID,
		OriginalContent:     originalContent,
		EditedContent:       m.Content,
		OriginalAttachments: originalAttachments,
		EditedAttachments:   editedAttachmentsJSON,
		EditTimestamp:       time.Now().Unix(),
	}

	if err := h.db.InsertMessageEdit(edit); err != nil {
		log.Printf("PlusHandler: Error saving message edit for guild %s: %v", h.config.GuildsID, err)
	}
}

// HandleDelete processes message deletions for the plus mode.
func (h *PlusHandler) HandleDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	if m.GuildID != h.config.GuildsID || slices.Contains(h.config.Exclude, m.ChannelID) {
		return
	}

	messageID, _, guildID, channelID, err := parseIDs(m.ID, "", m.GuildID, m.ChannelID)
	if err != nil {
		log.Printf("PlusHandler: Error parsing IDs for deletion: %v", err)
		return
	}

	deletion := models.MessageDeletion{
		MessageID:         messageID,
		GuildID:           guildID,
		ChannelID:         channelID,
		DeletionTimestamp: time.Now().Unix(),
	}

	if err := h.db.InsertMessageDeletion(deletion); err != nil {
		log.Printf("PlusHandler: Error saving message deletion for guild %s: %v", h.config.GuildsID, err)
	}
}

// getOriginalMessageContent tries to fetch the original content of an edited message.
func (h *PlusHandler) getOriginalMessageContent(s *discordgo.Session, messageID int64, channelID, editedContent, editedAttachments string) (string, string) {
	// First, try to get the message from the Discord API. It might still have the original content.
	discordMessage, err := s.ChannelMessage(channelID, fmt.Sprintf("%d", messageID))
	if err == nil && discordMessage != nil {
		apiAttachmentsJSON := getAttachmentsJSON(discordMessage.Attachments)

		// If the content from the API is the same as the edited content, the CDN has likely updated.
		// In this case, we need to fall back to our database.
		if discordMessage.Content == editedContent && apiAttachmentsJSON == editedAttachments {
			log.Printf("PlusHandler: API content matches edited content for message %d. Falling back to DB.", messageID)
			originalMessage, dbErr := h.db.GetMessage(messageID)
			if dbErr == nil && originalMessage != nil {
				return originalMessage.MessageContent, originalMessage.Attachments
			}
			if dbErr != nil {
				log.Printf("PlusHandler: Error fetching original message from DB: %v", dbErr)
			}
			// If not found in DB or error, return empty strings.
			return "", ""
		}
		// Otherwise, the API content is the original content.
		return discordMessage.Content, apiAttachmentsJSON
	}

	if err != nil {
		log.Printf("PlusHandler: Failed to fetch message from Discord API: %v", err)
	}

	// If API fails, try the database directly.
	originalMessage, dbErr := h.db.GetMessage(messageID)
	if dbErr == nil && originalMessage != nil {
		return originalMessage.MessageContent, originalMessage.Attachments
	}
	if dbErr != nil {
		log.Printf("PlusHandler: Error fetching original message from DB after API failure: %v", dbErr)
	}

	return "", ""
}

// Close closes the database connection.
func (h *PlusHandler) Close() error {
	return h.db.Close()
}
