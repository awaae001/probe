package database

import (
	"database/sql"
	"discord-bot/models"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// MessageDB handles message database operations
type MessageDB struct {
	connections map[string]*sql.DB // guildID -> db connection
	mutex       sync.RWMutex
	config      models.MessageConfig
}

// NewMessageDB creates a new message database manager
func NewMessageDB(config models.MessageConfig) *MessageDB {
	return &MessageDB{
		connections: make(map[string]*sql.DB),
		config:      config,
	}
}

// CreateMessagesTable creates the messages table if it doesn't exist
func CreateMessagesTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS messages (
        message_id INTEGER PRIMARY KEY,
        user_id INTEGER NOT NULL,
        guild_id INTEGER NOT NULL,
        channel_id INTEGER NOT NULL,
        timestamp INTEGER NOT NULL,
        message_content TEXT NOT NULL,
        attachments TEXT DEFAULT ''
    );`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	// Create indexes for better query performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_user_timestamp ON messages(user_id, timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_channel_timestamp ON messages(channel_id, timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_guild_timestamp ON messages(guild_id, timestamp);",
	}

	for _, indexQuery := range indexes {
		if _, err := db.Exec(indexQuery); err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}

	log.Println("Messages table and indexes created successfully")
	return nil
}

// GetDBPath generates database path based on time type and current time
func (mdb *MessageDB) GetDBPath(guildID string) (string, error) {
	guildConfig, exists := mdb.config.MessageListener.Data[guildID]
	if !exists {
		return "", fmt.Errorf("guild %s not found in configuration", guildID)
	}

	basePath := guildConfig.DBPath
	timeType := guildConfig.TimeType

	now := time.Now()
	var timeSuffix string

	switch timeType {
	case "week":
		year, week := now.ISOWeek()
		timeSuffix = fmt.Sprintf("%d-week-%d", year, week)
	case "month":
		timeSuffix = fmt.Sprintf("%d-month-%02d", now.Year(), now.Month())
	case "day":
		timeSuffix = fmt.Sprintf("%d-%02d-%02d", now.Year(), now.Month(), now.Day())
	default:
		timeSuffix = "default"
	}

	return strings.Replace(basePath, "$time_type", timeSuffix, 1), nil
}

// GetDB gets or creates a database connection for a guild
func (mdb *MessageDB) GetDB(guildID string) (*sql.DB, error) {
	mdb.mutex.RLock()
	if db, exists := mdb.connections[guildID]; exists {
		mdb.mutex.RUnlock()
		return db, nil
	}
	mdb.mutex.RUnlock()

	mdb.mutex.Lock()
	defer mdb.mutex.Unlock()

	// Double check after acquiring write lock
	if db, exists := mdb.connections[guildID]; exists {
		return db, nil
	}

	dbPath, err := mdb.GetDBPath(guildID)
	if err != nil {
		return nil, err
	}

	db, err := InitDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize message database for guild %s: %w", guildID, err)
	}

	if err := CreateMessagesTable(db); err != nil {
		db.Close()
		return nil, err
	}

	mdb.connections[guildID] = db
	log.Printf("Message database initialized for guild %s at %s", guildID, dbPath)
	return db, nil
}

// InsertMessage inserts a message record into the database
func (mdb *MessageDB) InsertMessage(msg models.Message) error {
	guildID := fmt.Sprintf("%d", msg.GuildID)
	db, err := mdb.GetDB(guildID)
	if err != nil {
		return err
	}

	query := `INSERT OR IGNORE INTO messages (message_id, user_id, guild_id, channel_id, timestamp, message_content, attachments) 
              VALUES (?, ?, ?, ?, ?, ?, ?)`

	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(msg.MessageID, msg.UserID, msg.GuildID, msg.ChannelID, msg.Timestamp, msg.MessageContent, msg.Attachments)
	if err != nil {
		return fmt.Errorf("failed to insert message %d: %w", msg.MessageID, err)
	}

	return nil
}

// GetChannelStats retrieves channel message statistics
func (mdb *MessageDB) GetChannelStats(guildID int64, from, to *time.Time) ([]models.ChannelStat, error) {
	guildIDStr := fmt.Sprintf("%d", guildID)
	db, err := mdb.GetDB(guildIDStr)
	if err != nil {
		return nil, err
	}

	query := `SELECT channel_id, COUNT(*) as message_count 
              FROM messages 
              WHERE guild_id = ?`
	args := []interface{}{guildID}

	if from != nil {
		query += " AND timestamp >= ?"
		args = append(args, from.Unix())
	}
	if to != nil {
		query += " AND timestamp < ?"
		args = append(args, to.Unix())
	}

	query += " GROUP BY channel_id ORDER BY message_count DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query channel stats: %w", err)
	}
	defer rows.Close()

	var stats []models.ChannelStat
	for rows.Next() {
		var stat models.ChannelStat
		if err := rows.Scan(&stat.ChannelID, &stat.MessageCount); err != nil {
			return nil, fmt.Errorf("failed to scan channel stat: %w", err)
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// GetUserStats retrieves user message statistics
func (mdb *MessageDB) GetUserStats(guildID int64, channelIDs []int64, from, to *time.Time) ([]models.UserStat, error) {
	guildIDStr := fmt.Sprintf("%d", guildID)
	db, err := mdb.GetDB(guildIDStr)
	if err != nil {
		return nil, err
	}

	query := `SELECT user_id, COUNT(*) as message_count 
              FROM messages 
              WHERE guild_id = ?`
	args := []interface{}{guildID}

	if len(channelIDs) > 0 {
		placeholders := strings.Repeat("?,", len(channelIDs))
		placeholders = placeholders[:len(placeholders)-1] // remove last comma
		query += " AND channel_id IN (" + placeholders + ")"
		for _, channelID := range channelIDs {
			args = append(args, channelID)
		}
	}

	if from != nil {
		query += " AND timestamp >= ?"
		args = append(args, from.Unix())
	}
	if to != nil {
		query += " AND timestamp < ?"
		args = append(args, to.Unix())
	}

	query += " GROUP BY user_id ORDER BY message_count DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user stats: %w", err)
	}
	defer rows.Close()

	var stats []models.UserStat
	for rows.Next() {
		var stat models.UserStat
		if err := rows.Scan(&stat.UserID, &stat.MessageCount); err != nil {
			return nil, fmt.Errorf("failed to scan user stat: %w", err)
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// Close closes all database connections
func (mdb *MessageDB) Close() error {
	mdb.mutex.Lock()
	defer mdb.mutex.Unlock()

	var lastErr error
	for guildID, db := range mdb.connections {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database for guild %s: %v", guildID, err)
			lastErr = err
		}
	}
	mdb.connections = make(map[string]*sql.DB)
	return lastErr
}

// GetMessageCount returns the total message count for a guild
func (mdb *MessageDB) GetMessageCount(guildID int64) (int64, error) {
	guildIDStr := fmt.Sprintf("%d", guildID)
	db, err := mdb.GetDB(guildIDStr)
	if err != nil {
		return 0, err
	}

	var count int64
	err = db.QueryRow("SELECT COUNT(*) FROM messages WHERE guild_id = ?", guildID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get message count for guild %d: %w", guildID, err)
	}

	return count, nil
}