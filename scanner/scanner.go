package scanner

import (
	"discord-bot/database"
	"discord-bot/models"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// StartScanning initiates the scanning process based on the configuration.
func StartScanning(s *discordgo.Session) {
	log.Println("Starting the scanning process...")

	// Manually decode the settings to avoid conflicts with other config files.
	allSettings := viper.AllSettings()
	scanningConfig := make(models.ScanningConfig)

	for key, value := range allSettings {
		// A simple check to see if the key is a digit-only string (like a Discord ID)
		if _, err := strconv.ParseUint(key, 10, 64); err != nil {
			continue // Skip keys that are not snowflakes (e.g., "bot")
		}

		var guildConf models.GuildConfig
		// Use mapstructure to decode the generic map into our struct
		if err := mapstructure.Decode(value, &guildConf); err != nil {
			log.Printf("Could not decode config for key %s: %v", key, err)
			continue
		}
		scanningConfig[key] = guildConf
	}

	if len(scanningConfig) == 0 {
		log.Println("No valid guild configurations found in scanning_config.json.")
		return
	}

	for guildID, guildConfig := range scanningConfig {
		log.Printf("Scanning guild: %s (%s)", guildConfig.Name, guildID)

		// Initialize database for the guild
		if err := database.InitDB(guildConfig.DBPath); err != nil {
			log.Printf("Failed to initialize database for guild %s: %v", guildID, err)
			continue
		}

		for categoryID, categoryData := range guildConfig.Data {
			log.Printf("Scanning category: %s (%s)", categoryData.CategoryName, categoryID)

			channels, err := s.GuildChannels(guildID)
			if err != nil {
				log.Printf("Failed to get channels for guild %s: %v", guildID, err)
				continue
			}

			for _, channel := range channels {
				if channel.Type == discordgo.ChannelTypeGuildForum && channel.ParentID == categoryID {
					// Determine if this channel should be scanned.
					shouldScan := false
					if len(categoryData.ChannelID) == 0 {
						// If no channels are specified, scan all forums in the category.
						shouldScan = true
					} else {
						// If channels are specified, only scan if this channel is in the list.
						for _, id := range categoryData.ChannelID {
							if id == channel.ID {
								shouldScan = true
								break
							}
						}
					}

					if shouldScan {
						log.Printf("Found forum channel to scan: %s (%s)", channel.Name, channel.ID)
						scanForumChannel(s, channel.ID)
					}
				}
			}
		}
	}
	log.Println("Scanning process finished.")
}

// scanForumChannel scans a specific forum channel for threads, including active and archived ones.
func scanForumChannel(s *discordgo.Session, channelID string) {
	// Create a table for the channel if it doesn't exist
	if err := database.CreateTableForChannel(channelID); err != nil {
		log.Printf("Could not create table for channel %s: %v", channelID, err)
		return
	}

	processedThreads := make(map[string]bool)

	// 1. Get active threads in the channel
	ch, err := s.Channel(channelID)
	if err != nil {
		log.Printf("Failed to get channel details for %s: %v", channelID, err)
		return
	}
	guildID := ch.GuildID

	activeThreads, err := s.GuildThreadsActive(guildID)
	if err != nil {
		log.Printf("Failed to get active threads for guild %s: %v", guildID, err)
	} else {
		for _, thread := range activeThreads.Threads {
			if thread.ParentID == channelID {
				if !processedThreads[thread.ID] {
					processThread(s, thread)
					processedThreads[thread.ID] = true
				}
			}
		}
	}

	// 2. Get archived threads in the channel
	var before *time.Time
	for {
		archived, err := s.ThreadsArchived(channelID, before, 100)
		if err != nil {
			log.Printf("Failed to get archived threads for channel %s: %v", channelID, err)
			break
		}

		if len(archived.Threads) == 0 {
			break
		}

		for _, thread := range archived.Threads {
			if !processedThreads[thread.ID] {
				processThread(s, thread)
				processedThreads[thread.ID] = true
			}
			if thread.ThreadMetadata != nil {
				// The API expects the timestamp of the last thread to paginate.
				// We need to assign the address of the ArchiveTimestamp.
				t := thread.ThreadMetadata.ArchiveTimestamp
				before = &t
			}
		}

		if !archived.HasMore {
			break
		}
	}
}

// processThread processes a single thread and saves it to the database.
func processThread(s *discordgo.Session, thread *discordgo.Channel) {
	// The thread's ID is the same as the ID of the message that started it.
	// We need to fetch that specific message from the thread channel itself.
	firstMessage, err := s.ChannelMessage(thread.ID, thread.ID)
	if err != nil {
		if restErr, ok := err.(*discordgo.RESTError); ok && restErr.Response.StatusCode == 404 {
			log.Printf("Thread %s not found (404), skipping.", thread.ID)
		} else {
			log.Printf("Error getting first message for thread %s: %v", thread.ID, err)
		}
		return
	}

	// Extract content and image URL
	content := firstMessage.Content
	runes := []rune(content)
	if len(runes) > 512 {
		content = string(runes[:512])
	}

	var firstImageURL string
	if len(firstMessage.Attachments) > 0 {
		firstImageURL = firstMessage.Attachments[0].URL
	}

	// Parse creation timestamp from thread ID
	creationTimestamp, err := discordgo.SnowflakeTimestamp(thread.ID)
	if err != nil {
		log.Printf("Could not parse creation timestamp for thread %s: %v", thread.ID, err)
		creationTimestamp = time.Now() // Fallback
	}

	// The LastMessageID can be used to get a rough idea of the last message time.
	var lastMessageTime time.Time
	if thread.LastMessageID != "" {
		lastMessageTime, err = discordgo.SnowflakeTimestamp(thread.LastMessageID)
		if err != nil {
			lastMessageTime = creationTimestamp // Fallback
		}
	} else {
		lastMessageTime = creationTimestamp
	}

	// Create ForumPost object
	post := models.ForumPost{
		AuthorID:        firstMessage.Author.ID,
		ThreadID:        thread.ID,
		Title:           thread.Name,
		Content:         content,
		FirstImageURL:   firstImageURL,
		MessageCount:    thread.MessageCount,
		CreationDate:    creationTimestamp.Unix(),
		LastMessageTime: lastMessageTime.Unix(),
		TagID:           strings.Join(thread.AppliedTags, ","),
		ChannelID:       thread.ParentID,
	}

	// Save the post to the database
	if err := database.SavePost(post); err != nil {
		log.Printf("Failed to save post %s: %v", post.ThreadID, err)
	} else {
		log.Printf("Successfully saved post: %s", post.Title)
	}
}
