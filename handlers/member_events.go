package handlers

import (
	"discord-bot/database"
	"discord-bot/models"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// MemberAddHandler handles the GuildMemberAdd event.
func MemberAddHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	var newScamConfig models.NewScamConfig
	if err := viper.UnmarshalKey("new_scam", &newScamConfig); err != nil {
		log.Printf("Unable to decode into struct, %v", err)
		return
	}
	config, ok := newScamConfig[m.GuildID]
	if !ok {
		return
	}
	db, err := database.NewGuildStatsDB(config.Filepath, m.GuildID)
	if err != nil {
		log.Printf("Failed to open database for guild %s: %v", m.GuildID, err)
		return
	}
	defer db.Close()
	if err := db.IncrementJoins(1); err != nil {
		log.Printf("Failed to increment joins for guild %s: %v", m.GuildID, err)
		return
	}
	log.Printf("Member %s joined guild %s, joins incremented successfully", m.User.Username, m.GuildID)
}

// MemberRemoveHandler handles the GuildMemberRemove event.
func MemberRemoveHandler(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	var newScamConfig models.NewScamConfig
	if err := viper.UnmarshalKey("new_scam", &newScamConfig); err != nil {
		log.Printf("Unable to decode into struct, %v", err)
		return
	}
	config, ok := newScamConfig[m.GuildID]
	if !ok {
		return
	}
	db, err := database.NewGuildStatsDB(config.Filepath, m.GuildID)
	if err != nil {
		log.Printf("Failed to open database for guild %s: %v", m.GuildID, err)
		return
	}
	defer db.Close()
	if err := db.IncrementLeaves(1); err != nil {
		log.Printf("Failed to increment leaves for guild %s: %v", m.GuildID, err)
	}
	log.Printf("Member %s left guild %s", m.User.Username, m.GuildID)
}

// MemberUpdateHandler handles the GuildMemberUpdate event.
func MemberUpdateHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	var newScamConfig models.NewScamConfig
	if err := viper.UnmarshalKey("new_scam", &newScamConfig); err != nil {
		log.Printf("Unable to decode into struct, %v", err)
		return
	}

	config, ok := newScamConfig[m.GuildID]
	if !ok {
		return
	}

	roleAdded := false
	for _, roleID := range m.Roles {
		if roleID == config.RoleID {
			roleAdded = true
			break
		}
	}

	roleExisted := false
	for _, roleID := range m.BeforeUpdate.Roles {
		if roleID == config.RoleID {
			roleExisted = true
			break
		}
	}

	if roleAdded && !roleExisted {
		db, err := database.NewGuildStatsDB(config.Filepath, m.GuildID)
		if err != nil {
			log.Printf("Failed to open database for guild %s: %v", m.GuildID, err)
			return
		}
		defer db.Close()
		if err := db.IncrementRoleGains(1); err != nil {
			log.Printf("Failed to increment role gains for guild %s: %v", m.GuildID, err)
		}
	}
}
