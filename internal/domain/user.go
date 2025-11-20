package domain

// User represents a registered user
type User struct {
	InternalID string `json:"internal_id"`
	TwitchId   string `json:"twitch_id"`
	YoutubeId  string `json:"youtube_id"`
	DiscordId  string `json:"discord_id"`
	Username string `json:"username"`
}
