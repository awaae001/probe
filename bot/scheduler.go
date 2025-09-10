package bot

import (
	"log"
	"math/rand"
	"time"

	"discord-bot/models"
	"discord-bot/scanner"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
)

var c *cron.Cron

// getScanningConfig extracts scanning configuration from Viper.
func getScanningConfig() models.ScanningConfig {
	var fileConfig models.ScanningFileConfig
	if err := viper.Unmarshal(&fileConfig); err != nil {
		log.Printf("Error unmarshalling config for scheduler: %v", err)
		return make(models.ScanningConfig)
	}
	return fileConfig.ScanningConfig
}

// startScheduler starts the cron jobs.
func startScheduler(s *discordgo.Session) {
	log.Println("Initializing scheduler...")
	c = cron.New()
	scanningConfig := getScanningConfig() // Get config once
	rand.Seed(time.Now().UnixNano())

	// Hourly scan
	_, err := c.AddFunc("@every 10m", func() {
		log.Println("Running hourly scan...")
		scanner.StartScanning(s, scanningConfig, false) // Incremental scan

		// Set random status
		if rand.Intn(8) == 0 { // 1/8 chance to be online
			err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
				Status: string(discordgo.StatusOnline),
			})
			if err != nil {
				log.Printf("Error setting bot status to online: %v", err)
			} else {
				log.Println("Bot status set to Online.")
			}
		} else {
			err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
				Status: string(discordgo.StatusInvisible),
			})
			if err != nil {
				log.Printf("Error setting bot status to offline: %v", err)
			} else {
				log.Println("Bot status set to Offline.")
			}
		}
	})
	if err != nil {
		log.Fatalf("Could not set up cron job: %v", err)
	}

	// Daily member stats update has been removed.

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
