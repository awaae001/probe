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

// Definition returns the application command definition.
func (c *RecentPostsCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "recent_posts",
		Description: "查询最近的新帖子",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "time_range",
				Description: "时间范围",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "昨天",
						Value: "yesterday",
					},
					{
						Name:  "最近1天",
						Value: "1day",
					},
					{
						Name:  "最近3天",
						Value: "3days",
					},
					{
						Name:  "最近7天",
						Value: "7days",
					},
				},
			},
			{
				Name:        "channel_id",
				Description: "筛选特定频道的帖子（可选）",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    false,
			},
			{
				Name:        "author_id",
				Description: "筛选特定作者的帖子（可选）",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    false,
			},
			{
				Name:        "page_size",
				Description: "每页显示的帖子数量（默认10）",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    false,
			},
			{
				Name:        "page_number",
				Description: "页码（默认1）",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    false,
			},
		},
	}
}
