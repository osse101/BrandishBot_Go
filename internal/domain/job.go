package domain

import (
	"time"

	"github.com/google/uuid"
)

// Job represents a profession that users can level up
type Job struct {
	ID                 int       `json:"id"`
	JobKey             string    `json:"job_key"`      // "blacksmith", "explorer", etc.
	DisplayName        string    `json:"display_name"` // "Blacksmith"
	Description        string    `json:"description"`
	AssociatedFeatures []string  `json:"associated_features"` // ["upgrade", "craft"]
	CreatedAt          time.Time `json:"created_at"`
}

// UserJob tracks a user's progress in a specific job
type UserJob struct {
	UserID        string     `json:"user_id"`
	JobID         int        `json:"job_id"`
	CurrentXP     int64      `json:"current_xp"`
	CurrentLevel  int        `json:"current_level"`
	XPGainedToday int64      `json:"xp_gained_today"`
	LastXPGain    *time.Time `json:"last_xp_gain,omitempty"`
}

// JobXPEvent records an XP gain event for auditing
type JobXPEvent struct {
	ID             uuid.UUID              `json:"id"`
	UserID         string                 `json:"user_id"`
	JobID          int                    `json:"job_id"`
	XPAmount       int                    `json:"xp_amount"`
	SourceType     string                 `json:"source_type"`     // "upgrade", "search", "gamble"
	SourceMetadata map[string]interface{} `json:"source_metadata"` // flexible metadata
	RecordedAt     time.Time              `json:"recorded_at"`
}

// JobLevelBonus defines bonuses available at certain job levels
type JobLevelBonus struct {
	ID          int     `json:"id"`
	JobID       int     `json:"job_id"`
	MinLevel    int     `json:"min_level"`
	BonusType   string  `json:"bonus_type"`  // "bonus_money_chance", "prize_increase"
	BonusValue  float64 `json:"bonus_value"` // 0.25 = 25%
	Description string  `json:"description"`
}

// UserJobInfo combines job info with user progress for API responses
type UserJobInfo struct {
	JobKey        string `json:"job_key"`
	DisplayName   string `json:"display_name"`
	Level         int    `json:"level"`
	CurrentXP     int64  `json:"current_xp"`
	XPToNextLevel int64  `json:"xp_to_next_level"`
	MaxLevel      int    `json:"max_level"` // From progression system
}

// XPAwardResult contains the outcome of awarding XP
type XPAwardResult struct {
	JobKey    string `json:"job_key"`
	XPGained  int    `json:"xp_gained"`
	NewXP     int64  `json:"new_xp"`
	NewLevel  int    `json:"new_level"`
	LeveledUp bool   `json:"leveled_up"`
}

// DailyResetStatus shows the state of daily job XP resets
type DailyResetStatus struct {
	LastResetTime   time.Time `json:"last_reset_time"`
	NextResetTime   time.Time `json:"next_reset_time"`
	RecordsAffected int64     `json:"records_affected"`
}
