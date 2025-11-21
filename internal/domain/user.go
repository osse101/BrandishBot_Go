package domain

// User represents a registered user
type User struct {
	ID        string `json:"internal_id"`
	Username  string `json:"username"`
	TwitchID  string `json:"twitch_id,omitempty"`
	YoutubeID string `json:"youtube_id,omitempty"`
	DiscordID string `json:"discord_id,omitempty"`
}
