package handlers

import (
	"discord-bot/models"
	"encoding/json"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
)

// HandleAutocomplete handles all autocomplete interactions.
func HandleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	switch data.Name {
	case "scan":
		for _, opt := range i.ApplicationCommandData().Options {
			if opt.Name == "guild_id" && opt.Focused {
				handleGuildAutocomplete(s, i)
			}
		}
	}
}

func handleGuildAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Read the config file
	configFile, err := os.ReadFile("config/scanning_config.json")
	if err != nil {
		log.Printf("Error reading config file for autocomplete: %v", err)
		return
	}

	var fileConfig models.ScanningFileConfig
	if err := json.Unmarshal(configFile, &fileConfig); err != nil {
		log.Printf("Error unmarshalling config file for autocomplete: %v", err)
		return
	}
	guilds := fileConfig.ScanningConfig

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(guilds))
	for _, guild := range guilds {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  guild.Name,
			Value: guild.GuildsID,
		})
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	if err != nil {
		log.Printf("Error responding to autocomplete interaction: %v", err)
	}
}
