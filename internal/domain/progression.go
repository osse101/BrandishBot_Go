package domain

import "time"

// ProgressionNode represents a node in the progression tree
type ProgressionNode struct {
	ID          int    `json:"id"`
	NodeKey     string `json:"node_key"`
	NodeType    string `json:"node_type"` // 'feature', 'item', 'mechanic', 'upgrade'
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	MaxLevel    int    `json:"max_level"`
	UnlockCost  int    `json:"unlock_cost"` // Calculated from tier+size, stored for performance

	// Dynamic cost calculation fields (v2.0)
	Tier     int    `json:"tier"`     // 0-4: Foundation → Endgame
	Size     string `json:"size"`     // small, medium, large
	Category string `json:"category"` // Grouping: economy, combat, etc.

	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	// Value modification configuration (nullable JSON in DB)
	ModifierConfig *ModifierConfig `json:"modifier_config,omitempty"`
}

// ModifierConfig defines how a progression node modifies feature values
type ModifierConfig struct {
	FeatureKey    string   `json:"feature_key"`
	ModifierType  string   `json:"modifier_type"`
	PerLevelValue float64  `json:"per_level_value"`
	BaseValue     float64  `json:"base_value"`
	MaxValue      *float64 `json:"max_value,omitempty"`
	MinValue      *float64 `json:"min_value,omitempty"`
}

// ProgressionUnlock represents a globally unlocked node
type ProgressionUnlock struct {
	ID              int       `json:"id"`
	NodeID          int       `json:"node_id"`
	CurrentLevel    int       `json:"current_level"`
	UnlockedAt      time.Time `json:"unlocked_at"`
	UnlockedBy      string    `json:"unlocked_by"` // 'vote', 'admin', 'auto', 'instant_override'
	EngagementScore int       `json:"engagement_score"`
}

// ProgressionVoting represents active voting for a node
type ProgressionVoting struct {
	ID              int        `json:"id"`
	NodeID          int        `json:"node_id"`
	TargetLevel     int        `json:"target_level"`
	VoteCount       int        `json:"vote_count"`
	VotingStartedAt time.Time  `json:"voting_started_at"`
	VotingEndsAt    *time.Time `json:"voting_ends_at"`
	IsActive        bool       `json:"is_active"`
}

// UserVote represents a user's vote for a specific node/level
type UserVote struct {
	UserID      string    `json:"user_id"`
	NodeID      int       `json:"node_id"`
	TargetLevel int       `json:"target_level"`
	VotedAt     time.Time `json:"voted_at"`
}

// UserProgression represents individual user progression (recipes)
type UserProgression struct {
	UserID          string                 `json:"user_id"`
	ProgressionType string                 `json:"progression_type"` // 'recipe'
	ProgressionKey  string                 `json:"progression_key"`
	UnlockedAt      time.Time              `json:"unlocked_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// EngagementMetric tracks user engagement events
type EngagementMetric struct {
	ID          int                    `json:"id"`
	UserID      string                 `json:"user_id"`
	MetricType  string                 `json:"metric_type"` // 'message', 'command', 'item_crafted', 'item_used'
	MetricValue int                    `json:"metric_value"`
	RecordedAt  time.Time              `json:"recorded_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EngagementWeight defines the weight for each metric type
type EngagementWeight struct {
	MetricType  string    `json:"metric_type"`
	Weight      float64   `json:"weight"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProgressionReset records tree reset events
type ProgressionReset struct {
	ID                     int       `json:"id"`
	ResetAt                time.Time `json:"reset_at"`
	ResetBy                string    `json:"reset_by"`
	Reason                 string    `json:"reason"`
	NodesResetCount        int       `json:"nodes_reset_count"`
	EngagementScoreAtReset int       `json:"engagement_score_at_reset"`
}

// ProgressionTreeNode combines node info with unlock status for display
type ProgressionTreeNode struct {
	ProgressionNode
	IsUnlocked    bool  `json:"is_unlocked"`
	UnlockedLevel int   `json:"unlocked_level"` // 0 if not unlocked
	Children      []int `json:"children"`       // Child node IDs
}

// ProgressionStatus represents current community status
type ProgressionStatus struct {
	TotalUnlocked        int                       `json:"total_unlocked"`
	TotalNodes           int                       `json:"total_nodes"`
	AllNodesUnlocked     bool                      `json:"all_nodes_unlocked"`
	ContributionScore    int                       `json:"contribution_score"`
	ActiveSession        *ProgressionVotingSession `json:"active_session,omitempty"`
	ActiveUnlockProgress *UnlockProgress           `json:"active_unlock_progress,omitempty"`
	IsTransitioning      bool                      `json:"is_transitioning"` // Bug #5: True when between unlock completion and new session start
}

// ContributionBreakdown shows user's contribution by type
type ContributionBreakdown struct {
	MessagesSent int            `json:"messages_sent"`
	CommandsUsed int            `json:"commands_used"`
	ItemsCrafted int            `json:"items_crafted"`
	ItemsUsed    int            `json:"items_used"`
	TotalScore   int            `json:"total_score"`
	ByType       map[string]int `json:"by_type,omitempty"`
}

// ProgressionVotingSession represents a voting session for selecting next unlock
type ProgressionVotingSession struct {
	ID              int                       `json:"id"`
	StartedAt       time.Time                 `json:"started_at"`
	EndedAt         *time.Time                `json:"ended_at,omitempty"`
	VotingDeadline  time.Time                 `json:"voting_deadline"`
	WinningOptionID *int                      `json:"winning_option_id,omitempty"`
	Status          string                    `json:"status"` // 'voting', 'completed'
	Options         []ProgressionVotingOption `json:"options,omitempty"`
}

// ProgressionVotingOption represents one voting choice in a session
type ProgressionVotingOption struct {
	ID                  int              `json:"id"`
	SessionID           int              `json:"session_id"`
	NodeID              int              `json:"node_id"`
	TargetLevel         int              `json:"target_level"`
	VoteCount           int              `json:"vote_count"`
	LastHighestVoteAt   *time.Time       `json:"last_highest_vote_at"` // When first reached current highest
	NodeDetails         *ProgressionNode `json:"node_details,omitempty"`
	EstimatedUnlockDate *time.Time       `json:"estimated_unlock_date,omitempty"`
}

// UnlockProgress tracks contribution points accumulated toward next unlock
type UnlockProgress struct {
	ID                       int        `json:"id"`
	NodeID                   *int       `json:"node_id"`      // NULL until vote ends
	TargetLevel              *int       `json:"target_level"` // NULL until vote ends
	ContributionsAccumulated int        `json:"contributions_accumulated"`
	StartedAt                time.Time  `json:"started_at"`
	UnlockedAt               *time.Time `json:"unlocked_at"`
	VotingSessionID          *int       `json:"voting_session_id"`
	EstimatedUnlockDate      *time.Time `json:"estimated_unlock_date,omitempty"`
}

// ContributionLeaderboardEntry represents a user's rank and contribution total
type ContributionLeaderboardEntry struct {
	UserID       string `json:"user_id"`
	Contribution int    `json:"contribution"`
	Rank         int    `json:"rank"`
}

// VelocityMetrics holds engagement velocity data
type VelocityMetrics struct {
	PointsPerDay float64 `json:"points_per_day"`
	Trend        string  `json:"trend"` // "increasing", "stable", "decreasing"
	PeriodDays   int     `json:"period_days"`
	SampleSize   int     `json:"sample_size"`
	TotalPoints  int     `json:"total_points"`
}

// UnlockEstimate holds prediction data for node unlock
type UnlockEstimate struct {
	NodeKey             string     `json:"node_key"`
	EstimatedDays       float64    `json:"estimated_days"`
	Confidence          string     `json:"confidence"` // "high", "medium", "low"
	RequiredPoints      int        `json:"required_points"`
	CurrentProgress     int        `json:"current_progress"`
	CurrentVelocity     float64    `json:"current_velocity"`
	EstimatedUnlockDate *time.Time `json:"estimated_unlock_date"`
}
