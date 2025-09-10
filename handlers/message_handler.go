package handlers

import (
	"context"
	"discord-bot/bot"
	"discord-bot/database"
	"discord-bot/models"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strconv"
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

// MessageUpdateHandler handles Discord message update events
func MessageUpdateHandler(b *bot.Bot) func(s *discordgo.Session, m *discordgo.MessageUpdate) {
	return func(s *discordgo.Session, m *discordgo.MessageUpdate) {
		// Skip bot messages
		if m.Author != nil && m.Author.Bot {
			return
		}

		// Validate guild and channel
		_, valid := validateGuildAndChannel(m.GuildID, m.ChannelID)
		if !valid {
			return
		}

		// Convert Discord IDs to int64 (using empty string for userID since it may not be available in updates)
		messageID, _, guildIDInt, _, err := parseDiscordIDs(m.ID, "", m.GuildID, m.ChannelID)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		// Check if BeforeUpdate is available (contains previous content)
		if m.BeforeUpdate == nil {
			// Try to get original content with priority order: immediate API call first, then database
			go func() {
				// 创建5秒超时的上下文
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				
				var originalContent string
				var originalAttachments string
				found := false
				
				// 使用 channel 来接收结果或超时
				resultChan := make(chan bool, 1)
				
				go func() {
					defer func() {
						select {
						case resultChan <- found:
						default:
						}
					}()
					
					// First, try immediate API call (before CDN updates)
					if originalMessage, err := s.ChannelMessage(m.ChannelID, m.ID); err == nil {
						apiContent := originalMessage.Content
						
						// Extract attachment URLs from API
						var apiAttachmentURLs []string
						for _, attachment := range originalMessage.Attachments {
							apiAttachmentURLs = append(apiAttachmentURLs, attachment.URL)
						}
						var apiAttachments string
						if len(apiAttachmentURLs) > 0 {
							if jsonData, err := json.Marshal(apiAttachmentURLs); err == nil {
								apiAttachments = string(jsonData)
							}
						}
						
						// Extract current message attachments for comparison
						var currentAttachmentURLs []string
						for _, attachment := range m.Attachments {
							currentAttachmentURLs = append(currentAttachmentURLs, attachment.URL)
						}
						var currentAttachments string
						if len(currentAttachmentURLs) > 0 {
							if jsonData, err := json.Marshal(currentAttachmentURLs); err == nil {
								currentAttachments = string(jsonData)
							}
						}
						
						// Check if API content is different from current content
						if apiContent != m.Content || apiAttachments != currentAttachments {
							originalContent = apiContent
							originalAttachments = apiAttachments
							found = true
							log.Printf("Retrieved original message content from immediate API call for message %s", m.ID)
						} else {
							// API content is same as current content, try database
							log.Printf("API content same as current content, checking database for message %s", m.ID)
							if originalMessage, err := globalMessageCollector.messageDB.GetMessage(messageID); err == nil {
								// Check if database content is also the same as current content
								if originalMessage.MessageContent == m.Content && originalMessage.Attachments == currentAttachments {
									// All content is identical, set error message
									originalContent = "错误：原始内容无法获取"
									originalAttachments = ""
									found = true
									log.Printf("All content identical for message %s, using error message", m.ID)
								} else {
									// Database has different content, use it
									originalContent = originalMessage.MessageContent
									originalAttachments = originalMessage.Attachments
									found = true
									log.Printf("Retrieved original message content from database for message %s", m.ID)
								}
							} else {
								// Database lookup failed, set error message
								originalContent = "错误：原始内容无法获取"
								originalAttachments = ""
								found = true
								log.Printf("Database lookup failed for message %s, using error message", m.ID)
							}
						}
					} else {
						// API call failed, try database
						log.Printf("API call failed, checking database for message %s", m.ID)
						if originalMessage, err := globalMessageCollector.messageDB.GetMessage(messageID); err == nil {
							originalContent = originalMessage.MessageContent
							originalAttachments = originalMessage.Attachments
							found = true
							log.Printf("Retrieved original message content from database for message %s", m.ID)
						} else {
							// Database lookup failed after API call failed, set error message
							originalContent = "错误：原始内容无法获取"
							originalAttachments = ""
							found = true
							log.Printf("Both API and database lookup failed for message %s, using error message", m.ID)
						}
					}
				}()
				
				// 等待结果或超时
				select {
				case <-resultChan:
					if found {
						// Create a synthetic BeforeUpdate message
						m.BeforeUpdate = &discordgo.Message{
							ID:          m.ID,
							Content:     originalContent,
							Attachments: []*discordgo.MessageAttachment{},
						}
						
						// Parse back attachments if we have them
						if originalAttachments != "" {
							var urls []string
							if err := json.Unmarshal([]byte(originalAttachments), &urls); err == nil {
								for _, url := range urls {
									m.BeforeUpdate.Attachments = append(m.BeforeUpdate.Attachments, &discordgo.MessageAttachment{URL: url})
								}
							}
						}
						
						// Re-process the edit with the retrieved original content
						processMessageEdit(s, m, guildIDInt, messageID)
					}
				case <-ctx.Done():
					// 超时处理
					log.Printf("消息内容获取超时 (5秒): Guild %s, Message %s", m.GuildID, m.ID)
					
					// 创建带有超时错误信息的 BeforeUpdate
					m.BeforeUpdate = &discordgo.Message{
						ID:          m.ID,
						Content:     "错误：获取超时",
						Attachments: []*discordgo.MessageAttachment{},
					}
					
					// 处理超时情况下的编辑记录
					processMessageEdit(s, m, guildIDInt, messageID)
				}
			}()
			return
		}

		// Process the message edit
		processMessageEdit(s, m, guildIDInt, messageID)
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

// processMessageEdit handles the actual message edit processing logic
func processMessageEdit(s *discordgo.Session, m *discordgo.MessageUpdate, guildIDInt, messageID int64) {
	// Extract attachment URLs from before and after
	var previousAttachmentURLs []string
	for _, attachment := range m.BeforeUpdate.Attachments {
		previousAttachmentURLs = append(previousAttachmentURLs, attachment.URL)
	}

	var newAttachmentURLs []string
	for _, attachment := range m.Attachments {
		newAttachmentURLs = append(newAttachmentURLs, attachment.URL)
	}

	// Convert attachments to JSON strings
	previousAttachmentsJSON := ""
	if len(previousAttachmentURLs) > 0 {
		if jsonData, err := json.Marshal(previousAttachmentURLs); err == nil {
			previousAttachmentsJSON = string(jsonData)
		}
	}

	newAttachmentsJSON := ""
	if len(newAttachmentURLs) > 0 {
		if jsonData, err := json.Marshal(newAttachmentURLs); err == nil {
			newAttachmentsJSON = string(jsonData)
		}
	}

	// Create message edit record
	messageEdit := models.MessageEdit{
		MessageID:           messageID,
		GuildID:             guildIDInt,
		EditTimestamp:       time.Now().Unix(),
		PreviousContent:     m.BeforeUpdate.Content,
		NewContent:          m.Content,
		PreviousAttachments: previousAttachmentsJSON,
		NewAttachments:      newAttachmentsJSON,
	}

	// Insert edit record and update original message
	if err := globalMessageCollector.messageDB.InsertMessageEdit(messageEdit, m.Content, newAttachmentsJSON); err != nil {
		log.Printf("Error recording message edit for message %d: %v", messageID, err)
		return
	}

	// Log successful message edit tracking
	log.Printf("Message edit tracked: Guild %s, Channel %s, Message %s", m.GuildID, m.ChannelID, m.ID)
}