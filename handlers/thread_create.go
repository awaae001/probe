package handlers

import (
	"discord-bot/database"
	"discord-bot/models"
	"discord-bot/utils"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// ThreadCreateHandler handles the THREAD_CREATE event.
func ThreadCreateHandler(s *discordgo.Session, t *discordgo.ThreadCreate) {
	log.Printf("New thread created event received for thread ID %s in guild %s", t.ID, t.GuildID)
	// 1. Load configurations
	var scanningConfig models.GuildConfig
	if err := viper.UnmarshalKey(t.GuildID, &scanningConfig); err != nil {
		log.Printf("Error unmarshalling scanning config for guild %s: %v", t.GuildID, err)
		return
	}
	var threadConfig models.ThreadConfig
	if err := viper.Unmarshal(&threadConfig); err != nil {
		log.Printf("Error unmarshalling thread config: %v", err)
		return
	}

	// 2. Check if the thread's category is monitored and the channel is not excluded.
	forumChannel, err := s.Channel(t.ParentID)
	if err != nil {
		log.Printf("Error getting details for forum channel %s: %v", t.ParentID, err)
		return
	}
	categoryID := forumChannel.ParentID
	log.Printf("Thread created in channel %s (Category ID: %s)", t.ParentID, categoryID)

	isMonitored := false
	if categoryConfig, ok := scanningConfig.Data[categoryID]; ok {
		log.Printf("Thread's category '%s' (%s) is configured for monitoring.", categoryConfig.CategoryName, categoryID)

		// Check if the specific forum channel is in the exclusion list.
		isExcluded := false
		for _, excludedID := range categoryConfig.ChannelID {
			if t.ParentID == excludedID {
				isExcluded = true
				break
			}
		}

		if !isExcluded {
			isMonitored = true
			log.Printf("Channel %s is not in the exclusion list. Proceeding...", t.ParentID)
		} else {
			log.Printf("Channel %s is in the exclusion list for category %s. Ignoring.", t.ParentID, categoryID)
		}
	} else {
		log.Printf("Thread's category %s is not configured for monitoring. Ignoring.", categoryID)
	}

	if !isMonitored {
		return
	}

	// 3. Get guild-specific thread config
	guildThreadConfig, ok := threadConfig[t.GuildID]
	if !ok {
		log.Printf("No thread configuration found for guild %s", t.GuildID)
		return
	}

	// 4. Initialize independent database
	db, err := database.InitThreadDB(guildThreadConfig.Database)
	if err != nil {
		log.Printf("Failed to initialize independent database for guild %s: %v", t.GuildID, err)
		return
	}
	defer db.Close()

	// 5. Create table if not exists
	if err := database.CreateTableForChannel(db, guildThreadConfig.TableName); err != nil {
		log.Printf("Error creating table %s: %v", guildThreadConfig.TableName, err)
		return
	}

	// 6. Get the first message of the thread
	// The first message has the same ID as the thread itself.
	var firstMessage *discordgo.Message
	for i := 0; i < 3; i++ {
		firstMessage, err = s.ChannelMessage(t.ID, t.ID)
		if err == nil {
			break
		}
		if restErr, ok := err.(*discordgo.RESTError); ok && restErr.Response.StatusCode == 404 {
			utils.Warn("ThreadCreate", "GetFirstMessage", fmt.Sprintf("Got 404 for first message in thread %s, retrying in 10s... (%d/3)", t.ID, i+1))
			time.Sleep(10 * time.Second)
			continue
		}
		utils.Error("ThreadCreate", "GetFirstMessage", fmt.Sprintf("Error getting first message for thread %s: %v", t.ID, err))
		return
	}

	if err != nil {
		utils.Error("ThreadCreate", "GetFirstMessage", fmt.Sprintf("Failed to get first message for thread %s after 3 retries: %v", t.ID, err))
		return
	}

	// 7. Populate the Post model
	// Get the parent channel (forum) to access its tags
	parentChannel, err := s.Channel(t.ParentID)
	if err != nil {
		utils.Warn("ThreadCreate", "GetParentChannel", fmt.Sprintf("Error getting parent channel %s: %v. Continuing without tags.", t.ParentID, err))
		// We can continue without tags if this fails
	}

	// Create a map of tag IDs to tag names for efficient lookup
	tagMap := make(map[string]string)
	if parentChannel != nil {
		for _, tag := range parentChannel.AvailableTags {
			tagMap[tag.ID] = tag.Name
		}
	}

	// Populate tagNames using the map
	var tagNames []string
	if t.AppliedTags != nil {
		for _, tagID := range t.AppliedTags {
			if name, ok := tagMap[tagID]; ok {
				tagNames = append(tagNames, name)
			}
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
	if err := database.InsertPost(db, post, guildThreadConfig.TableName); err != nil {
		utils.Error("ThreadCreate", "AddPostToDB", fmt.Sprintf("Error adding post to database for thread %s: %v", t.ID, err))
		return
	}

	log.Printf("Successfully added post for thread %s to database.", t.ID)

	// 9. Add a reaction to the first message to indicate it's been processed
	if err := s.MessageReactionAdd(t.ID, firstMessage.ID, "✅"); err != nil {
		utils.Warn("ThreadCreate", "AddReaction", fmt.Sprintf("Error adding reaction to thread %s: %v", t.ID, err))
	}
}
