package thread

import (
	"discord-bot/database"
	"discord-bot/models"
	"discord-bot/utils"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// ThreadDeleteHandler handles the THREAD_DELETE event.
func ThreadDeleteHandler(s *discordgo.Session, t *discordgo.ThreadDelete) {
	log.Printf("Thread delete event received for thread ID %s in guild %s", t.ID, t.GuildID)
	// Load the scanning configuration for the specific guild.
	var scanningConfig models.GuildConfig
	if err := viper.UnmarshalKey("scanning_config."+t.GuildID, &scanningConfig); err != nil {
		log.Printf("Error unmarshalling scanning config for guild %s: %v", t.GuildID, err)
		return
	}

	// Check if a database path is configured.
	if scanningConfig.DBPath == "" {
		log.Printf("No scanning database configuration (DBPath) found for guild %s. Ignoring delete event.", t.GuildID)
		return
	}

	// Initialize the database connection using the path from scanning_config.
	db, err := database.InitThreadDB(scanningConfig.DBPath)
	if err != nil {
		log.Printf("Failed to initialize scanning database for guild %s: %v", t.GuildID, err)
		return
	}
	defer db.Close()

	tableName := "channel_" + t.ParentID
	if err := database.UpdatePostStatus(db, tableName, t.ID, "deleted"); err != nil {
		details := fmt.Sprintf("Error updating status for thread %s in table %s: %v", t.ID, tableName, err)
		utils.Error("ThreadDelete", "DatabaseUpdate", details)
	} else {
		details := fmt.Sprintf("Successfully marked thread %s as deleted in table %s", t.ID, tableName)
		utils.Info("ThreadDelete", "DatabaseUpdate", details)
	}
}
