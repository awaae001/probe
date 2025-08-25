package models

// ScanningConfig represents the structure of the scanning_config.json file.
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
type ThreadConfig map[string]GuildThreadConfig

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

// NewScamConfig represents the structure of the new_scam.json file.
type NewScamConfig map[string]NewScamGuildConfig

// NewScamGuildConfig represents the configuration for a single guild's scam monitoring.
type NewScamGuildConfig struct {
	Name     string `json:"name" mapstructure:"name"`
	Filepath string `json:"filepath" mapstructure:"filepath"`
	RoleID   string `json:"role_id" mapstructure:"role_id"`
}
