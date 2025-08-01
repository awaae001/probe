package models

// Post represents a unified forum post structure for the database.
type Post struct {
	DBID          int64  `db:"db_id"`
	ThreadID      string `db:"thread_id"` // Unique
	ChannelID     string `db:"channel_id"`
	Title         string `db:"title"`
	Author        string `db:"author"`
	AuthorID      string `db:"author_id"`
	Content       string `db:"content"`
	Tags          string `db:"tags"`
	MessageCount  int    `db:"message_count"`
	Timestamp     int64  `db:"timestamp"` // Unix timestamp (CreationDate)
	CoverImageURL string `db:"cover_image_url"`
}

// Exclusion represents a thread that should be excluded from scanning.
type Exclusion struct {
	ThreadID  string `db:"thread_id"`
	GuildID   string `db:"guild_id"`
	ChannelID string `db:"channel_id"`
	Reason    string `db:"reason"`
	Timestamp int64  `db:"timestamp"`
}
