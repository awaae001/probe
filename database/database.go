package database

import (
	"database/sql"
	"discord-bot/models"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3" // Import the SQLite3 driver
)

// InitDB initializes the database connection. It takes the database path as input.
func InitDB(dbPath string) (*sql.DB, error) {
	// Ensure the directory for the database file exists.
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open the SQLite database. It will be created if it doesn't exist.
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Ping the database to verify the connection.
	if err = db.Ping(); err != nil {
		db.Close() // Close the connection if ping fails
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Create the exclusions table if it doesn't exist
	if err := createExclusionsTable(db); err != nil {
		db.Close() // Close the connection if table creation fails
		return nil, fmt.Errorf("failed to create exclusions table: %w", err)
	}

	log.Println("Successfully connected to the database at", dbPath)
	return db, nil
}

// CreateTableForChannel creates a new table for a specific forum channel if it doesn't already exist.
func CreateTableForChannel(db *sql.DB, tableName string) error {
	query := fmt.Sprintf(`
    CREATE TABLE IF NOT EXISTS %s (
        db_id INTEGER PRIMARY KEY AUTOINCREMENT,
        thread_id TEXT UNIQUE,
        channel_id TEXT,
        title TEXT,
        author TEXT,
        author_id TEXT,
        content TEXT,
        tags TEXT,
        message_count INTEGER,
        timestamp INTEGER,
        cover_image_url TEXT,
        total_reactions INTEGER,
        unique_reactions INTEGER,
        status TEXT DEFAULT 'active'
    );`, tableName)

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	// Add the 'status' column if it doesn't exist, with a default value.
	alterQuery := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN status TEXT DEFAULT 'active'`, tableName)
	if _, err := db.Exec(alterQuery); err != nil {
		// We expect an error if the column already exists, so we can ignore it.
		// A more robust solution would check the error message.
	}

	log.Printf("Table %s ensured to exist and is up to date.", tableName)
	return nil
}

// InsertPost saves a single forum post to the appropriate channel's table.
func InsertPost(db *sql.DB, post models.Post, tableName string) error {
	query := fmt.Sprintf(`
    INSERT OR IGNORE INTO %s (
        thread_id, channel_id, title, author, author_id, content, tags, message_count, timestamp, cover_image_url, total_reactions, unique_reactions
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`, tableName)

	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement for saving post: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		post.ThreadID,
		post.ChannelID,
		post.Title,
		post.Author,
		post.AuthorID,
		post.Content,
		post.Tags,
		post.MessageCount,
		post.Timestamp,
		post.CoverImageURL,
		post.TotalReactions,
		post.UniqueReactions,
	)
	if err != nil {
		return fmt.Errorf("failed to execute statement for saving post %s: %w", post.ThreadID, err)
	}

	return nil
}

// GetAllPostIDs retrieves all post IDs from a given table and returns them as a map for quick lookups.
func GetAllPostIDs(db *sql.DB, tableName string) (map[string]bool, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT thread_id FROM %s", tableName))
	if err != nil {
		// If the table doesn't exist, return an empty map instead of an error.
		if err.Error() == fmt.Sprintf("no such table: %s", tableName) {
			return make(map[string]bool), nil
		}
		return nil, fmt.Errorf("failed to query post IDs from table %s: %w", tableName, err)
	}
	defer rows.Close()

	postIDs := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan post ID: %w", err)
		}
		postIDs[id] = true
	}

	return postIDs, nil
}

// createExclusionsTable creates the 'exclusions' table if it doesn't exist.
func createExclusionsTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS exclusions (
        thread_id TEXT PRIMARY KEY,
        guild_id TEXT,
        channel_id TEXT,
        reason TEXT,
        timestamp INTEGER
    );`
	_, err := db.Exec(query)
	return err
}

// AddThreadToExclusionList adds a thread to the exclusion list.
func AddThreadToExclusionList(db *sql.DB, guildID, channelID, threadID, reason string) error {
	query := `INSERT OR REPLACE INTO exclusions (thread_id, guild_id, channel_id, reason, timestamp) VALUES (?, ?, ?, ?, ?)`
	stmt, err := db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(threadID, guildID, channelID, reason, time.Now().Unix())
	return err
}

// GetExcludedThreads returns a map of excluded thread IDs for a specific guild and channel.
func GetExcludedThreads(db *sql.DB, guildID, channelID string) (map[string]bool, error) {
	query := "SELECT thread_id FROM exclusions WHERE guild_id = ? AND channel_id = ?"
	rows, err := db.Query(query, guildID, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	excluded := make(map[string]bool)
	for rows.Next() {
		var threadID string
		if err := rows.Scan(&threadID); err != nil {
			return nil, err
		}
		excluded[threadID] = true
	}
	return excluded, nil
}
func UpdatePostStatus(db *sql.DB, tableName, threadID, status string) error {
	query := fmt.Sprintf(`UPDATE %s SET status = ? WHERE thread_id = ?`, tableName)

	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement for updating post status: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(status, threadID)
	if err != nil {
		return fmt.Errorf("failed to execute statement for updating post status for thread %s: %w", threadID, err)
	}

	return nil
}

// ArchiveAllPosts marks all posts in a table as archived
func ArchiveAllPosts(db *sql.DB, tableName string) error {
	query := fmt.Sprintf(`UPDATE %s SET status = 'archived' WHERE 1=1`, tableName)
	
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to archive all posts in table %s: %w", tableName, err)
	}
	
	log.Printf("All posts in table %s marked as archived", tableName)
	return nil
}

// UpsertActivePost inserts a new post or updates existing post with latest data and marks it as active
func UpsertActivePost(db *sql.DB, post models.Post, tableName string) error {
	query := fmt.Sprintf(`
    INSERT OR REPLACE INTO %s (
        thread_id, channel_id, title, author, author_id, content, tags, 
        message_count, timestamp, cover_image_url, total_reactions, unique_reactions, status
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active');`, tableName)

	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement for upserting active post: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		post.ThreadID,
		post.ChannelID,
		post.Title,
		post.Author,
		post.AuthorID,
		post.Content,
		post.Tags,
		post.MessageCount,
		post.Timestamp,
		post.CoverImageURL,
		post.TotalReactions,
		post.UniqueReactions,
	)
	if err != nil {
		return fmt.Errorf("failed to execute statement for upserting active post %s: %w", post.ThreadID, err)
	}

	return nil
}
