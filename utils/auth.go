package utils

import (
	"discord-bot/models"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// Auth provides methods for authorization checks.
type Auth struct {
	config models.CommandsConfig
}

// NewAuth creates a new Auth instance with the loaded configuration.
func NewAuth() (*Auth, error) {
	var commandsConfig models.CommandsConfig
	if err := viper.UnmarshalKey("commands", &commandsConfig); err != nil {
		return nil, err
	}
	return &Auth{config: commandsConfig}, nil
}

// IsDeveloper checks if a user is a developer.
func (a *Auth) IsDeveloper(userID string) bool {
	for _, devID := range a.config.Auth.Developers {
		if userID == devID {
			return true
		}
	}
	return false
}

// IsAdmin checks if a user has an admin role.
func (a *Auth) IsAdmin(member *discordgo.Member) bool {
	for _, adminRoleID := range a.config.Auth.AdminsRoles {
		for _, userRoleID := range member.Roles {
			if userRoleID == adminRoleID {
				return true
			}
		}
	}
	return false
}

// IsGuest checks if a user is a guest.
// In this configuration, "0" might signify a public or guest user.
func (a *Auth) IsGuest(userID string) bool {
	for _, guestID := range a.config.Auth.Guest {
		if guestID == "0" { // Special case for public access
			return true
		}
		if userID == guestID {
			return true
		}
	}
	return false
}

// CheckPermission checks if a user has the required permission level.
func (a *Auth) CheckPermission(s *discordgo.Session, i *discordgo.InteractionCreate, requiredLevel string) bool {
	user := i.Member.User
	member := i.Member

	switch requiredLevel {
	case "developer":
		return a.IsDeveloper(user.ID)
	case "admin":
		return a.IsDeveloper(user.ID) || a.IsAdmin(member)
	case "guest":
		return true // Guests are allowed
	default:
		return false
	}
}
