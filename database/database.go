package database

import (
	"database/sql"
	"discord-bot/models"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // Import the SQLite3 driver
)

// DB is the global database connection pool.
var DB *sql.DB

// InitDB initializes the database connection. It takes the database path as input.
func InitDB(dbPath string) error {
	// Ensure the directory for the database file exists.
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open the SQLite database. It will be created if it doesn't exist.
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Ping the database to verify the connection.
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Successfully connected to the database at", dbPath)
	return nil
}

// CreateTableForChannel creates a new table for a specific forum channel if it doesn't already exist.
// The table name is sanitized to be a valid SQL table name.
func CreateTableForChannel(channelID string) error {
	// Basic sanitization for table name
	tableName := "channel_" + channelID

	query := fmt.Sprintf(`
    CREATE TABLE IF NOT EXISTS %s (
        author_id TEXT,
        thread_id TEXT PRIMARY KEY,
        title TEXT,
        content TEXT,
        first_image_url TEXT,
        message_count INTEGER,
        creation_date INTEGER,
        last_message_time INTEGER,
        tag_id TEXT,
        channel_id TEXT
    );`, tableName)

	_, err := DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create table for channel %s: %w", channelID, err)
	}

	log.Printf("Table %s ensured to exist.", tableName)
	return nil
}

// SavePost saves a single forum post to the appropriate channel's table.
// It uses an "INSERT OR REPLACE" statement to handle both new posts and updates.
func SavePost(post models.ForumPost) error {
	tableName := "channel_" + post.ChannelID

	query := fmt.Sprintf(`
    INSERT OR REPLACE INTO %s (
        author_id, thread_id, title, content, first_image_url, 
        message_count, creation_date, last_message_time, tag_id, channel_id
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`, tableName)

	stmt, err := DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement for saving post: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		post.AuthorID,
		post.ThreadID,
		post.Title,
		post.Content,
		post.FirstImageURL,
		post.MessageCount,
		post.CreationDate,
		post.LastMessageTime,
		post.TagID,
		post.ChannelID,
	)
	if err != nil {
		return fmt.Errorf("failed to execute statement for saving post %s: %w", post.ThreadID, err)
	}

	return nil
}
