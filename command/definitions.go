package command

import "github.com/bwmarrin/discordgo"

// ScanCommand defines the structure for the /scan command.
type ScanCommand struct{}

// Definition returns the application command definition.
func (c *ScanCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "scan",
		Description: "Manually trigger a scan",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "type",
				Description: "The type of scan to perform",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Global Scan",
						Value: "global",
					},
					{
						Name:  "Guild Scan",
						Value: "guild",
					},
				},
			},
			{
				Name:        "scan_mode",
				Description: "The mode of scan to perform",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Full Scan",
						Value: "full_scan",
					},
					{
						Name:  "Active Thread Scan",
						Value: "active_thread_scan",
					},
				},
			},
			{
				Name:         "guild_id",
				Description:  "The guild to scan (required for guild scan)",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     false,
				Autocomplete: true,
			},
		},
	}
}

// PingCommand defines the structure for the /ping command.
type PingCommand struct{}

// Definition returns the application command definition.
func (c *PingCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Responds with Pong!",
	}
}

// RecentPostsCommand defines the structure for the /recent_posts command.
type RecentPostsCommand struct{}
