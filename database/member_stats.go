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

// GuildStatsDB handles database operations for a single guild.
type GuildStatsDB struct {
	db        *sql.DB
	tableName string
}

// NewGuildStatsDB creates a new GuildStatsDB instance.
func NewGuildStatsDB(filepath_str, guildID string) (*GuildStatsDB, error) {
	dir := filepath.Dir(filepath_str)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory for database: %w", err)
	}

	db, err := sql.Open("sqlite3", filepath_str)
	if err != nil {
		return nil, err
	}
	g := &GuildStatsDB{
		db:        db,
		tableName: "new_status",
	}
	if err := g.InitTable(); err != nil {
		db.Close()
		return nil, err
	}
	return g, nil
}

// Close closes the database connection.
func (g *GuildStatsDB) Close() {
	g.db.Close()
}

// InitTable creates the necessary table if it doesn't exist.
func (g *GuildStatsDB) InitTable() error {
	createTableSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		"date" TEXT NOT NULL PRIMARY KEY,
		"total_members" INTEGER DEFAULT 0,
		"joins" INTEGER DEFAULT 0,
		"leaves" INTEGER DEFAULT 0,
		"role_members" INTEGER DEFAULT 0,
		"role_gains" INTEGER DEFAULT 0
	);`, g.tableName)
	_, err := g.db.Exec(createTableSQL)
	return err
}

func (g *GuildStatsDB) ensureTodayRecord() error {
	today := time.Now().Format("2006-01-02")
	query := fmt.Sprintf("INSERT OR IGNORE INTO %s (date) VALUES (?)", g.tableName)
	result, err := g.db.Exec(query, today)
	if err != nil {
		log.Printf("ensureTodayRecord: Failed to insert record for %s: %v", today, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	log.Printf("ensureTodayRecord: Inserted/verified record for %s, rows affected: %d", today, rowsAffected)
	return nil
}

// IncrementJoins increments the join count for today.
func (g *GuildStatsDB) IncrementJoins(count int) error {
	log.Printf("IncrementJoins: Starting for count %d", count)
	if err := g.ensureTodayRecord(); err != nil {
		log.Printf("IncrementJoins: Failed to ensure today record: %v", err)
		return err
	}
	today := time.Now().Format("2006-01-02")
	query := fmt.Sprintf("UPDATE %s SET joins = joins + ? WHERE date = ?", g.tableName)
	result, err := g.db.Exec(query, count, today)
	if err != nil {
		log.Printf("IncrementJoins: Failed to execute update query: %v", err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	log.Printf("IncrementJoins: Successfully updated %d rows for date %s", rowsAffected, today)
	return nil
}

// IncrementLeaves increments the leave count for today.
func (g *GuildStatsDB) IncrementLeaves(count int) error {
	if err := g.ensureTodayRecord(); err != nil {
		return err
	}
	today := time.Now().Format("2006-01-02")
	_, err := g.db.Exec(fmt.Sprintf("UPDATE %s SET leaves = leaves + ? WHERE date = ?", g.tableName), count, today)
	return err
}

// IncrementRoleGains increments the role gain count for today.
func (g *GuildStatsDB) IncrementRoleGains(count int) error {
	if err := g.ensureTodayRecord(); err != nil {
		return err
	}
	today := time.Now().Format("2006-01-02")
	_, err := g.db.Exec(fmt.Sprintf("UPDATE %s SET role_gains = role_gains + ? WHERE date = ?", g.tableName), count, today)
	return err
}

// UpdateTotals updates the total and role member counts for today.
func (g *GuildStatsDB) UpdateTotals(totalMembers, roleMembers int) error {
	if err := g.ensureTodayRecord(); err != nil {
		return err
	}
	today := time.Now().Format("2006-01-02")
	_, err := g.db.Exec(fmt.Sprintf("UPDATE %s SET total_members = ?, role_members = ? WHERE date = ?", g.tableName), totalMembers, roleMembers, today)
	return err
}

// ScheduledUpdate performs the scheduled update for all guilds.
func ScheduledUpdate(s *discordgo.Session) {
	var newScamConfig models.NewScamConfig
	if err := viper.UnmarshalKey("new_scam", &newScamConfig); err != nil {
		log.Printf("Unable to decode into struct, %v", err)
		return
	}

	for guildID, config := range newScamConfig {
		db, err := NewGuildStatsDB(config.Filepath, guildID)
		if err != nil {
			log.Printf("Failed to open database for guild %s: %v", guildID, err)
			continue
		}
		members, err := s.GuildMembers(guildID, "", 1000)
		if err != nil {
			log.Printf("Failed to get members for guild %s: %v", guildID, err)
			db.Close()
			continue
		}
		totalMembers := len(members)

		roleMembers := 0
		for _, member := range members {
			for _, roleID := range member.Roles {
				if roleID == config.RoleID {
					roleMembers++
					break
				}
			}
		}

		if err := db.UpdateTotals(totalMembers, roleMembers); err != nil {
			log.Printf("Failed to update totals for guild %s: %v", guildID, err)
		}
		db.Close()
	}
}
