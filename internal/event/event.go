package event

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Type represents the type of an event
type Type string

// Event represents a generic event in the system
type Event struct {
	Version  string                 `json:"version"` // Event schema version (e.g., "1.0")
	Type     Type                   `json:"type"`
	Payload  interface{}            `json:"payload"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Common event types
const (
	ProgressionCycleCompleted Type = "progression.cycle.completed"
	ProgressionTargetSet      Type = "progression.target.set"
	ProgressionVotingStarted  Type = "progression.voting_started"
	ProgressionAllUnlocked    Type = "progression.all_unlocked"
	EventTypeEngagement       Type = "engagement"
)

// Typed event payloads for type safety

// EngagementPayloadV1 is the typed payload for engagement events
type EngagementPayloadV1 struct {
	UserID       int64  `json:"user_id"`
	PlatformID   int64  `json:"platform_id"`
	ActivityType string `json:"activity_type"`
	Timestamp    int64  `json:"timestamp"`
}

// ProgressionCyclePayloadV1 is the typed payload for progression cycle events
type ProgressionCyclePayloadV1 struct {
	CycleID   int64  `json:"cycle_id"`
	NodeKey   string `json:"node_key,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// ProgressionTargetPayloadV1 is the typed payload for progression target events
type ProgressionTargetPayloadV1 struct {
	NodeKey   string `json:"node_key"`
	Timestamp int64  `json:"timestamp"`
}

// Type-safe event constructors

// NewEngagementEvent creates a new engagement event with type-safe payload
func NewEngagementEvent(userID, platformID int64, activityType string) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    EventTypeEngagement,
		Payload: EngagementPayloadV1{
			UserID:       userID,
			PlatformID:   platformID,
			ActivityType: activityType,
			Timestamp:    time.Now().Unix(),
		},
		Metadata: make(map[string]interface{}),
	}
}

// NewProgressionCycleEvent creates a new progression cycle event
func NewProgressionCycleEvent(cycleID int64, nodeKey string) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    ProgressionCycleCompleted,
		Payload: ProgressionCyclePayloadV1{
			CycleID:   cycleID,
			NodeKey:   nodeKey,
			Timestamp: time.Now().Unix(),
		},
		Metadata: make(map[string]interface{}),
	}
}

// NewProgressionTargetEvent creates a new progression target event
func NewProgressionTargetEvent(nodeKey string) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    ProgressionTargetSet,
		Payload: ProgressionTargetPayloadV1{
			NodeKey:   nodeKey,
			Timestamp: time.Now().Unix(),
		},
		Metadata: make(map[string]interface{}),
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

// NewGambleCompletedEvent creates a new gamble completed event with type-safe payload
func NewGambleCompletedEvent(gambleID, winnerID string, totalValue int64, participantCount int) Event {
	return Event{
		Version: EventSchemaVersion,
		Type:    "GambleCompleted",
		Payload: GambleCompletedPayloadV1{
			GambleID:         gambleID,
			WinnerID:         winnerID,
			TotalValue:       totalValue,
			ParticipantCount: participantCount,
			Timestamp:        time.Now().Unix(),
		},
		Metadata: make(map[string]interface{}),
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
