package models

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

// MessageEdit represents a message edit record
type MessageEdit struct {
	EditID              int64  `json:"edit_id"`              // Auto-increment ID for each edit
	MessageID           int64  `json:"message_id"`           // ID of the edited message
	GuildID             int64  `json:"guild_id"`             // Guild ID where the message was edited
	ChannelID           int64  `json:"channel_id"`           // Channel ID where the message was edited
	OriginalContent     string `json:"original_content"`     // Original message content
	EditedContent       string `json:"edited_content"`       // Edited message content
	OriginalAttachments string `json:"original_attachments"` // JSON array of original attachment URLs
	EditedAttachments   string `json:"edited_attachments"`   // JSON array of edited attachment URLs
	EditTimestamp       int64  `json:"edit_timestamp"`       // Timestamp when the edit occurred
}
