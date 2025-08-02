package command

import "github.com/bwmarrin/discordgo"

// StartScanCommand defines the structure for the /start_scan command.
type StartScanCommand struct{}

// Definition returns the application command definition.
func (c *StartScanCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "start_scan",
		Description: "Manually trigger a scan",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "action",
				Description: "The type of scan to perform",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "全局扫描 (Global Scan)",
						Value: "global_scan",
					},
					{
						Name:  "活跃贴扫描 (Active Thread Scan)",
						Value: "active_thread_scan",
					},
				},
			},
			{
				Name:         "guild",
				Description:  "The guild to scan",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     true,
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
