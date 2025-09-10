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