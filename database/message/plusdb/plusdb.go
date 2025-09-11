package plusdb

import (
	"database/sql"
	"discord-bot/models"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// PlusDB handles message database operations for the plus mode.
type PlusDB struct {
	db     *sql.DB
	config models.PlusGuildConfig
	mutex  sync.RWMutex
}

// NewPlusDB creates a new message database manager for the plus mode.
func NewPlusDB(config models.PlusGuildConfig) (*PlusDB, error) {
	dbPath, err := getDBPath(config, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to generate initial DB path: %w", err)
	}

	// Ensure the directory for the database file exists.
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize message database for guild %s: %w", config.GuildsID, err)
	}

	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	log.Printf("Plus database initialized for guild %s at %s", config.GuildsID, dbPath)

	return &PlusDB{
		db:     db,
		config: config,
	}, nil
}

// createTables creates all necessary tables for the plus mode.
func createTables(db *sql.DB) error {
	if err := createMessagesTable(db); err != nil {
		return err
	}
	if err := createMessageDeletionsTable(db); err != nil {
		return err
	}
	if err := createMessageEditsTable(db); err != nil {
		return err
	}
	return nil
}

// getDBPath generates a database path based on the configuration and a given time.
func getDBPath(config models.PlusGuildConfig, t time.Time) (string, error) {
	basePath := config.DBPath
	timeType := config.TimeType
	var timeSuffix string

	switch timeType {
	case "week":
		year, week := t.ISOWeek()
		timeSuffix = fmt.Sprintf("%d-week-%d", year, week)
	case "month":
		timeSuffix = fmt.Sprintf("%d-month-%02d", t.Year(), t.Month())
	case "day":
		timeSuffix = fmt.Sprintf("%d-%02d-%02d", t.Year(), t.Month(), t.Day())
	default:
		timeSuffix = "default"
	}

	return strings.Replace(basePath, "$time_type", timeSuffix, 1), nil
}

// GetDB returns the database connection.
// In this refactored module, we assume one PlusDB instance per guild,
// so we don't need to manage a map of connections.
func (pdb *PlusDB) GetDB() *sql.DB {
	return pdb.db
}

// Close closes the database connection.
func (pdb *PlusDB) Close() error {
	pdb.mutex.Lock()
	defer pdb.mutex.Unlock()
	if pdb.db != nil {
		return pdb.db.Close()
	}
	return nil
}

// The following functions are migrated and adapted from the original message_db.go

// createMessagesTable creates the messages table if it doesn't exist
func createMessagesTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS messages (
        message_id INTEGER PRIMARY KEY,
        user_id INTEGER NOT NULL,
        guild_id INTEGER NOT NULL,
        channel_id INTEGER NOT NULL,
        timestamp INTEGER NOT NULL,
        message_content TEXT NOT NULL,
        attachments TEXT DEFAULT '',
        is_edited BOOLEAN DEFAULT FALSE
    );`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

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
	return nil
}

// createMessageDeletionsTable creates the message_deletions table if it doesn't exist
func createMessageDeletionsTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS message_deletions (
        deletion_id INTEGER PRIMARY KEY AUTOINCREMENT,
        message_id INTEGER NOT NULL,
        guild_id INTEGER NOT NULL,
        channel_id INTEGER NOT NULL,
        deletion_timestamp INTEGER NOT NULL
    );`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create message_deletions table: %w", err)
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_message_deletions_message_id ON message_deletions(message_id);",
		"CREATE INDEX IF NOT EXISTS idx_message_deletions_guild_timestamp ON message_deletions(guild_id, deletion_timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_message_deletions_channel_timestamp ON message_deletions(channel_id, deletion_timestamp);",
	}

	for _, indexQuery := range indexes {
		if _, err := db.Exec(indexQuery); err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}
	return nil
}

// createMessageEditsTable creates the message_edits table if it doesn't exist
func createMessageEditsTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS message_edits (
        edit_id INTEGER PRIMARY KEY AUTOINCREMENT,
        message_id INTEGER NOT NULL,
        guild_id INTEGER NOT NULL,
        channel_id INTEGER NOT NULL,
        original_content TEXT DEFAULT '',
        edited_content TEXT NOT NULL,
        original_attachments TEXT DEFAULT '',
        edited_attachments TEXT DEFAULT '',
        edit_timestamp INTEGER NOT NULL
    );`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create message_edits table: %w", err)
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_message_edits_message_id ON message_edits(message_id);",
		"CREATE INDEX IF NOT EXISTS idx_message_edits_guild_timestamp ON message_edits(guild_id, edit_timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_message_edits_channel_timestamp ON message_edits(channel_id, edit_timestamp);",
	}

	for _, indexQuery := range indexes {
		if _, err := db.Exec(indexQuery); err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}
	return nil
}

// InsertMessage inserts a message record into the database
func (pdb *PlusDB) InsertMessage(msg models.Message) error {
	db := pdb.GetDB()
	query := `INSERT OR IGNORE INTO messages (message_id, user_id, guild_id, channel_id, timestamp, message_content, attachments, is_edited) 
              VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(msg.MessageID, msg.UserID, msg.GuildID, msg.ChannelID, msg.Timestamp, msg.MessageContent, msg.Attachments, msg.IsEdited)
	if err != nil {
		return fmt.Errorf("failed to insert message %d: %w", msg.MessageID, err)
	}
	return nil
}

// InsertMessageDeletion records a message deletion event
func (pdb *PlusDB) InsertMessageDeletion(deletion models.MessageDeletion) error {
	db := pdb.GetDB()
	query := `INSERT INTO message_deletions (message_id, guild_id, channel_id, deletion_timestamp) 
              VALUES (?, ?, ?, ?)`

	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare deletion insert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(deletion.MessageID, deletion.GuildID, deletion.ChannelID, deletion.DeletionTimestamp)
	if err != nil {
		return fmt.Errorf("failed to insert message deletion %d: %w", deletion.MessageID, err)
	}
	return nil
}

// InsertMessageEdit records a message edit event
func (pdb *PlusDB) InsertMessageEdit(edit models.MessageEdit) error {
	db := pdb.GetDB()
	query := `INSERT INTO message_edits (message_id, guild_id, channel_id, original_content, edited_content, original_attachments, edited_attachments, edit_timestamp) 
              VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare edit insert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(edit.MessageID, edit.GuildID, edit.ChannelID, edit.OriginalContent, edit.EditedContent, edit.OriginalAttachments, edit.EditedAttachments, edit.EditTimestamp)
	if err != nil {
		return fmt.Errorf("failed to insert message edit %d: %w", edit.MessageID, err)
	}
	return nil
}

// GetMessage retrieves a single message by its ID from the database.
func (pdb *PlusDB) GetMessage(messageID int64) (*models.Message, error) {
	db := pdb.GetDB()
	query := `SELECT message_id, user_id, guild_id, channel_id, timestamp, message_content, attachments, is_edited
	             FROM messages WHERE message_id = ?`

	var msg models.Message
	err := db.QueryRow(query, messageID).Scan(
		&msg.MessageID, &msg.UserID, &msg.GuildID, &msg.ChannelID,
		&msg.Timestamp, &msg.MessageContent, &msg.Attachments, &msg.IsEdited,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil, nil if no message is found
		}
		return nil, fmt.Errorf("failed to query message %d: %w", messageID, err)
	}

	return &msg, nil
}
