package bot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"discord-bot/command"
	"discord-bot/config"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// Bot encapsulates the bot's state.
type Bot struct {
	Session *discordgo.Session
}

// NewBot creates and initializes a new Bot instance.
func NewBot() (*Bot, error) {
	config.LoadConfig()
	token := viper.GetString("BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("no bot token provided")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds | discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessageReactions

	return &Bot{
		Session: dg,
	}, nil
}

// Start opens the bot's session and registers handlers.
func (b *Bot) Start(registerHandlers func(*Bot)) error {
	registerHandlers(b)

	err := b.Session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %w", err)
	}

	// Register slash commands
	commandDefs := command.GetCommandDefinitions()
	guildIDs := viper.GetStringSlice("commands.allowguils")

	if viper.GetBool("bot.commands.clear_on_startup") {
		log.Println("Clearing existing commands...")
		// Clear global commands
		existingCommands, err := b.Session.ApplicationCommands(b.Session.State.User.ID, "")
		if err != nil {
			log.Printf("Could not fetch global commands: %v", err)
		} else {
			for _, cmd := range existingCommands {
				err := b.Session.ApplicationCommandDelete(b.Session.State.User.ID, "", cmd.ID)
				if err != nil {
					log.Printf("Cannot delete global command '%v': %v", cmd.Name, err)
				}
			}
		}

		// Clear commands in specified guilds
		for _, guildID := range guildIDs {
			existingCommands, err := b.Session.ApplicationCommands(b.Session.State.User.ID, guildID)
			if err != nil {
				log.Printf("Could not fetch commands for guild %s: %v", guildID, err)
				continue
			}
			for _, cmd := range existingCommands {
				err := b.Session.ApplicationCommandDelete(b.Session.State.User.ID, guildID, cmd.ID)
				if err != nil {
					log.Printf("Cannot delete command '%v' in guild %s: %v", cmd.Name, guildID, err)
				}
			}
		}
		log.Println("Finished clearing commands.")
	}

	log.Println("Registering commands...")
	for _, guildID := range guildIDs {
		for _, cmdDef := range commandDefs {
			_, err := b.Session.ApplicationCommandCreate(b.Session.State.User.ID, guildID, cmdDef)
			if err != nil {
				log.Printf("Cannot create '%v' command in guild %s: %v", cmdDef.Name, guildID, err)
			} else {
				log.Printf("Successfully created '%v' command in guild %s.", cmdDef.Name, guildID)
			}
		}
	}
	log.Println("Finished registering commands.")

	startScheduler(b.Session)

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	return nil
}

// Stop gracefully closes the bot's session.
func (b *Bot) Stop() {
	stopScheduler()
	if b.Session != nil {
		b.Session.Close()
	}
	fmt.Println("Bot stopped gracefully.")
}

// Run is the main entry point for the bot application.
func Run(registerHandlers func(*Bot)) {
	bot, err := NewBot()
	if err != nil {
		log.Fatalf("Error initializing bot: %v", err)
	}

	if err := bot.Start(registerHandlers); err != nil {
		log.Fatalf("Error starting bot: %v", err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	bot.Stop()
}
