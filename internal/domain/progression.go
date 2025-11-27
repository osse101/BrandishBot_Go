package domain

import "time"

// ProgressionNode represents a node in the progression tree
type ProgressionNode struct {
	ID           int       `json:"id"`
	NodeKey      string    `json:"node_key"`
	NodeType     string    `json:"node_type"` // 'feature', 'item', 'mechanic', 'upgrade'
	DisplayName  string    `json:"display_name"`
	Description  string    `json:"description"`
	ParentNodeID *int      `json:"parent_node_id"` // NULL for root
	MaxLevel     int       `json:"max_level"`
	UnlockCost   int       `json:"unlock_cost"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
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
	TotalUnlocked   int                `json:"total_unlocked"`
	EngagementScore int                `json:"engagement_score"`
	ActiveVoting    *ProgressionVoting `json:"active_voting,omitempty"`
}

// EngagementBreakdown shows user's contribution by type
type EngagementBreakdown struct {
	MessagesSent int `json:"messages_sent"`
	CommandsUsed int `json:"commands_used"`
	ItemsCrafted int `json:"items_crafted"`
	ItemsUsed    int `json:"items_used"`
	TotalScore   int `json:"total_score"`
}
