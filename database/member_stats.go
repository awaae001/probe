package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"discord-bot/models"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

// MemberStatsDB 处理成员统计数据库操作
// 管理所有服务器的成员统计数据
type MemberStatsDB struct {
	db *sql.DB
	s  *discordgo.Session
}

// NewMemberStatsDB 创建新的成员统计数据库实例
// dbPath: 数据库文件路径
// session: DiscordGo session
func NewMemberStatsDB(dbPath string, session *discordgo.Session) (*MemberStatsDB, error) {
	// 确保数据库目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	// 打开数据库连接
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 创建数据库实例
	mdb := &MemberStatsDB{db: db, s: session}

	// 初始化数据表
	if err := mdb.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化数据表失败: %w", err)
	}

	return mdb, nil
}

// Close 关闭数据库连接
func (mdb *MemberStatsDB) Close() {
	if mdb.db != nil {
		mdb.db.Close()
	}
}

// initTables 创建必要的数据表
func (mdb *MemberStatsDB) initTables() error {
	var name string
	// 检查表是否已存在
	err := mdb.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='member_stats'").Scan(&name)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("检查表是否存在失败: %w", err)
	}

	// 如果表已存在，则不执行任何操作
	if name == "member_stats" {
		return nil
	}

	// 创建成员统计表 - 按服务器和日期分组统计
	createStatsSQL := `CREATE TABLE member_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		guild_id TEXT NOT NULL,
		date TEXT NOT NULL,
		total_members INTEGER DEFAULT 0,
		joins_today INTEGER DEFAULT 0,
		leaves_today INTEGER DEFAULT 0,
		role_members_total INTEGER DEFAULT 0,
		role_gains_today INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(guild_id, date)
	);`

	if _, err := mdb.db.Exec(createStatsSQL); err != nil {
		return fmt.Errorf("创建成员统计表失败: %w", err)
	}

	log.Println("成员统计数据表初始化完成")
	return nil
}

// ensureTodayRecord 确保今天的统计记录存在，并在创建时填充初始数据
func (mdb *MemberStatsDB) ensureTodayRecord(guildID string) error {
	today := time.Now().Format("2006-01-02")

	// 检查记录是否已存在
	var exists int
	checkQuery := "SELECT 1 FROM member_stats WHERE guild_id = ? AND date = ?"
	err := mdb.db.QueryRow(checkQuery, guildID, today).Scan(&exists)

	// 如果记录已存在，或发生除“未找到行”之外的错误，则直接返回
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("检查记录是否存在时出错: %w", err)
	}
	if exists == 1 {
		return nil // 记录已存在，无需任何操作
	}

	// 获取服务器成员总数
	members, err := mdb.s.GuildMembers(guildID, "", 1000)
	if err != nil {
		log.Printf("获取服务器 %s 成员列表失败: %v", guildID, err)
		// 即使API调用失败，也继续创建记录，但总数可能为0
	}

	// 使用 INSERT OR IGNORE 确保记录存在
	query := `INSERT OR IGNORE INTO member_stats (guild_id, date, total_members, joins_today, leaves_today, role_members_total, role_gains_today)
			  VALUES (?, ?, ?, 0, 0, 0, 0)`

	result, err := mdb.db.Exec(query, guildID, today, len(members))
	if err != nil {
		return fmt.Errorf("确保今日记录失败: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("为服务器 %s 创建了 %s 的新统计记录，总人数: %d", guildID, today, len(members))
	}

	return nil
}

// IncrementJoins 增加指定服务器今日的加入人数
// guildID: 服务器ID, count: 增加的数量
func (mdb *MemberStatsDB) IncrementJoins(guildID string, count int) error {
	// 确保今日记录存在
	if err := mdb.ensureTodayRecord(guildID); err != nil {
		return fmt.Errorf("确保今日记录失败: %w", err)
	}

	today := time.Now().Format("2006-01-02")

	// 更新加入人数并刷新更新时间
	query := `UPDATE member_stats SET 
				joins_today = joins_today + ?, 
				updated_at = CURRENT_TIMESTAMP 
			  WHERE guild_id = ? AND date = ?`

	result, err := mdb.db.Exec(query, count, guildID, today)
	if err != nil {
		return fmt.Errorf("更新加入人数失败: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("服务器 %s 今日加入人数增加 %d，影响记录数: %d", guildID, count, rowsAffected)

	return nil
}

// IncrementLeaves 增加指定服务器今日的离开人数
// guildID: 服务器ID, count: 增加的数量
func (mdb *MemberStatsDB) IncrementLeaves(guildID string, count int) error {
	// 确保今日记录存在
	if err := mdb.ensureTodayRecord(guildID); err != nil {
		return fmt.Errorf("确保今日记录失败: %w", err)
	}

	today := time.Now().Format("2006-01-02")

	// 更新离开人数并刷新更新时间
	query := `UPDATE member_stats SET 
				leaves_today = leaves_today + ?, 
				updated_at = CURRENT_TIMESTAMP 
			  WHERE guild_id = ? AND date = ?`

	result, err := mdb.db.Exec(query, count, guildID, today)
	if err != nil {
		return fmt.Errorf("更新离开人数失败: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("服务器 %s 今日离开人数增加 %d，影响记录数: %d", guildID, count, rowsAffected)

	return nil
}

// IncrementRoleGains 增加指定服务器今日获得指定身份组的人数
// guildID: 服务器ID, count: 增加的数量
func (mdb *MemberStatsDB) IncrementRoleGains(guildID string, count int) error {
	// 确保今日记录存在
	if err := mdb.ensureTodayRecord(guildID); err != nil {
		return fmt.Errorf("确保今日记录失败: %w", err)
	}

	today := time.Now().Format("2006-01-02")

	// 更新身份组获得人数并刷新更新时间
	query := `UPDATE member_stats SET 
				role_gains_today = role_gains_today + ?, 
				updated_at = CURRENT_TIMESTAMP 
			  WHERE guild_id = ? AND date = ?`

	result, err := mdb.db.Exec(query, count, guildID, today)
	if err != nil {
		return fmt.Errorf("更新身份组获得人数失败: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("服务器 %s 今日身份组获得人数增加 %d，影响记录数: %d", guildID, count, rowsAffected)

	return nil
}

// UpdateTotals 更新指定服务器今日的总人数和身份组人数
// guildID: 服务器ID, totalMembers: 总人数, roleMembers: 拥有指定身份组的人数
func (mdb *MemberStatsDB) UpdateTotals(guildID string, totalMembers, roleMembers int) error {
	// 确保今日记录存在
	if err := mdb.ensureTodayRecord(guildID); err != nil {
		return fmt.Errorf("确保今日记录失败: %w", err)
	}

	today := time.Now().Format("2006-01-02")

	// 更新总人数和身份组人数
	query := `UPDATE member_stats SET 
				total_members = ?, 
				role_members_total = ?, 
				updated_at = CURRENT_TIMESTAMP 
			  WHERE guild_id = ? AND date = ?`

	result, err := mdb.db.Exec(query, totalMembers, roleMembers, guildID, today)
	if err != nil {
		return fmt.Errorf("更新总计数据失败: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("服务器 %s 更新统计数据: 总人数=%d, 身份组人数=%d，影响记录数: %d",
		guildID, totalMembers, roleMembers, rowsAffected)

	return nil
}

// ScheduledUpdate 执行定时统计更新，更新所有配置的服务器数据
// 定期统计总人数和拥有指定身份组的人数
func ScheduledUpdate(s *discordgo.Session) {
	// 读取配置
	var newScanConfig models.NewScanConfig
	if err := viper.UnmarshalKey("new_scan", &newScanConfig); err != nil {
		log.Printf("无法解析成员统计配置: %v", err)
		return
	}

	// 打开数据库连接
	db, err := NewMemberStatsDB(newScanConfig.DBFilePath, s)
	if err != nil {
		log.Printf("打开成员统计数据库失败: %v", err)
		return
	}
	defer db.Close()

	log.Println("开始执行定时成员统计更新...")

	// 遍历所有配置的服务器
	for guildID, guildData := range newScanConfig.Data {
		log.Printf("正在统计服务器 %s 的成员数据...", guildID)

		// 获取服务器成员列表
		members, err := s.GuildMembers(guildID, "", 1000)
		if err != nil {
			log.Printf("获取服务器 %s 成员列表失败: %v", guildID, err)
			continue
		}

		// 统计总人数
		totalMembers := len(members)

		// 统计拥有指定身份组的人数
		roleMembers := 0
		for _, member := range members {
			for _, roleID := range member.Roles {
				if roleID == guildData.RoleID {
					roleMembers++
					break
				}
			}
		}

		// 更新数据库中的统计数据
		if err := db.UpdateTotals(guildID, totalMembers, roleMembers); err != nil {
			log.Printf("更新服务器 %s 统计数据失败: %v", guildID, err)
		} else {
			log.Printf("服务器 %s 统计完成: 总人数=%d, 身份组人数=%d", guildID, totalMembers, roleMembers)
		}
	}

	log.Println("定时成员统计更新完成")
}
