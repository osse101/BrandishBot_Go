package sse

// JobLevelUpPayload represents the SSE payload for job level up events
type JobLevelUpPayload struct {
	UserID   string `json:"user_id"`
	JobKey   string `json:"job_key"`
	OldLevel int    `json:"old_level"`
	NewLevel int    `json:"new_level"`
	Source   string `json:"source,omitempty"` // What activity caused the levelup (e.g., "search", "sell")
}

// VotingStartedPayload represents the SSE payload for voting session start
type VotingStartedPayload struct {
	NodeKey        string             `json:"node_key,omitempty"` // Set when auto-selected
	TargetLevel    int                `json:"target_level"`       // Set when auto-selected
	AutoSelected   bool               `json:"auto_selected"`      // True if only one option was available
	Options        []VotingOptionInfo `json:"options,omitempty"`  // Available voting options
	PreviousUnlock string             `json:"previous_unlock"`    // Node that was just unlocked
}

// CycleCompletedPayload represents the SSE payload for progression cycle completion
type CycleCompletedPayload struct {
	UnlockedNode NodeInfo `json:"unlocked_node"`
}

// NodeInfo contains basic information about a progression node
type NodeInfo struct {
	NodeKey     string `json:"node_key"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

// VotingOptionInfo contains information about a voting option
type VotingOptionInfo struct {
	NodeKey        string `json:"node_key"`
	DisplayName    string `json:"display_name"`
	Description    string `json:"description"`
	UnlockDuration string `json:"unlock_duration"` // "Short", "Medium", or "Long"
}

// AllUnlockedPayload represents the SSE payload when all nodes are unlocked
type AllUnlockedPayload struct {
	Message string `json:"message"`
}

// TimeoutPayload represents the SSE payload for timeout events
type TimeoutPayload struct {
	Platform        string `json:"platform"`
	Username        string `json:"username"`
	Action          string `json:"action"` // "applied" or "cleared"
	DurationSeconds int    `json:"duration_seconds"`
	Reason          string `json:"reason,omitempty"`
}
