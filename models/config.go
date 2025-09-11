package models

import "time"

// ScanningFileConfig represents the top-level structure of the scanning_config.json file.
type ScanningFileConfig struct {
	ScanningConfig ScanningConfig `json:"scanning_config" mapstructure:"scanning_config"`
}

// It's a map where keys are guild IDs.
type ScanningConfig map[string]GuildConfig

// GuildConfig represents the configuration for a single guild.
type GuildConfig struct {
	Name     string                  `json:"name" mapstructure:"name"`
	GuildsID string                  `json:"guilds_id" mapstructure:"guilds_id"`
	DBPath   string                  `json:"db_path" mapstructure:"db_path"`
	Data     map[string]CategoryData `json:"data" mapstructure:"data"`
}

// CategoryData represents the data for a category to be scanned.
type CategoryData struct {
	CategoryName string   `json:"category_name" mapstructure:"category_name"`
	ID           string   `json:"id" mapstructure:"id"`
	ChannelID    []string `json:"channel_id" mapstructure:"channel_id"`
}

// ThreadConfig represents the structure of the thread_config.json file.
type ThreadConfig struct {
	ThreadConfig map[string]GuildThreadConfig `json:"thread_config" mapstructure:"thread_config"`
}

// GuildThreadConfig represents the configuration for a single guild's thread handling.
type GuildThreadConfig struct {
	Name      string `json:"name" mapstructure:"name"`
	Database  string `json:"database" mapstructure:"database"`
	TableName string `json:"tableName" mapstructure:"tableName"`
}

// CommandsConfig represents the commands configuration.
type CommandsConfig struct {
	AllowGuils []string   `mapstructure:"allowguils"`
	Auth       AuthConfig `mapstructure:"auth"`
}

// AuthConfig represents the authentication settings.
type AuthConfig struct {
	Developers  []string `mapstructure:"Developers"`
	AdminsRoles []string `mapstructure:"AdminsRoles"`
	Guest       []string `mapstructure:"Guest"`
}

// NewScanConfig 代表 new_scan.json 配置文件的结构
// 用于配置成员统计和监控功能
type NewScanConfig struct {
	// 数据库文件路径
	DBFilePath string `json:"db_file_path" mapstructure:"db_file_path"`
	// 服务器配置数据，键为服务器ID
	Data map[string]NewScanGuildData `json:"data" mapstructure:"data"`
}

// NewScanGuildData 代表单个服务器的成员统计配置
type NewScanGuildData struct {
	// 需要监控的身份组ID
	RoleID string `json:"role_id" mapstructure:"role_id"`
}

// MessageListenerFileConfig represents the top-level structure of the message_listener.json file.
type MessageListenerFileConfig struct {
	MessageListener MessageListenerConfig `json:"message_listener" mapstructure:"message_listener"`
}

// MessageListenerConfig holds the configuration for the message listener.
type MessageListenerConfig struct {
	DBStatus  string              `json:"db_status" mapstructure:"db_status"`
	GloabMode []string            `json:"gloab_mode" mapstructure:"gloab_mode"`
	Data      MessageListenerData `json:"data" mapstructure:"data"`
}

// MessageListenerData contains the configurations for different modes.
type MessageListenerData struct {
	BaseModeConfig map[string]BaseGuildConfig `json:"base_mode_config" mapstructure:"base_mode_config"`
	PlusModeConfig map[string]PlusGuildConfig `json:"plus_mode_config" mapstructure:"plus_mode_config"`
}

// BaseGuildConfig represents the base mode configuration for a single guild.
type BaseGuildConfig struct {
	GuildsID string   `json:"guilds_id" mapstructure:"guilds_id"`
	DBPath   string   `json:"db_path" mapstructure:"db_path"`
	Exclude  []string `json:"exclude" mapstructure:"exclude"`
}

// DBStatus represents the overall status of active databases, designed to be written to db_status.json.
type DBStatus struct {
	LastUpdated     time.Time                  `json:"last_updated"`
	ActiveDatabases map[string]*GuildDatabases `json:"active_databases"`
}

// GuildDatabases holds the database instances for a specific guild, categorized by mode.
type GuildDatabases struct {
	Base *DatabaseInfo `json:"base,omitempty"`
	Plus *DatabaseInfo `json:"plus,omitempty"`
}

// DatabaseInfo contains details about a single database instance.
type DatabaseInfo struct {
	DBPath string `json:"db_path"`
	Status string `json:"status"`
}

// PlusGuildConfig represents the plus mode configuration for a single guild.
type PlusGuildConfig struct {
	GuildsID string   `json:"guilds_id" mapstructure:"guilds_id"`
	DBPath   string   `json:"db_path" mapstructure:"db_path"`
	TimeType string   `json:"time_type" mapstructure:"time_type"`
	Exclude  []string `json:"exclude" mapstructure:"exclude"`
}
