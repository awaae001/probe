package message

import (
	"context"
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
	db                *plusdb.PlusDB
	config            models.PlusGuildConfig
	ctx               context.Context      // 控制goroutine生命周期
	cancel            context.CancelFunc   // 取消函数
	recentEdits       map[int64]time.Time
	recentMessages    map[int64]time.Time  // 防止重复消息创建
	recentDeletions   map[int64]time.Time  // 防止重复删除事件
	editsMutex        sync.RWMutex
	messagesMutex     sync.RWMutex         // 保护 recentMessages
	deletionsMutex    sync.RWMutex         // 保护 recentDeletions
	lastCleanup       time.Time
	lastMsgCleanup    time.Time            // 消息清理时间戳
	lastDelCleanup    time.Time            // 删除清理时间戳
	cleanupMutex      sync.Mutex
	msgCleanupMutex   sync.Mutex           // 消息清理互斥锁
	delCleanupMutex   sync.Mutex           // 删除清理互斥锁
}

// NewPlusHandler creates a new handler for the plus mode.
func NewPlusHandler(config models.PlusGuildConfig) (*PlusHandler, error) {
	db, err := plusdb.NewPlusDB(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize plusDB for guild %s: %w", config.GuildsID, err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &PlusHandler{
		db:              db,
		config:          config,
		ctx:             ctx,
		cancel:          cancel,
		recentEdits:     make(map[int64]time.Time),
		recentMessages:  make(map[int64]time.Time),
		recentDeletions: make(map[int64]time.Time),
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

	// 防止重复处理消息创建事件（检查是否在最近3秒内已处理过）
	h.messagesMutex.RLock()
	if lastMsg, exists := h.recentMessages[messageID]; exists && time.Since(lastMsg) < 3*time.Second {
		h.messagesMutex.RUnlock()
		log.Printf("PlusHandler: 跳过重复的消息创建事件 - 消息ID: %d", messageID)
		return
	}
	h.messagesMutex.RUnlock()

	// 记录这次消息处理
	h.messagesMutex.Lock()
	h.recentMessages[messageID] = time.Now()
	h.messagesMutex.Unlock()

	// 定期清理旧的消息记录（每60秒清理一次）
	h.msgCleanupMutex.Lock()
	if time.Since(h.lastMsgCleanup) > 60*time.Second {
		go func() {
			defer h.msgCleanupMutex.Unlock()
			
			// 检查context是否已取消
			select {
			case <-h.ctx.Done():
				return
			default:
			}
			
			h.messagesMutex.Lock()
			for id, timestamp := range h.recentMessages {
				if time.Since(timestamp) > 30*time.Second {
					delete(h.recentMessages, id)
				}
			}
			h.messagesMutex.Unlock()
			h.lastMsgCleanup = time.Now()
		}()
	} else {
		h.msgCleanupMutex.Unlock()
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
			
			// 检查context是否已取消
			select {
			case <-h.ctx.Done():
				return
			default:
			}
			
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

	// 防止重复处理消息删除事件（检查是否在最近3秒内已处理过）
	h.deletionsMutex.RLock()
	if lastDel, exists := h.recentDeletions[messageID]; exists && time.Since(lastDel) < 3*time.Second {
		h.deletionsMutex.RUnlock()
		log.Printf("PlusHandler: 跳过重复的消息删除事件 - 消息ID: %d", messageID)
		return
	}
	h.deletionsMutex.RUnlock()

	// 记录这次删除处理
	h.deletionsMutex.Lock()
	h.recentDeletions[messageID] = time.Now()
	h.deletionsMutex.Unlock()

	// 定期清理旧的删除记录（每60秒清理一次）
	h.delCleanupMutex.Lock()
	if time.Since(h.lastDelCleanup) > 60*time.Second {
		go func() {
			defer h.delCleanupMutex.Unlock()
			
			// 检查context是否已取消
			select {
			case <-h.ctx.Done():
				return
			default:
			}
			
			h.deletionsMutex.Lock()
			for id, timestamp := range h.recentDeletions {
				if time.Since(timestamp) > 30*time.Second {
					delete(h.recentDeletions, id)
				}
			}
			h.deletionsMutex.Unlock()
			h.lastDelCleanup = time.Now()
		}()
	} else {
		h.delCleanupMutex.Unlock()
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

// Close closes the database connection and cancels all running goroutines.
func (h *PlusHandler) Close() error {
	// 取消所有正在运行的goroutine
	h.cancel()
	
	// 关闭数据库连接
	return h.db.Close()
}
