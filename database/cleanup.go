package database

import (
	"fmt"
	"log"
	"time"

	"discord-bot/models"
	"discord-bot/utils"

	"github.com/spf13/viper"
)

// CleanupOldPosts iterates through the thread configurations, connects to each database,
// and deletes posts older than 31 days.
func CleanupOldPosts() {
	log.Println("Starting cleanup of old posts...")

	var threadConfig models.ThreadConfig
	if err := viper.UnmarshalKey("thread_config", &threadConfig.ThreadConfig); err != nil {
		log.Printf("Error unmarshalling thread_config: %v", err)
		return
	}

	for guildID, config := range threadConfig.ThreadConfig {
		log.Printf("Cleaning up posts for guild %s (%s)", config.Name, guildID)

		db, err := InitThreadDB(config.Database)
		if err != nil {
			log.Printf("Error connecting to database for guild %s: %v", guildID, err)
			continue
		}
		defer db.Close()

		thirtyOneDaysAgo := time.Now().AddDate(0, 0, -31).Unix()

		query := fmt.Sprintf("DELETE FROM %s WHERE timestamp < ?", config.TableName)
		stmt, err := db.Prepare(query)
		if err != nil {
			log.Printf("Error preparing delete statement for guild %s: %v", guildID, err)
			continue
		}
		defer stmt.Close()

		res, err := stmt.Exec(thirtyOneDaysAgo)
		if err != nil {
			log.Printf("Error executing delete statement for guild %s: %v", guildID, err)
			continue
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			log.Printf("Error getting rows affected for guild %s: %v", guildID, err)
			continue
		}

		log.Printf("Successfully cleaned up %d old posts for guild %s", rowsAffected, guildID)
		utils.Info("CleanupOldPosts", "Cleanup", fmt.Sprintf("Successfully cleaned up %d old posts for guild %s", rowsAffected, guildID))
	}

	log.Println("Finished cleanup of old posts.")
}
