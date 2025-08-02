package bot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"discord-bot/config"
	"discord-bot/handlers"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

var (
	dg *discordgo.Session
)

// Start a Discord bot session.
func Start() {
	config.LoadConfig()

	token := viper.GetString("BOT_TOKEN")
	if token == "" {
		log.Fatal("No bot token provided. Please set the BOT_TOKEN in your .env or config file.")
	}

	var err error
	dg, err = discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	for _, handler := range handlers.Handlers {
		dg.AddHandler(handler)
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds | discordgo.IntentsGuildMembers

	err = dg.Open()
	if err != nil {
		log.Fatalf("Error opening connection: %v", err)
	}

	startScheduler(dg)

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	Stop()
}

// Stop the Discord bot session.
func Stop() {
	stopScheduler()
	if dg != nil {
		dg.Close()
	}
	fmt.Println("Bot stopped gracefully.")
}
