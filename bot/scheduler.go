package bot

import (
	"log"

	"discord-bot/scanner"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

var c *cron.Cron

// startScheduler starts the cron jobs.
func startScheduler(s *discordgo.Session) {
	log.Println("Initializing scheduler...")
	c = cron.New()
	_, err := c.AddFunc("@hourly", func() {
		log.Println("Running hourly scan...")
		scanner.StartScanning(s, false) // Incremental scan
	})
	if err != nil {
		log.Fatalf("Could not set up cron job: %v", err)
	}
	c.Start()
	log.Println("Cron job scheduled to run hourly.")

	// Perform an initial scan on startup.
	go func() {
		log.Println("Performing initial scan...")
		scanner.StartScanning(s, true) // Full scan
	}()
}

// stopScheduler stops the cron jobs.
func stopScheduler() {
	if c != nil {
		c.Stop()
		log.Println("Scheduler stopped.")
	}
}
