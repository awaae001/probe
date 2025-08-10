package handlers

import (
	"context"
	"discord-bot/grpc"
	"discord-bot/models"
	"discord-bot/proto"
	"discord-bot/scanner"
	"fmt"
	"log"
	"time"

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
		var fullConfig models.ScanningConfig
		if err := viper.Unmarshal(&fullConfig); err != nil {
			log.Printf("Error unmarshalling config for manual scan: %v", err)
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "Error: Could not load scanning configuration.",
			})
			return
		}

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

// HandleRecentPosts handles the logic for the /recent_posts command.
func HandleRecentPosts(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	var timeRange, channelID, authorID string
	var pageSize, pageNumber int64

	if opt, ok := optionMap["time_range"]; ok {
		timeRange = opt.StringValue()
	}
	if opt, ok := optionMap["channel_id"]; ok {
		channelID = opt.StringValue()
	}
	if opt, ok := optionMap["author_id"]; ok {
		authorID = opt.StringValue()
	}
	if opt, ok := optionMap["page_size"]; ok {
		pageSize = opt.IntValue()
	}
	if opt, ok := optionMap["page_number"]; ok {
		pageNumber = opt.IntValue()
	}

	// 设置默认值
	if pageSize == 0 {
		pageSize = 10
	}
	if pageNumber == 0 {
		pageNumber = 1
	}

	// 获取 gRPC 服务器地址
	serverAddress := viper.GetString("grpc.server_address")
	if serverAddress == "" {
		serverAddress = "0.0.0.0:50051" // 默认地址
	}

	// 立即响应，表示正在处理
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// 在 goroutine 中处理 gRPC 调用
	go func() {
		// 创建 gRPC 客户端
		client, err := grpc.NewPostClient(serverAddress, 30*time.Second)
		if err != nil {
			sendErrorFollowup(s, i, fmt.Sprintf("无法连接到远程服务器: %v", err))
			return
		}
		defer client.Close()

		// 构建查询选项
		var opts []grpc.QueryOption

		if channelID != "" {
			opts = append(opts, grpc.WithChannelId(channelID))
		}
		if authorID != "" {
			opts = append(opts, grpc.WithAuthorId(authorID))
		}
		opts = append(opts, grpc.WithPagination(int32(pageSize), int32(pageNumber)))

		// 根据时间范围调用相应的查询方法
		var resp *proto.QueryPostsResponse
		switch timeRange {
		case "yesterday":
			resp, err = client.QueryYesterdayPosts(context.Background(), opts...)
		case "1day":
			resp, err = client.QueryLastDayPosts(context.Background(), opts...)
		case "3days":
			resp, err = client.QueryLast3DaysPosts(context.Background(), opts...)
		case "7days":
			resp, err = client.QueryLast7DaysPosts(context.Background(), opts...)
		default:
			err = fmt.Errorf("无效的时间范围: %s", timeRange)
		}

		if err != nil {
			sendErrorFollowup(s, i, fmt.Sprintf("查询帖子时出错: %v", err))
			return
		}

		// 格式化响应
		content := formatPostsResponse(resp, timeRange)

		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: content,
		})
	}()
}

// sendErrorFollowup 发送错误消息
func sendErrorFollowup(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("❌ 错误: %s", message),
	})
}

// formatPostsResponse 格式化帖子查询响应
func formatPostsResponse(resp *proto.QueryPostsResponse, timeRange string) string {
	posts := resp.GetPosts()
	totalCount := resp.GetTotalCount()

	if len(posts) == 0 {
		return "📭 没有找到符合条件的帖子。"
	}

	var content string
	switch timeRange {
	case "yesterday":
		content += fmt.Sprintf("📅 **昨天的帖子** (共 %d 条):\n\n", totalCount)
	case "1day":
		content += fmt.Sprintf("📅 **最近1天的帖子** (共 %d 条):\n\n", totalCount)
	case "3days":
		content += fmt.Sprintf("📅 **最近3天的帖子** (共 %d 条):\n\n", totalCount)
	case "7days":
		content += fmt.Sprintf("📅 **最近7天的帖子** (共 %d 条):\n\n", totalCount)
	}

	for i, post := range posts {
		content += fmt.Sprintf("**%d. %s**\n", i+1, post.GetTitle())
		content += fmt.Sprintf("   👤 作者: %s\n", post.GetAuthorId())
		content += fmt.Sprintf("   📺 频道: %s\n", post.GetChannelId())
		content += fmt.Sprintf("   ⏰ 时间: %s\n", grpc.FormatTimestamp(post.GetCreatedAt()))
		content += fmt.Sprintf("   👍 反应: %d | 💬 回复: %d\n", post.GetReactionCount(), post.GetReplyCount())

		// 如果内容太长，截断显示
		postContent := post.GetContent()
		if len(postContent) > 100 {
			postContent = postContent[:100] + "..."
		}
		content += fmt.Sprintf("   📝 内容: %s\n\n", postContent)
	}

	return content
}
