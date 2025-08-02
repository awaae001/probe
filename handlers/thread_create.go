package handlers

import (
	"discord-bot/database"
	"discord-bot/models"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// ThreadCreateHandler handles the THREAD_CREATE event.
func ThreadCreateHandler(s *discordgo.Session, t *discordgo.ThreadCreate) {
	// 1. Load configurations
	var scanningConfig models.ScanningConfig
	if err := viper.UnmarshalKey(t.GuildID, &scanningConfig); err != nil {
		// This can happen if the guild is not in scanning_config.json, which is fine.
		return
	}

	var threadConfig models.ThreadConfig
	if err := viper.Unmarshal(&threadConfig); err != nil {
		log.Printf("Error unmarshalling thread config: %v", err)
		return
	}

	// 2. Check if the channel is monitored
	isMonitored := false
	for _, category := range scanningConfig[t.GuildID].Data {
		if t.ParentID == category.ID {
			isMonitored = true
			break
		}
		for _, channelID := range category.ChannelID {
			if t.ParentID == channelID {
				isMonitored = true
				break
			}
		}
		if isMonitored {
			break
		}
	}

	if !isMonitored {
		// log.Printf("Thread created in unmonitored channel %s, ignoring.", t.ParentID)
		return
	}

	// 3. Get guild-specific thread config
	guildThreadConfig, ok := threadConfig[t.GuildID]
	if !ok {
		log.Printf("No thread configuration found for guild %s", t.GuildID)
		return
	}

	// 4. Initialize database
	if err := database.InitDB(guildThreadConfig.Database); err != nil {
		log.Printf("Failed to initialize database for guild %s: %v", t.GuildID, err)
		return
	}

	// 5. Create table if not exists
	if err := database.CreateTableForChannel(database.DB, guildThreadConfig.TableName); err != nil {
		log.Printf("Error creating table %s: %v", guildThreadConfig.TableName, err)
		return
	}

	// 6. Get the first message of the thread
	// The first message has the same ID as the thread itself.
	firstMessage, err := s.ChannelMessage(t.ID, t.ID)
	if err != nil {
		log.Printf("Error getting first message for thread %s: %v", t.ID, err)
		return
	}

	// 7. Populate the Post model
	var tagNames []string
	if t.AppliedTags != nil {
		for _, tagID := range t.AppliedTags {
			tagNames = append(tagNames, string(tagID))
		}
	}

	content := firstMessage.Content
	runes := []rune(content)
	if len(runes) > 512 {
		content = string(runes[:512])
	}

	var coverImageURL string
	if len(firstMessage.Attachments) > 0 {
		coverImageURL = firstMessage.Attachments[0].URL
	}

	post := models.Post{
		ThreadID:        t.ID,
		ChannelID:       t.ParentID,
		Title:           t.Name,
		Author:          firstMessage.Author.Username,
		AuthorID:        firstMessage.Author.ID,
		Content:         content,
		Tags:            strings.Join(tagNames, ","),
		MessageCount:    0,
		Timestamp:       firstMessage.Timestamp.Unix(),
		CoverImageURL:   coverImageURL,
		TotalReactions:  0,
		UniqueReactions: 0,
	}

	// 8. Insert the post into the database
	if err := database.InsertPost(database.DB, post, guildThreadConfig.TableName); err != nil {
		log.Printf("Error inserting post %s into database: %v", post.ThreadID, err)
	} else {
		log.Printf("Successfully saved new thread: %s to table %s", post.ThreadID, guildThreadConfig.TableName)
	}
}
