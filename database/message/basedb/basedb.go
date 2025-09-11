package basedb

import (
	"database/sql"
	"discord-bot/models"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// BaseDB handles database operations for the base message listener mode.
type BaseDB struct {
	db *sql.DB
}

// NewBaseDB creates and initializes a new database connection for the base mode.
// It ensures the database file and the necessary tables are created if they don't exist.
func NewBaseDB(config models.BaseGuildConfig) (*BaseDB, error) {
	// Ensure the directory for the database file exists.
	dir := filepath.Dir(config.DBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
	}

	// Open the SQLite database file. It will be created if it doesn't exist.
	db, err := sql.Open("sqlite3", config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database at %s: %w", config.DBPath, err)
	}

	// Create the messages table if it doesn't exist.
	if err := createMessagesTable(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create messages table: %w", err)
	}

	log.Printf("Successfully initialized base database at %s", config.DBPath)

	return &BaseDB{db: db}, nil
}

// createMessagesTable creates the 'messages' table for the base mode.
func createMessagesTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS messages (
        author_id TEXT NOT NULL,
        timestamp INTEGER NOT NULL,
        message_id TEXT PRIMARY KEY,
        channel_id TEXT NOT NULL,
        guild_id TEXT NOT NULL
    );`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to execute table creation query: %w", err)
	}

	// Optional: Create indexes for better performance on common queries.
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_author_timestamp ON messages(author_id, timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_channel_timestamp ON messages(channel_id, timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_guild_timestamp ON messages(guild_id, timestamp);",
	}

	for _, indexQuery := range indexes {
		if _, err := db.Exec(indexQuery); err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}

	return nil
}

// Close closes the database connection.
func (b *BaseDB) Close() error {
	if b.db != nil {
		return b.db.Close()
	}
	return nil
}

// SaveMessage saves a single message to the database.
// This is a simplified version for the base mode, storing only essential fields.
func (b *BaseDB) SaveMessage(msg models.Message) error {
	query := `
    INSERT OR IGNORE INTO messages (author_id, timestamp, message_id, channel_id, guild_id)
    VALUES (?, ?, ?, ?, ?);`

	stmt, err := b.db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement for base db: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(msg.UserID, msg.Timestamp, msg.MessageID, msg.ChannelID, msg.GuildID)
	if err != nil {
		return fmt.Errorf("failed to insert message into base database: %w", err)
	}

	return nil
}
