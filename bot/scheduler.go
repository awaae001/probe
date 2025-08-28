package bot

import (
	"discord-bot/database"
	"log"

	"discord-bot/models"
	"discord-bot/scanner"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
)

var c *cron.Cron

// getScanningConfig extracts scanning configuration from Viper.
func getScanningConfig() models.ScanningConfig {
	scanningConfig := make(models.ScanningConfig)
	allSettings := viper.AllSettings()

	for key, value := range allSettings {
		if key == "db_file_path" || key == "data" {
			continue
		}

		if configMap, ok := value.(map[string]interface{}); ok {
			var guildConfig models.GuildConfig
			if name, ok := configMap["name"].(string); ok {
				guildConfig.Name = name
			}
			if guildsID, ok := configMap["guilds_id"].(string); ok {
				guildConfig.GuildsID = guildsID
			}
			if dbPath, ok := configMap["db_path"].(string); ok {
				guildConfig.DBPath = dbPath
			}
			if data, ok := configMap["data"].(map[string]interface{}); ok {
				guildConfig.Data = make(map[string]models.CategoryData)
				for catKey, catValue := range data {
					if catMap, ok := catValue.(map[string]interface{}); ok {
						var categoryData models.CategoryData
						if categoryName, ok := catMap["category_name"].(string); ok {
							categoryData.CategoryName = categoryName
						}
						if id, ok := catMap["id"].(string); ok {
							categoryData.ID = id
						}
						if channelID, ok := catMap["channel_id"].([]interface{}); ok {
							for _, chID := range channelID {
								if chIDStr, ok := chID.(string); ok {
									categoryData.ChannelID = append(categoryData.ChannelID, chIDStr)
								}
							}
						}
						guildConfig.Data[catKey] = categoryData
					}
				}
			}

			if guildConfig.Name != "" && guildConfig.GuildsID != "" && guildConfig.DBPath != "" {
				scanningConfig[key] = guildConfig
			}
		}
	}
	return scanningConfig
}

// startScheduler starts the cron jobs.
func startScheduler(s *discordgo.Session) {
	log.Println("Initializing scheduler...")
	c = cron.New()
	scanningConfig := getScanningConfig() // Get config once

	// Hourly scan
	_, err := c.AddFunc("@hourly", func() {
		log.Println("Running hourly scan...")
		scanner.StartScanning(s, scanningConfig, false) // Incremental scan
	})
	if err != nil {
		log.Fatalf("Could not set up cron job: %v", err)
	}

	// Daily member stats update
	_, err = c.AddFunc("@daily", func() {
		log.Println("Running daily member stats update...")
		database.ScheduledUpdate(s)
	})
	if err != nil {
		log.Fatalf("Could not set up daily member stats cron job: %v", err)
	}

	c.Start()
	log.Println("Cron jobs scheduled.")

	// Perform an initial scan on startup
	if viper.GetBool("bot.ScanAtStartup") {
		go func() {
			log.Println("Performing initial scan on startup...")
			scanner.StartScanning(s, scanningConfig, true) // Full scan
		}()
	} else {
		log.Println("Skipping initial scan on startup as per configuration.")
	}
}

// stopScheduler stops the cron jobs.
func stopScheduler() {
	if c != nil {
		c.Stop()
		log.Println("Scheduler stopped.")
	}
}
