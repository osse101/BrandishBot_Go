package domain

import "time"

// User represents a registered user
type User struct {
	ID        string    `json:"internal_id"`
	Username  string    `json:"username"`
	TwitchID  string    `json:"twitch_id,omitempty"`
	YoutubeID string    `json:"youtube_id,omitempty"`
	DiscordID string    `json:"discord_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Cooldown action names
const (
	ActionSearch = "search"
)

// Cooldown durations
const (
	SearchCooldownDuration = 30 * time.Minute
)
