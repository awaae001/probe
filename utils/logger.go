package utils

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

const (
	ColorInfo  = 0x00ff00 // Green
	ColorWarn  = 0xffff00 // Yellow
	ColorError = 0xff0000 // Red
)

var (
	session   *discordgo.Session
	channelID string
)

// InitLogger initializes the logger with a Discord session.
func InitLogger(s *discordgo.Session) {
	session = s
	channelID = viper.GetString("bot.adminChannelId")
	if channelID == "" {
		log.Println("Warning: bot.adminChannelId is not set in config.yaml. Logging to channel will be disabled.")
	}
}

// Log sends a log message to the admin channel.
func Log(level, module, operation, details string) {
	if session == nil || channelID == "" {
		log.Printf("[%s] Module: %s, Operation: %s, Details: %s", level, module, operation, details)
		return
	}

	var color int
	switch level {
	case "INFO":
		color = ColorInfo
	case "WARN":
		color = ColorWarn
	case "ERROR":
		color = ColorError
	default:
		color = ColorInfo
	}

	embed := &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("Log Level: %s", level),
		Color:     color,
		Timestamp: time.Now().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "模块",
				Value:  module,
				Inline: true,
			},
			{
				Name:   "操作",
				Value:  operation,
				Inline: true,
			},
			{
				Name:  "附加信息",
				Value: details,
			},
		},
	}

	_, err := session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		log.Printf("Error sending log message to Discord: %v", err)
	}
}

// Info logs an informational message.
func Info(module, operation, details string) {
	Log("INFO", module, operation, details)
}

// Warn logs a warning message.
func Warn(module, operation, details string) {
	Log("WARN", module, operation, details)
}

// Error logs an error message.
func Error(module, operation, details string) {
	Log("ERROR", module, operation, details)
}
