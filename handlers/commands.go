package handlers

import (
	"discord-bot/utils"
	"log"

	"github.com/bwmarrin/discordgo"
)

// CommandDispatcher is the central handler for all application command interactions.
// It performs permission checks and then dispatches the interaction to the appropriate handler.
func CommandDispatcher(s *discordgo.Session, i *discordgo.InteractionCreate) {
	auth, err := utils.NewAuth()
	if err != nil {
		log.Printf("Failed to create auth instance: %v", err)
		// Handle error appropriately
		return
	}

	commandPermissions := map[string]string{
		"scan":         "admin",
		"ping":         "guest",
		"recent_posts": "guest",
	}

	commandName := i.ApplicationCommandData().Name
	requiredLevel, ok := commandPermissions[commandName]

	if ok {
		if !auth.CheckPermission(s, i, requiredLevel) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "🚫 你没有权限执行此命令",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

	switch commandName {
	case "scan":
		HandleScan(s, i)
	case "ping":
		HandlePing(s, i)
	default:
		// Optionally, send an error message for unknown commands.
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "🚫内部错误：Unknown command.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
}
