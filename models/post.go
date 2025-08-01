package models

// ForumPost represents the structure of a forum post to be saved in the database.
// The table name will be the channel ID.
type ForumPost struct {
	AuthorID        string `db:"author_id"`
	ThreadID        string `db:"thread_id"` // Primary Key
	Title           string `db:"title"`
	Content         string `db:"content"` // First 1024 characters
	FirstImageURL   string `db:"first_image_url"`
	MessageCount    int    `db:"message_count"`
	CreationDate    int64  `db:"creation_date"`     // Unix timestamp
	LastMessageTime int64  `db:"last_message_time"` // Unix timestamp
	TagID           string `db:"tag_id"`
	ChannelID       string `db:"channel_id"`
}
