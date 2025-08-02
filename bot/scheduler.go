package bot

import (
	"log"

	"discord-bot/models"
	"discord-bot/scanner"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
)

var c *cron.Cron

// startScheduler starts the cron jobs.
func startScheduler(s *discordgo.Session) {
	log.Println("Initializing scheduler...")
	c = cron.New()
	_, err := c.AddFunc("@hourly", func() {
		log.Println("Running hourly scan...")
		var scanningConfig models.ScanningConfig
		if err := viper.Unmarshal(&scanningConfig); err != nil {
			log.Printf("Error unmarshalling scanning config for scheduled scan: %v", err)
			return
		}
		scanner.StartScanning(s, scanningConfig, false) // Incremental scan
	})
	if err != nil {
		log.Fatalf("Could not set up cron job: %v", err)
	}
	c.Start()
	log.Println("Cron job scheduled to run hourly.")

	// Perform an initial scan on startup based on config.
	if viper.GetBool("bot.ScanAtStartup") {
		go func() {
			log.Println("Performing initial scan on startup...")
			var scanningConfig models.ScanningConfig
			if err := viper.Unmarshal(&scanningConfig); err != nil {
				log.Printf("Error unmarshalling scanning config for startup scan: %v", err)
				return
			}
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
