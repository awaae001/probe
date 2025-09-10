package models

import "time"

// MessageConfig represents the message_init.json configuration structure
type MessageConfig struct {
	MessageListener MessageListener `json:"message_listener" mapstructure:"message_listener"`
}

// MessageListener represents the message listener configuration
type MessageListener struct {
	DBStatus string                    `json:"db_status" mapstructure:"db_status"`
	Data     map[string]GuildMsgConfig `json:"data" mapstructure:"data"` // key is guild_id
}

// GuildMsgConfig represents the configuration for a single guild's message collection
type GuildMsgConfig struct {
	GuildsID string   `json:"guilds_id" mapstructure:"guilds_id"`
	DBPath   string   `json:"db_path" mapstructure:"db_path"`     // supports $time_type placeholder
	TimeType string   `json:"time_type" mapstructure:"time_type"` // week/month/day
	Exclude  []string `json:"exclude" mapstructure:"exclude"`     // excluded channel IDs
}

// Message represents a Discord message record
type Message struct {
	MessageID      int64  `json:"message_id"`
	UserID         int64  `json:"user_id"`
	GuildID        int64  `json:"guild_id"`
	ChannelID      int64  `json:"channel_id"`
	Timestamp      int64  `json:"timestamp"`
	MessageContent string `json:"message_content"`
	Attachments    string `json:"attachments"` // JSON array of attachment URLs
	IsEdited       bool   `json:"is_edited"`   // Flag indicating if the message was edited
}

// MessageEdit represents a message edit history record
type MessageEdit struct {
	EditID              int64  `json:"edit_id"`              // Auto-increment ID for each edit
	MessageID           int64  `json:"message_id"`           // Reference to original message ID
	GuildID             int64  `json:"guild_id"`             // Guild ID (redundant storage for cross-database queries)
	EditTimestamp       int64  `json:"edit_timestamp"`       // Timestamp when the edit occurred
	PreviousContent     string `json:"previous_content"`     // Content before the edit
	NewContent          string `json:"new_content"`          // Content after the edit
	PreviousAttachments string `json:"previous_attachments"` // Attachments before the edit (JSON)
	NewAttachments      string `json:"new_attachments"`      // Attachments after the edit (JSON)
}

// ChannelStat represents channel message statistics
type ChannelStat struct {
	ChannelID    int64 `json:"channel_id"`
	MessageCount int64 `json:"message_count"`
}

// UserStat represents user message statistics
type UserStat struct {
	UserID       int64 `json:"user_id"`
	MessageCount int64 `json:"message_count"`
}

// MessageDeletion represents a message deletion record
type MessageDeletion struct {
	DeletionID        int64 `json:"deletion_id"`        // Auto-increment ID for each deletion
	MessageID         int64 `json:"message_id"`         // ID of the deleted message
	GuildID           int64 `json:"guild_id"`           // Guild ID where the message was deleted
	ChannelID         int64 `json:"channel_id"`         // Channel ID where the message was deleted
	DeletionTimestamp int64 `json:"deletion_timestamp"` // Timestamp when the deletion occurred
}

// DatabaseStatus represents the structure of db_status.json
type DatabaseStatus struct {
	CurrentDatabases []DatabaseInfo `json:"current_databases"`
	TotalDatabases   int            `json:"total_databases"`
	LastUpdated      time.Time      `json:"last_updated"`
}

// DatabaseInfo represents information about an active database
type DatabaseInfo struct {
	GuildID      string    `json:"guild_id"`
	DBFile       string    `json:"db_file"`
	CreatedAt    time.Time `json:"created_at"`
	MessageCount int64     `json:"message_count"`
	LastMessage  time.Time `json:"last_message"`
}