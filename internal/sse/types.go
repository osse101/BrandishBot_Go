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
	SessionID      int                `json:"session_id"`
	NodeKey        string             `json:"node_key,omitempty"`   // Set when auto-selected
	TargetLevel    int                `json:"target_level"`         // Set when auto-selected
	AutoSelected   bool               `json:"auto_selected"`        // True if only one option was available
	Options        []VotingOptionInfo `json:"options,omitempty"`    // Available voting options
	PreviousUnlock string             `json:"previous_unlock"`      // Node that was just unlocked
}

// CycleCompletedPayload represents the SSE payload for progression cycle completion
type CycleCompletedPayload struct {
	UnlockedNode  NodeInfo           `json:"unlocked_node"`
	VotingSession *VotingSessionInfo `json:"voting_session,omitempty"`
}

// NodeInfo contains basic information about a progression node
type NodeInfo struct {
	NodeKey     string `json:"node_key"`
	DisplayName string `json:"display_name"`
}

// VotingSessionInfo contains information about a voting session
type VotingSessionInfo struct {
	SessionID int                `json:"session_id"`
	Options   []VotingOptionInfo `json:"options"`
}

// VotingOptionInfo contains information about a voting option
type VotingOptionInfo struct {
	NodeKey     string `json:"node_key"`
	DisplayName string `json:"display_name"`
}
