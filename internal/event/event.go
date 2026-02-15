package event

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Type represents the type of an event
type Type string

// Metadata defines the type for event metadata
type Metadata interface{}

// Event represents a generic event in the system
type Event struct {
	Version  string      `json:"version"` // Event schema version (e.g., "1.0")
	Type     Type        `json:"type"`
	Payload  interface{} `json:"payload"`
	Metadata Metadata    `json:"metadata"`
}

// GetMetadataValue extracts a value from the event metadata safely
func (e Event) GetMetadataValue(key string) interface{} {
	if e.Metadata == nil {
		return nil
	}

	// Check for map
	if m, ok := e.Metadata.(map[string]interface{}); ok {
		return m[key]
	}

	// Check for JobMetadata struct
	if m, ok := e.Metadata.(domain.JobMetadata); ok {
		if key == "source" {
			return m.Source
		}
	}

	return nil
}

// Common event types
const (
	ProgressionCycleCompleted Type = "progression.cycle.completed"
	ProgressionTargetSet      Type = "progression.target.set"
	ProgressionVotingStarted  Type = "progression.voting_started"
	ProgressionAllUnlocked    Type = "progression.all_unlocked"
	ProgressionNodeUnlocked   Type = "progression.node_unlocked"
	ProgressionNodeRelocked   Type = "progression.node_relocked"
	EventTypeEngagement       Type = "engagement"

	// Timeout event types
	TimeoutApplied Type = "timeout.applied"
	TimeoutCleared Type = "timeout.cleared"

	// Subscription event types
	SubscriptionActivated  Type = "subscription.activated"
	SubscriptionRenewed    Type = "subscription.renewed"
	SubscriptionUpgraded   Type = "subscription.upgraded"
	SubscriptionDowngraded Type = "subscription.downgraded"
	SubscriptionExpired    Type = "subscription.expired"
	SubscriptionCancelled  Type = "subscription.cancelled"
)

// Typed event payloads for type safety

// EngagementPayloadV1 is the typed payload for engagement events
type EngagementPayloadV1 struct {
	UserID       int64  `json:"user_id"`
	UserIDStr    string `json:"user_id_str,omitempty"` // UUID string form, used for Scholar XP
	PlatformID   int64  `json:"platform_id"`
	ActivityType string `json:"activity_type"`
	Timestamp    int64  `json:"timestamp"`
}

// ProgressionNodeInfo contains basic info about a progression node for events
type ProgressionNodeInfo struct {
	NodeKey     string `json:"node_key"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

// ProgressionCycleCompletedPayloadV1 is the typed payload for progression cycle events
type ProgressionCycleCompletedPayloadV1 struct {
	UnlockedNode ProgressionNodeInfo `json:"unlocked_node"`
}

// ProgressionTargetSetPayloadV1 is the typed payload for progression target events
type ProgressionTargetSetPayloadV1 struct {
	NodeKey      string `json:"node_key"`
	TargetLevel  int    `json:"target_level"`
	AutoSelected bool   `json:"auto_selected"`
	SessionID    int    `json:"session_id"`
}

// ProgressionVotingOptionV1 represents an option in a voting session event
type ProgressionVotingOptionV1 struct {
	NodeKey        string `json:"node_key"`
	DisplayName    string `json:"display_name"`
	Description    string `json:"description"`
	UnlockDuration string `json:"unlock_duration"`
}

// ProgressionVotingStartedPayloadV1 is the typed payload for voting started events
type ProgressionVotingStartedPayloadV1 struct {
	Options        []ProgressionVotingOptionV1 `json:"options"`
	PreviousUnlock string                      `json:"previous_unlock,omitempty"`
}

// ProgressionAllUnlockedPayloadV1 is the typed payload for all unlocked events
type ProgressionAllUnlockedPayloadV1 struct {
	Message string `json:"message"`
}

// JobLevelUpPayloadV1 is the typed payload for job level up events
type JobLevelUpPayloadV1 struct {
	UserID   string `json:"user_id"`
	JobKey   string `json:"job_key"`
	OldLevel int    `json:"old_level"`
	NewLevel int    `json:"new_level"`
	Source   string `json:"source,omitempty"`
}

// ToMap converts the payload to a map - REMOVED

// JobXPCriticalPayloadV1 is the typed payload for job XP critical (Epiphany) events
type JobXPCriticalPayloadV1 struct {
	UserID     string  `json:"user_id"`
	JobKey     string  `json:"job_key"`
	BaseXP     int     `json:"base_xp"`
	BonusXP    int     `json:"bonus_xp"`
	Multiplier float64 `json:"multiplier"`
	Source     string  `json:"source,omitempty"`
}

// ToMap converts the payload to a map - REMOVED

// DailyResetCompletePayloadV1 is the typed payload for daily reset complete events
type DailyResetCompletePayloadV1 struct {
	ResetTime       time.Time `json:"reset_time"`
	RecordsAffected int64     `json:"records_affected"`
}

// ToMap converts the payload to a map - REMOVED

// Type-safe event constructors

// NewEngagementEvent creates a new engagement event with type-safe payload
func NewEngagementEvent(userID, platformID int64, activityType string, userIDStr string) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    EventTypeEngagement,
		Payload: EngagementPayloadV1{
			UserID:       userID,
			UserIDStr:    userIDStr,
			PlatformID:   platformID,
			ActivityType: activityType,
			Timestamp:    time.Now().Unix(),
		},
		Metadata: nil,
	}
}

// NewProgressionCycleEvent creates a new progression cycle event
func NewProgressionCycleEvent(nodeKey, displayName, description string) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    ProgressionCycleCompleted,
		Payload: ProgressionCycleCompletedPayloadV1{
			UnlockedNode: ProgressionNodeInfo{
				NodeKey:     nodeKey,
				DisplayName: displayName,
				Description: description,
			},
		},
		Metadata: nil,
	}
}

// NewProgressionTargetEvent creates a new progression target event
func NewProgressionTargetEvent(nodeKey string, targetLevel int, autoSelected bool, sessionID int) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    ProgressionTargetSet,
		Payload: ProgressionTargetSetPayloadV1{
			NodeKey:      nodeKey,
			TargetLevel:  targetLevel,
			AutoSelected: autoSelected,
			SessionID:    sessionID,
		},
		Metadata: map[string]interface{}{
			"session_id": sessionID,
		},
	}
}

// NewProgressionVotingStartedEvent creates a new voting started event
func NewProgressionVotingStartedEvent(options []ProgressionVotingOptionV1, previousUnlock string, sessionID int) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    ProgressionVotingStarted,
		Payload: ProgressionVotingStartedPayloadV1{
			Options:        options,
			PreviousUnlock: previousUnlock,
		},
		Metadata: map[string]interface{}{
			"session_id": sessionID,
		},
	}
}

// NewProgressionAllUnlockedEvent creates a new all unlocked event
func NewProgressionAllUnlockedEvent(message string) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    ProgressionAllUnlocked,
		Payload: ProgressionAllUnlockedPayloadV1{
			Message: message,
		},
		Metadata: nil,
	}
}

// NewJobLevelUpEvent creates a new job level up event
func NewJobLevelUpEvent(userID, jobKey string, oldLevel, newLevel int, source string) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    Type(domain.EventTypeJobLevelUp),
		Payload: JobLevelUpPayloadV1{
			UserID:   userID,
			JobKey:   jobKey,
			OldLevel: oldLevel,
			NewLevel: newLevel,
			Source:   source,
		},
		Metadata: domain.JobMetadata{
			Source: source,
		},
	}
}

// NewJobXPCriticalEvent creates a new job XP critical event
func NewJobXPCriticalEvent(userID, jobKey string, baseXP, bonusXP int, multiplier float64, source string) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    Type(domain.EventTypeJobXPCritical),
		Payload: JobXPCriticalPayloadV1{
			UserID:     userID,
			JobKey:     jobKey,
			BaseXP:     baseXP,
			BonusXP:    bonusXP,
			Multiplier: multiplier,
			Source:     source,
		},
		Metadata: domain.JobMetadata{
			Source: source,
		},
	}
}

// NewDailyResetCompleteEvent creates a new daily reset complete event
func NewDailyResetCompleteEvent(resetTime time.Time, recordsAffected int64) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    Type(domain.EventTypeDailyResetComplete),
		Payload: DailyResetCompletePayloadV1{
			ResetTime:       resetTime,
			RecordsAffected: recordsAffected,
		},
		Metadata: nil,
	}
}

// GambleCompletedPayloadV1 is the typed payload for gamble completion events
type GambleCompletedPayloadV1 struct {
	GambleID         string `json:"gamble_id"`
	WinnerID         string `json:"winner_id"`
	TotalValue       int64  `json:"total_value"`
	ParticipantCount int    `json:"participant_count"`
	Timestamp        int64  `json:"timestamp"`
}

// TimeoutPayloadV1 is the typed payload for timeout events
type TimeoutPayloadV1 struct {
	Platform        string `json:"platform"`
	Username        string `json:"username"`
	Action          string `json:"action"` // "applied" or "cleared"
	DurationSeconds int    `json:"duration_seconds"`
	Reason          string `json:"reason,omitempty"`
	Timestamp       int64  `json:"timestamp"`
}

// SubscriptionPayloadV1 is the typed payload for subscription events
type SubscriptionPayloadV1 struct {
	UserID    string `json:"user_id"`
	Platform  string `json:"platform"`
	TierName  string `json:"tier_name"`
	Timestamp int64  `json:"timestamp"`
}

// NewTimeoutAppliedEvent creates a new timeout applied event
func NewTimeoutAppliedEvent(platform, username string, durationSeconds int, reason string) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    TimeoutApplied,
		Payload: TimeoutPayloadV1{
			Platform:        platform,
			Username:        username,
			Action:          "applied",
			DurationSeconds: durationSeconds,
			Reason:          reason,
			Timestamp:       time.Now().Unix(),
		},
		Metadata: nil,
	}
}

// NewTimeoutClearedEvent creates a new timeout cleared event
func NewTimeoutClearedEvent(platform, username string) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    TimeoutCleared,
		Payload: TimeoutPayloadV1{
			Platform:        platform,
			Username:        username,
			Action:          "cleared",
			DurationSeconds: 0,
			Timestamp:       time.Now().Unix(),
		},
		Metadata: nil,
	}
}

// NewGambleCompletedEvent creates a new gamble completed event with type-safe payload
func NewGambleCompletedEvent(gambleID, winnerID string, totalValue int64, participantCount int, participants []domain.GambleParticipantOutcome) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    "GambleCompleted",
		Payload: domain.GambleCompletedPayloadV2{
			GambleID:         gambleID,
			WinnerID:         winnerID,
			TotalValue:       totalValue,
			ParticipantCount: participantCount,
			Participants:     participants,
			Timestamp:        time.Now().Unix(),
		},
		Metadata: nil,
	}
}

// Handler is a function that handles an event
type Handler func(ctx context.Context, event Event) error

// Bus defines the interface for an event bus
type Bus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventType Type, handler Handler)
}

// MemoryBus is an in-memory implementation of the Event Bus
type MemoryBus struct {
	handlers map[Type][]Handler
	mu       sync.RWMutex
}

// NewMemoryBus creates a new MemoryBus
func NewMemoryBus() *MemoryBus {
	return &MemoryBus{
		handlers: make(map[Type][]Handler),
	}
}

// Publish publishes an event to all subscribers
func (b *MemoryBus) Publish(ctx context.Context, event Event) error {
	b.mu.RLock()
	handlers, ok := b.handlers[event.Type]
	b.mu.RUnlock()

	if !ok {
		return nil
	}

	// For now, we execute handlers synchronously.
	// In the future, or with configuration, we could dispatch these to a worker pool
	// or run them in goroutines.
	var errs []error
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf(LogMsgHandlerErrorFormat, len(errs), event.Type, errs)
	}

	return nil
}

// Subscribe subscribes a handler to an event type
func (b *MemoryBus) Subscribe(eventType Type, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
}
