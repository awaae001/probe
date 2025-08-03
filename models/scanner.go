package models

import (
	"database/sql"
	"sync"

	"github.com/bwmarrin/discordgo"
)

// PartitionTask represents a scanning task for a single forum channel.
type PartitionTask struct {
	DB                 *sql.DB
	GuildConfig        *GuildConfig
	ChannelID          string // The specific channel to scan
	Key                string // The category key, for context
	IsFullScan         bool
	PartitionsDone     *int64
	TotalNewPostsFound *int64
	Wg                 *sync.WaitGroup
}

// ThreadChunk is a slice of threads to be processed concurrently.
type ThreadChunk struct {
	Threads []*discordgo.Channel
	Index   int
}
