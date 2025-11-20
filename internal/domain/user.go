package domain

// User represents a registered user
type User struct {
	ID         string `json:"internal_id"`
	Username   string `json:"username"`
	PlatformID string `json:"platform_id"`
	Platform   string `json:"platform"`
}
