package handlers

import (
	"discord-bot/database"
	"discord-bot/models"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// MemberAddHandler 处理成员加入服务器事件
// 当有新成员加入服务器时，统计并记录加入人数
func MemberAddHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	log.Printf("检测到成员 %s (%s) 加入了服务器 %s", m.User.Username, m.User.ID, m.GuildID)

	// 读取成员统计配置
	var newScanConfig models.NewScanConfig
	newScanConfig.DBFilePath = viper.GetString("db_file_path")
	if err := viper.UnmarshalKey("data", &newScanConfig.Data); err != nil {
		log.Printf("无法解析成员统计配置: %v", err)
		return
	}

	// 检查该服务器是否在配置中
	_, exists := newScanConfig.Data[m.GuildID]
	if !exists {
		log.Printf("服务器 %s 不在成员统计配置中，跳过处理", m.GuildID)
		return
	}

	// 打开数据库连接
	db, err := database.NewMemberStatsDB(newScanConfig.DBFilePath, s)
	if err != nil {
		log.Printf("打开成员统计数据库失败: %v", err)
		return
	}
	defer db.Close()

	// 增加今日加入人数
	if err := db.IncrementJoins(m.GuildID, 1); err != nil {
		log.Printf("更新服务器 %s 加入人数失败: %v", m.GuildID, err)
		return
	}

	log.Printf("成功记录成员 %s 加入服务器 %s 的统计", m.User.Username, m.GuildID)
}

// MemberRemoveHandler 处理成员离开服务器事件
// 当成员离开服务器时，统计并记录离开人数
func MemberRemoveHandler(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	log.Printf("检测到成员 %s (%s) 离开了服务器 %s", m.User.Username, m.User.ID, m.GuildID)

	// 读取成员统计配置
	var newScanConfig models.NewScanConfig
	newScanConfig.DBFilePath = viper.GetString("db_file_path")
	if err := viper.UnmarshalKey("data", &newScanConfig.Data); err != nil {
		log.Printf("无法解析成员统计配置: %v", err)
		return
	}

	// 检查该服务器是否在配置中
	_, exists := newScanConfig.Data[m.GuildID]
	if !exists {
		log.Printf("服务器 %s 不在成员统计配置中，跳过处理", m.GuildID)
		return
	}

	// 打开数据库连接
	db, err := database.NewMemberStatsDB(newScanConfig.DBFilePath, s)
	if err != nil {
		log.Printf("打开成员统计数据库失败: %v", err)
		return
	}
	defer db.Close()

	// 增加今日离开人数
	if err := db.IncrementLeaves(m.GuildID, 1); err != nil {
		log.Printf("更新服务器 %s 离开人数失败: %v", m.GuildID, err)
		return
	}

	log.Printf("成功记录成员 %s 离开服务器 %s 的统计", m.User.Username, m.GuildID)
}

// MemberUpdateHandler 处理成员信息更新事件
// 主要监控身份组的变更，统计获得指定身份组的人数
func MemberUpdateHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	// 读取成员统计配置
	var newScanConfig models.NewScanConfig
	newScanConfig.DBFilePath = viper.GetString("db_file_path")
	if err := viper.UnmarshalKey("data", &newScanConfig.Data); err != nil {
		log.Printf("无法解析成员统计配置: %v", err)
		return
	}

	// 检查该服务器是否在配置中
	guildData, exists := newScanConfig.Data[m.GuildID]
	if !exists {
		return
	}

	// 检查用户是否获得了指定的身份组
	hasTargetRoleNow := false
	for _, roleID := range m.Roles {
		if roleID == guildData.RoleID {
			hasTargetRoleNow = true
			break
		}
	}

	// 检查用户之前是否已经有这个身份组
	hadTargetRoleBefore := false
	if m.BeforeUpdate != nil {
		for _, roleID := range m.BeforeUpdate.Roles {
			if roleID == guildData.RoleID {
				hadTargetRoleBefore = true
				break
			}
		}
	}

	// 如果现在有身份组但之前没有，说明是新获得的
	if hasTargetRoleNow && !hadTargetRoleBefore {
		log.Printf("检测到成员 %s (%s) 在服务器 %s 中获得了目标身份组 %s",
			m.User.Username, m.User.ID, m.GuildID, guildData.RoleID)

		// 打开数据库连接
		db, err := database.NewMemberStatsDB(newScanConfig.DBFilePath, s)
		if err != nil {
			log.Printf("打开成员统计数据库失败: %v", err)
			return
		}
		defer db.Close()

		// 增加今日身份组获得人数
		if err := db.IncrementRoleGains(m.GuildID, 1); err != nil {
			log.Printf("更新服务器 %s 身份组获得人数失败: %v", m.GuildID, err)
			return
		}

		log.Printf("成功记录成员 %s 在服务器 %s 获得目标身份组的统计", m.User.Username, m.GuildID)
	} else if !hasTargetRoleNow && hadTargetRoleBefore {
		// 如果之前有身份组但现在没有，记录日志但不统计（因为我们只关心获得的数量）
		log.Printf("成员 %s (%s) 在服务器 %s 中失去了目标身份组 %s",
			m.User.Username, m.User.ID, m.GuildID, guildData.RoleID)
	}
}
