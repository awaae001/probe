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
