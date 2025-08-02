package handlers

import (
	"discord-bot/database"
	"discord-bot/models"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// ThreadDeleteHandler handles the THREAD_DELETE event.
func ThreadDeleteHandler(s *discordgo.Session, t *discordgo.ThreadDelete) {
	log.Printf("Thread delete event received for thread ID %s in guild %s", t.ID, t.GuildID)
	var threadConfig models.ThreadConfig
	if err := viper.Unmarshal(&threadConfig); err != nil {
		log.Printf("Error unmarshalling thread config: %v", err)
		return
	}

	guildThreadConfig, ok := threadConfig[t.GuildID]
	if !ok {
		log.Printf("No thread configuration found for guild %s", t.GuildID)
		return
	}

	db, err := database.InitThreadDB(guildThreadConfig.Database)
	if err != nil {
		log.Printf("Failed to initialize independent database for guild %s: %v", t.GuildID, err)
		return
	}
	defer db.Close()

	tableName := "channel_" + t.ParentID
	if err := database.UpdatePostStatus(db, tableName, t.ID, "deleted"); err != nil {
		log.Printf("Error updating status for thread %s in table %s: %v", t.ID, tableName, err)
	} else {
		log.Printf("Successfully marked thread %s as deleted in table %s", t.ID, tableName)
	}
}
