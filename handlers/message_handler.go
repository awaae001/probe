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
		// Skip if message collector is not initialized
		if globalMessageCollector == nil {
			return
		}

		// Skip bot messages
		if m.Author.Bot {
			return
		}

		// Check if this guild is configured for message collection
		guildID := m.GuildID
		guildConfig, exists := globalMessageCollector.config.MessageListener.Data[guildID]
		if !exists {
			return
		}

		// Check if this channel is excluded
		if slices.Contains(guildConfig.Exclude, m.ChannelID) {
			return // Skip excluded channels
		}

		// Convert Discord IDs to int64
		messageID, err := strconv.ParseInt(m.ID, 10, 64)
		if err != nil {
			log.Printf("Error parsing message ID %s: %v", m.ID, err)
			return
		}

		userID, err := strconv.ParseInt(m.Author.ID, 10, 64)
		if err != nil {
			log.Printf("Error parsing user ID %s: %v", m.Author.ID, err)
			return
		}

		guildIDInt, err := strconv.ParseInt(guildID, 10, 64)
		if err != nil {
			log.Printf("Error parsing guild ID %s: %v", guildID, err)
			return
		}

		channelID, err := strconv.ParseInt(m.ChannelID, 10, 64)
		if err != nil {
			log.Printf("Error parsing channel ID %s: %v", m.ChannelID, err)
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
		}

		// Insert message into database
		if err := globalMessageCollector.messageDB.InsertMessage(message); err != nil {
			log.Printf("Error inserting message %d: %v", messageID, err)
			return
		}

		// Update status file with current database info
		go func() {
			dbPath, err := globalMessageCollector.messageDB.GetDBPath(guildID)
			if err != nil {
				log.Printf("Error getting DB path for status update: %v", err)
				return
			}

			messageCount, err := globalMessageCollector.messageDB.GetMessageCount(guildIDInt)
			if err != nil {
				log.Printf("Error getting message count for status update: %v", err)
				return
			}

			if err := globalMessageCollector.statusManager.UpdateDatabaseInfo(guildID, dbPath, messageCount); err != nil {
				log.Printf("Error updating database status: %v", err)
			}
		}()

		// Log successful message collection (can be removed in production)
		log.Printf("Message collected: Guild %s, Channel %s, User %s", guildID, m.ChannelID, m.Author.ID)
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