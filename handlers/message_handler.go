package handlers

import (
	"discord-bot/bot"
	"discord-bot/database"
	"discord-bot/handlers/message"
	"discord-bot/models"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/bwmarrin/discordgo"
)

// MessageCollector manages and dispatches events to registered message handlers.
type MessageCollector struct {
	handlers []message.MessageHandler
}

var globalMessageCollector *MessageCollector
var once sync.Once

// InitMessageCollector initializes the message collector and all configured mode handlers.
func InitMessageCollector(configPath string) error {
	var err error
	once.Do(func() {
		log.Println("Initializing MessageCollector and sub-handlers...")

		file, fileErr := os.Open(configPath)
		if fileErr != nil {
			err = fmt.Errorf("failed to open message listener config: %w", fileErr)
			return
		}
		defer file.Close()

		bytes, readErr := ioutil.ReadAll(file)
		if readErr != nil {
			err = fmt.Errorf("failed to read message listener config: %w", readErr)
			return
		}

		var fileConfig models.MessageListenerFileConfig
		if unmarshalErr := json.Unmarshal(bytes, &fileConfig); unmarshalErr != nil {
			err = fmt.Errorf("failed to unmarshal message listener config: %w", unmarshalErr)
			return
		}

		config := fileConfig.MessageListener
		collector := &MessageCollector{
			handlers: []message.MessageHandler{},
		}

		statusManager := database.NewStatusManager(config.DBStatus)

		for _, mode := range config.GloabMode {
			switch mode {
			case "base":
				for guildID, guildConfig := range config.Data.BaseModeConfig {
					handler, err := message.NewBaseHandler(guildConfig)
					if err != nil {
						log.Printf("Failed to initialize BaseHandler for guild %s: %v", guildConfig.GuildsID, err)
						continue
					}
					collector.handlers = append(collector.handlers, handler)
					statusManager.RegisterDB(guildID, "base", guildConfig.DBPath)
				}
			case "plus":
				for guildID, guildConfig := range config.Data.PlusModeConfig {
					handler, err := message.NewPlusHandler(guildConfig)
					if err != nil {
						log.Printf("Failed to initialize PlusHandler for guild %s: %v", guildConfig.GuildsID, err)
						continue
					}
					collector.handlers = append(collector.handlers, handler)
					statusManager.RegisterDB(guildID, "plus", guildConfig.DBPath)
				}
			default:
				log.Printf("Unknown message handler mode in config: %s", mode)
			}
		}

		if err := statusManager.Save(); err != nil {
			log.Printf("Failed to save db_status.json: %v", err)
		}

		globalMessageCollector = collector
		log.Printf("MessageCollector initialized with %d handlers.", len(collector.handlers))
	})
	return err
}

// MessageCreateHandler dispatches message create events to all handlers.
func MessageCreateHandler(b *bot.Bot) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if globalMessageCollector == nil {
			return
		}
		for _, handler := range globalMessageCollector.handlers {
			handler.HandleCreate(s, m)
		}
	}
}

// MessageUpdateHandler dispatches message update events to handlers that implement the method.
func MessageUpdateHandler(b *bot.Bot) func(s *discordgo.Session, m *discordgo.MessageUpdate) {
	return func(s *discordgo.Session, m *discordgo.MessageUpdate) {
		if globalMessageCollector == nil {
			return
		}
		for _, handler := range globalMessageCollector.handlers {
			// Type assert to check if the handler implements HandleUpdate
			if updater, ok := handler.(interface {
				HandleUpdate(*discordgo.Session, *discordgo.MessageUpdate)
			}); ok {
				updater.HandleUpdate(s, m)
			}
		}
	}
}

// MessageDeleteHandler dispatches message delete events to handlers that implement the method.
func MessageDeleteHandler(b *bot.Bot) func(s *discordgo.Session, m *discordgo.MessageDelete) {
	return func(s *discordgo.Session, m *discordgo.MessageDelete) {
		if globalMessageCollector == nil {
			return
		}
		for _, handler := range globalMessageCollector.handlers {
			// Type assert to check if the handler implements HandleDelete
			if deleter, ok := handler.(interface {
				HandleDelete(*discordgo.Session, *discordgo.MessageDelete)
			}); ok {
				deleter.HandleDelete(s, m)
			}
		}
	}
}

// CloseMessageCollector closes all registered handlers.
func CloseMessageCollector() error {
	if globalMessageCollector == nil {
		return nil
	}
	log.Println("Closing all message handlers...")
	for _, handler := range globalMessageCollector.handlers {
		if err := handler.Close(); err != nil {
			log.Printf("Error closing a handler: %v", err)
		}
	}
	return nil
}
