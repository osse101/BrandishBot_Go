package domain

import "time"

// User represents a registered user
type User struct {
	ID        string    `json:"internal_id" db:"user_id"`
	Username  string    `json:"username" db:"username"`
	TwitchID  string    `json:"twitch_id,omitempty"`
	YoutubeID string    `json:"youtube_id,omitempty"`
	DiscordID string    `json:"discord_id,omitempty"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
