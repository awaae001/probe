package handlers

import (
	"discord-bot/models"
	"discord-bot/scanner"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// HandleStartScan handles the logic for the /start_scan command.
func HandleStartScan(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	var action, guildID string

	if opt, ok := optionMap["action"]; ok {
		action = opt.StringValue()
	}
	if opt, ok := optionMap["guild"]; ok {
		guildID = opt.StringValue()
	}

	if guildID == "" || action == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error: Missing required options.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Respond to the interaction immediately to avoid timeout.
	initialResponse := fmt.Sprintf("Received command to start **%s** for guild **%s**. Preparing to scan...", action, guildID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: initialResponse,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	// Run the actual scanning in a separate goroutine.
	go func() {
		var fullConfig models.ScanningConfig
		if err := viper.Unmarshal(&fullConfig); err != nil {
			log.Printf("Error unmarshalling full config for manual scan: %v", err)
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "Error: Could not load scanning configuration.",
			})
			return
		}

		guildConfig, ok := fullConfig[guildID]
		if !ok {
			log.Printf("Error: Guild ID %s not found in scanning configuration.", guildID)
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: fmt.Sprintf("Error: Guild ID %s not found in your configuration.", guildID),
			})
			return
		}

		// Create a temporary config with only the selected guild.
		singleGuildConfig := make(models.ScanningConfig)
		singleGuildConfig[guildID] = guildConfig

		isFullScan := (action == "global_scan")

		log.Printf("Starting manual scan (isFullScan: %v) for guild: %s", isFullScan, guildConfig.Name)
		scanner.StartScanning(s, singleGuildConfig, isFullScan)
		log.Printf("Manual scan finished for guild: %s", guildConfig.Name)

		// Send a followup message to notify the user.
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("✅ Scan **%s** for guild **%s** has completed.", action, guildConfig.Name),
		})
	}()
}

// HandlePing handles the logic for the /ping command.
func HandlePing(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Pong!",
		},
	})
}
