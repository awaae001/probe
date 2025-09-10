package handlers

import (
	"discord-bot/models"
	"discord-bot/scanner"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// HandleScan handles the logic for the /scan command.
func HandleScan(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	var scanType, guildID, scanMode string

	if opt, ok := optionMap["type"]; ok {
		scanType = opt.StringValue()
	}
	if opt, ok := optionMap["guild_id"]; ok {
		guildID = opt.StringValue()
	}
	if opt, ok := optionMap["scan_mode"]; ok {
		scanMode = opt.StringValue()
	}

	if scanType == "guild" && guildID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error: Guild ID is required for a guild-specific scan.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Respond to the interaction immediately.
	var initialResponse string
	if scanType == "global" {
		initialResponse = fmt.Sprintf("Received command to start a **%s** global scan. Preparing to scan...", scanMode)
	} else {
		initialResponse = fmt.Sprintf("Received command to start a **%s** for guild **%s**. Preparing to scan...", scanMode, guildID)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: initialResponse,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	// Run the scanning in a goroutine.
	go func() {
		var fileConfig models.ScanningFileConfig
		if err := viper.Unmarshal(&fileConfig); err != nil {
			log.Printf("Error unmarshalling config for manual scan: %v", err)
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "Error: Could not load scanning configuration.",
			})
			return
		}

		fullConfig := fileConfig.ScanningConfig
		configToScan := make(models.ScanningConfig)
		switch scanType {
		case "guild":
			guildConfig, ok := fullConfig[guildID]
			if !ok {
				log.Printf("Error: Guild ID %s not found in config.", guildID)
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: fmt.Sprintf("Error: Guild ID %s not found in your configuration.", guildID),
				})
				return
			}
			configToScan[guildID] = guildConfig
		case "global":
			if scanMode == "active_thread_scan" && guildID != "" {
				guildConfig, ok := fullConfig[guildID]
				if !ok {
					log.Printf("Error: Guild ID %s not found in config.", guildID)
					s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
						Content: fmt.Sprintf("Error: Guild ID %s not found in your configuration.", guildID),
					})
					return
				}
				configToScan[guildID] = guildConfig
			} else {
				configToScan = fullConfig
			}
		}

		isFullScan := (scanMode == "full_scan")
		log.Printf("Starting manual scan (isFullScan: %v, type: %s)", isFullScan, scanType)
		scanner.StartScanning(s, configToScan, isFullScan)
		log.Printf("Manual scan finished (type: %s)", scanType)

		// Send a followup message.
		var followupContent string
		if scanType == "global" {
			followupContent = fmt.Sprintf("✅ Global scan (%s) has completed.", scanMode)
		} else {
			guildName := configToScan[guildID].Name
			followupContent = fmt.Sprintf("✅ Scan (%s) for guild **%s** has completed.", scanMode, guildName)
		}
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: followupContent,
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
