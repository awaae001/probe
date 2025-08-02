package bot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"discord-bot/config"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// Command defines the interface for a bot command.
type Command interface {
	Definition() *discordgo.ApplicationCommand
	Handler(s *discordgo.Session, i *discordgo.InteractionCreate)
	MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate)
}

// Bot encapsulates the bot's state.
type Bot struct {
	Session  *discordgo.Session
	Commands map[string]Command
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
		Session:  dg,
		Commands: make(map[string]Command),
	}, nil
}

// RegisterCommands registers the provided commands.
func (b *Bot) RegisterCommands(commands []Command) {
	for _, cmd := range commands {
		b.Commands[cmd.Definition().Name] = cmd
	}
}

// Start opens the bot's session and registers handlers.
func (b *Bot) Start(registerHandlers func(*Bot)) error {
	registerHandlers(b)

	err := b.Session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %w", err)
	}

	// Register slash commands
	for _, cmd := range b.Commands {
		_, err := b.Session.ApplicationCommandCreate(b.Session.State.User.ID, "", cmd.Definition())
		if err != nil {
			log.Printf("Cannot create '%v' command: %v", cmd.Definition().Name, err)
		}
	}

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
func Run(registerHandlers func(*Bot), commands []Command) {
	bot, err := NewBot()
	if err != nil {
		log.Fatalf("Error initializing bot: %v", err)
	}

	bot.RegisterCommands(commands)

	if err := bot.Start(registerHandlers); err != nil {
		log.Fatalf("Error starting bot: %v", err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	bot.Stop()
}
