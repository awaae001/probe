package handlers

import (
	"discord-bot/bot"
	"discord-bot/database"
	"discord-bot/models"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// MessageCollector manages message collection and storage
type MessageCollector struct {
	messageDB     *database.MessageDB
	statusManager *database.StatusManager
	config        models.MessageConfig
}

var globalMessageCollector *MessageCollector

// 用于防止重复处理消息编辑事件
var (
	recentEdits    = make(map[int64]time.Time)
	editsMutex     sync.RWMutex
	lastCleanup    time.Time
	cleanupMutex   sync.Mutex
)

// validateGuildAndChannel checks if message collection is enabled for the guild and channel
func validateGuildAndChannel(guildID, channelID string) (models.GuildMsgConfig, bool) {
	// Skip if message collector is not initialized
	if globalMessageCollector == nil {
		return models.GuildMsgConfig{}, false
	}

	// Skip if no guild ID (DM messages)
	if guildID == "" {
		return models.GuildMsgConfig{}, false
	}

	// Check if this guild is configured for message collection
	guildConfig, exists := globalMessageCollector.config.MessageListener.Data[guildID]
	if !exists {
		return models.GuildMsgConfig{}, false
	}

	// Check if this channel is excluded
	if slices.Contains(guildConfig.Exclude, channelID) {
		return models.GuildMsgConfig{}, false
	}

	return guildConfig, true
}

// parseDiscordIDs converts Discord string IDs to int64
func parseDiscordIDs(messageID, userID, guildID, channelID string) (int64, int64, int64, int64, error) {
	msgID, err := strconv.ParseInt(messageID, 10, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("error parsing message ID %s: %w", messageID, err)
	}

	var uID int64
	if userID != "" {
		uID, err = strconv.ParseInt(userID, 10, 64)
		if err != nil {
			return 0, 0, 0, 0, fmt.Errorf("error parsing user ID %s: %w", userID, err)
		}
	}

	gID, err := strconv.ParseInt(guildID, 10, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("error parsing guild ID %s: %w", guildID, err)
	}

	cID, err := strconv.ParseInt(channelID, 10, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("error parsing channel ID %s: %w", channelID, err)
	}

	return msgID, uID, gID, cID, nil
}

// InitMessageCollector initializes the global message collector
func InitMessageCollector() error {
	// Load message configuration from viper
	var config models.MessageConfig
	if err := viper.UnmarshalKey("message_listener", &config.MessageListener); err != nil {
		return fmt.Errorf("failed to unmarshal message listener config: %w", err)
	}

	// Validate configuration
	if len(config.MessageListener.Data) == 0 {
		log.Println("No guilds configured for message collection, message collector disabled")
		return nil
	}

	messageDB := database.NewMessageDB(config)
	statusManager := database.NewStatusManager(config.MessageListener.DBStatus)
	globalMessageCollector = &MessageCollector{
		messageDB:     messageDB,
		statusManager: statusManager,
		config:        config,
	}

	log.Printf("Message collector initialized for %d guilds", len(config.MessageListener.Data))
	return nil
}

// MessageCreateHandler handles Discord message create events
func MessageCreateHandler(b *bot.Bot) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Skip bot messages
		if m.Author.Bot {
			return
		}

		// Validate guild and channel
		_, valid := validateGuildAndChannel(m.GuildID, m.ChannelID)
		if !valid {
			return
		}

		// Convert Discord IDs to int64
		messageID, userID, guildIDInt, channelID, err := parseDiscordIDs(m.ID, m.Author.ID, m.GuildID, m.ChannelID)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		// Extract attachment URLs
		var attachmentURLs []string
		for _, attachment := range m.Attachments {
			attachmentURLs = append(attachmentURLs, attachment.URL)
		}

		// Convert attachments to JSON string
		attachmentsJSON := ""
		if len(attachmentURLs) > 0 {
			if jsonData, err := json.Marshal(attachmentURLs); err == nil {
				attachmentsJSON = string(jsonData)
			}
		}

		// Create message record
		message := models.Message{
			MessageID:      messageID,
			UserID:         userID,
			GuildID:        guildIDInt,
			ChannelID:      channelID,
			Timestamp:      time.Now().Unix(),
			MessageContent: m.Content,
			Attachments:    attachmentsJSON,
			IsEdited:       false,
		}

		// Insert message into database
		if err := globalMessageCollector.messageDB.InsertMessage(message); err != nil {
			log.Printf("Error inserting message %d: %v", messageID, err)
			return
		}

		// Update status file with current database info
		go func() {
			dbPath, err := globalMessageCollector.messageDB.GetDBPath(m.GuildID)
			if err != nil {
				log.Printf("Error getting DB path for status update: %v", err)
				return
			}

			messageCount, err := globalMessageCollector.messageDB.GetMessageCount(guildIDInt)
			if err != nil {
				log.Printf("Error getting message count for status update: %v", err)
				return
			}

			if err := globalMessageCollector.statusManager.UpdateDatabaseInfo(m.GuildID, dbPath, messageCount); err != nil {
				log.Printf("Error updating database status: %v", err)
			}
		}()

		// Log successful message collection (can be removed in production)
		log.Printf("Message collected: Guild %s, Channel %s, User %s", m.GuildID, m.ChannelID, m.Author.ID)
	}
}

// MessageDeleteHandler handles Discord message delete events
func MessageDeleteHandler(b *bot.Bot) func(s *discordgo.Session, m *discordgo.MessageDelete) {
	return func(s *discordgo.Session, m *discordgo.MessageDelete) {
		// Validate guild and channel
		_, valid := validateGuildAndChannel(m.GuildID, m.ChannelID)
		if !valid {
			return
		}

		// Convert Discord IDs to int64 (using empty string for userID since delete events don't have user info)
		messageID, _, guildIDInt, channelIDInt, err := parseDiscordIDs(m.ID, "", m.GuildID, m.ChannelID)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		// Create deletion record
		deletion := models.MessageDeletion{
			MessageID:         messageID,
			GuildID:           guildIDInt,
			ChannelID:         channelIDInt,
			DeletionTimestamp: time.Now().Unix(),
		}

		// Record the deletion in the deletions table
		if err := globalMessageCollector.messageDB.InsertMessageDeletion(deletion); err != nil {
			log.Printf("Error recording message deletion %d: %v", messageID, err)
			return
		}

		// Log successful message deletion tracking
		log.Printf("Message deletion tracked: Guild %s, Channel %s, Message %s", m.GuildID, m.ChannelID, m.ID)
	}
}


// GetChannelStats retrieves channel statistics for a guild
func GetChannelStats(guildID int64, from, to *time.Time) ([]models.ChannelStat, error) {
	if globalMessageCollector == nil {
		return nil, fmt.Errorf("message collector not initialized")
	}
	return globalMessageCollector.messageDB.GetChannelStats(guildID, from, to)
}

// GetUserStats retrieves user statistics for a guild
func GetUserStats(guildID int64, channelIDs []int64, from, to *time.Time) ([]models.UserStat, error) {
	if globalMessageCollector == nil {
		return nil, fmt.Errorf("message collector not initialized")
	}
	return globalMessageCollector.messageDB.GetUserStats(guildID, channelIDs, from, to)
}

// GetMessageCount returns the total message count for a guild
func GetMessageCount(guildID int64) (int64, error) {
	if globalMessageCollector == nil {
		return 0, fmt.Errorf("message collector not initialized")
	}
	return globalMessageCollector.messageDB.GetMessageCount(guildID)
}

// CloseMessageCollector closes the message collector and its database connections
func CloseMessageCollector() error {
	if globalMessageCollector == nil {
		return nil
	}
	return globalMessageCollector.messageDB.Close()
}

// MessageUpdateHandler 处理 Discord 消息编辑事件
func MessageUpdateHandler(b *bot.Bot) func(s *discordgo.Session, m *discordgo.MessageUpdate) {
	return func(s *discordgo.Session, m *discordgo.MessageUpdate) {
		// 跳过机器人消息
		if m.Author != nil && m.Author.Bot {
			return
		}

		// 验证服务器和频道
		_, valid := validateGuildAndChannel(m.GuildID, m.ChannelID)
		if !valid {
			return
		}

		// 转换 Discord ID 为 int64
		messageID, _, guildIDInt, channelID, err := parseDiscordIDs(m.ID, "", m.GuildID, m.ChannelID)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		// 检查是否在最近5秒内已经处理过这条消息的编辑事件
		editsMutex.RLock()
		if lastEdit, exists := recentEdits[messageID]; exists {
			if time.Since(lastEdit) < 5*time.Second {
				editsMutex.RUnlock()
				log.Printf("跳过重复的消息编辑事件 - 消息ID: %d", messageID)
				return
			}
		}
		editsMutex.RUnlock()

		// 记录这次编辑处理
		editsMutex.Lock()
		recentEdits[messageID] = time.Now()
		editsMutex.Unlock()

		// 定期清理旧记录（每60秒清理一次）
		cleanupMutex.Lock()
		if time.Since(lastCleanup) > 60*time.Second {
			go func() {
				defer cleanupMutex.Unlock()
				editsMutex.Lock()
				for id, timestamp := range recentEdits {
					if time.Since(timestamp) > 30*time.Second {
						delete(recentEdits, id)
					}
				}
				editsMutex.Unlock()
				lastCleanup = time.Now()
			}()
		} else {
			cleanupMutex.Unlock()
		}

		// 提取编辑后的内容和媒体
		editedContent := m.Content

		var editedAttachmentURLs []string
		for _, attachment := range m.Attachments {
			editedAttachmentURLs = append(editedAttachmentURLs, attachment.URL)
		}

		editedAttachmentsJSON := ""
		if len(editedAttachmentURLs) > 0 {
			if jsonData, err := json.Marshal(editedAttachmentURLs); err == nil {
				editedAttachmentsJSON = string(jsonData)
			}
		}

		// 尝试获取原始内容
		originalContent, originalAttachments := getOriginalMessageContent(s, messageID, m.ChannelID, editedContent, editedAttachmentsJSON)

		// 创建消息编辑记录
		edit := models.MessageEdit{
			MessageID:           messageID,
			GuildID:             guildIDInt,
			ChannelID:           channelID,
			OriginalContent:     originalContent,
			EditedContent:       editedContent,
			OriginalAttachments: originalAttachments,
			EditedAttachments:   editedAttachmentsJSON,
			EditTimestamp:       time.Now().Unix(),
		}

		// 将编辑记录插入数据库
		if err := globalMessageCollector.messageDB.InsertMessageEdit(edit); err != nil {
			log.Printf("插入消息编辑记录失败 %d: %v", messageID, err)
			return
		}

		// 记录成功的消息编辑跟踪
		log.Printf("消息编辑已跟踪: 服务器 %s, 频道 %s, 消息 %s", m.GuildID, m.ChannelID, m.ID)
	}
}

// getOriginalMessageContent 尝试获取原始消息内容
// 首先尝试从 Discord API 获取，如果内容与编辑后内容相同则查数据库
func getOriginalMessageContent(s *discordgo.Session, messageID int64, channelID, editedContent, editedAttachments string) (string, string) {
	// 首先立刻尝试从 Discord API 获取原始消息（CDN 刷新前）
	discordMessage, err := s.ChannelMessage(channelID, fmt.Sprintf("%d", messageID))
	if err == nil && discordMessage != nil {
		// 将 API 获取的媒体转换为 JSON 格式以便比较
		var apiAttachmentURLs []string
		for _, attachment := range discordMessage.Attachments {
			apiAttachmentURLs = append(apiAttachmentURLs, attachment.URL)
		}

		apiAttachmentsJSON := ""
		if len(apiAttachmentURLs) > 0 {
			if jsonData, err := json.Marshal(apiAttachmentURLs); err == nil {
				apiAttachmentsJSON = string(jsonData)
			}
		}

		// 如果 API 获取的内容和编辑后内容相同，说明 CDN 已刷新，需要查数据库
		if discordMessage.Content == editedContent && apiAttachmentsJSON == editedAttachments {
			log.Printf("API 内容与编辑后内容相同，尝试从数据库获取原始内容 - 消息ID: %d", messageID)
			
			// 查数据库获取原始内容
			if globalMessageCollector != nil {
				if originalMessage, dbErr := globalMessageCollector.messageDB.GetMessage(messageID); dbErr == nil && originalMessage != nil {
					log.Printf("从数据库成功获取原始内容 - 消息ID: %d", messageID)
					return originalMessage.MessageContent, originalMessage.Attachments
				} else {
					log.Printf("数据库查询失败或未找到原始内容 - 消息ID: %d, 错误: %v", messageID, dbErr)
				}
			}
		} else {
			// API 内容与编辑后内容不同，说明 API 还有原始内容
			log.Printf("从 API 获取到原始内容 - 消息ID: %d", messageID)
			return discordMessage.Content, apiAttachmentsJSON
		}
	} else {
		log.Printf("Discord API 获取消息失败 - 消息ID: %d, 错误: %v", messageID, err)
	}

	// 无法获取原始内容，只记录编辑后的内容
	log.Printf("无法获取消息 %d 的原始内容，原始内容留空", messageID)
	return "", ""
}

