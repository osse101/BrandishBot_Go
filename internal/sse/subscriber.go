package sse

import (
	"context"
	"log/slog"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

// Subscriber bridges the internal event bus to the SSE hub
type Subscriber struct {
	hub *Hub
	bus event.Bus
}

// NewSubscriber creates a new SSE subscriber
func NewSubscriber(hub *Hub, bus event.Bus) *Subscriber {
	return &Subscriber{
		hub: hub,
		bus: bus,
	}
}

// Subscribe registers handlers for all relevant event types
func (s *Subscriber) Subscribe() {
	// Subscribe to job level up events
	s.bus.Subscribe(event.Type(domain.EventJobLevelUp), s.handleJobLevelUp)

	// Subscribe to progression cycle completed events
	s.bus.Subscribe(event.ProgressionCycleCompleted, s.handleCycleCompleted)

	// Subscribe to progression voting started events
	s.bus.Subscribe(event.ProgressionVotingStarted, s.handleVotingStarted)

	// Subscribe to progression target set (can indicate auto-selected voting)
	s.bus.Subscribe(event.ProgressionTargetSet, s.handleTargetSet)

	// Subscribe to progression all unlocked events
	s.bus.Subscribe(event.ProgressionAllUnlocked, s.handleAllUnlocked)

	// Subscribe to timeout events
	s.bus.Subscribe(event.TimeoutApplied, s.handleTimeoutApplied)
	s.bus.Subscribe(event.TimeoutCleared, s.handleTimeoutCleared)

	// Subscribe to gamble completed events
	s.bus.Subscribe(event.Type(domain.EventGambleCompleted), s.handleGambleCompleted)

	// Subscribe to expedition events
	s.bus.Subscribe(event.Type(domain.EventExpeditionStarted), s.handleExpeditionStarted)
	s.bus.Subscribe(event.Type(domain.EventExpeditionTurn), s.handleExpeditionTurn)
	s.bus.Subscribe(event.Type(domain.EventExpeditionCompleted), s.handleExpeditionCompleted)

	// Subscribe to subscription events
	s.bus.Subscribe(event.SubscriptionActivated, s.handleSubscriptionEvent)
	s.bus.Subscribe(event.SubscriptionRenewed, s.handleSubscriptionEvent)
	s.bus.Subscribe(event.SubscriptionUpgraded, s.handleSubscriptionEvent)
	s.bus.Subscribe(event.SubscriptionDowngraded, s.handleSubscriptionEvent)
	s.bus.Subscribe(event.SubscriptionExpired, s.handleSubscriptionEvent)
	s.bus.Subscribe(event.SubscriptionCancelled, s.handleSubscriptionEvent)

	slog.Info("SSE subscriber registered for event types",
		"types", []string{
			string(domain.EventJobLevelUp),
			string(event.ProgressionCycleCompleted),
			string(event.ProgressionVotingStarted),
			string(event.ProgressionTargetSet),
			string(event.ProgressionAllUnlocked),
			string(event.TimeoutApplied),
			string(event.TimeoutCleared),
			string(domain.EventGambleCompleted),
			string(domain.EventExpeditionStarted),
			string(domain.EventExpeditionTurn),
			string(domain.EventExpeditionCompleted),
			string(event.SubscriptionActivated),
			string(event.SubscriptionRenewed),
			string(event.SubscriptionExpired),
			string(event.SubscriptionCancelled),
		})
}

// handleJobLevelUp processes job level up events and broadcasts to SSE clients
func (s *Subscriber) handleJobLevelUp(_ context.Context, evt event.Event) error {
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid job level up event payload type")
		return nil
	}

	// Extract source from metadata if available
	source := ""
	if evt.Metadata != nil {
		if src, ok := evt.Metadata["source"].(string); ok {
			source = src
		}
	}

	// Build SSE payload
	ssePayload := JobLevelUpPayload{
		UserID:   getStringFromMap(payload, "user_id"),
		JobKey:   getStringFromMap(payload, "job_key"),
		OldLevel: getIntFromMap(payload, "old_level"),
		NewLevel: getIntFromMap(payload, "new_level"),
		Source:   source,
	}

	s.hub.Broadcast(EventTypeJobLevelUp, ssePayload)

	slog.Debug(LogMsgEventBroadcast,
		"event_type", EventTypeJobLevelUp,
		"user_id", ssePayload.UserID,
		"job_key", ssePayload.JobKey,
		"new_level", ssePayload.NewLevel)

	return nil
}

// handleCycleCompleted processes progression cycle completed events
func (s *Subscriber) handleCycleCompleted(_ context.Context, evt event.Event) error {
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid cycle completed event payload type")
		return nil
	}

	// Build SSE payload
	ssePayload := CycleCompletedPayload{
		UnlockedNode: extractNodeInfo(payload["unlocked_node"]),
	}

	s.hub.Broadcast(EventTypeCycleCompleted, ssePayload)

	slog.Debug(LogMsgEventBroadcast,
		"event_type", EventTypeCycleCompleted,
		"unlocked_node", ssePayload.UnlockedNode.NodeKey)

	return nil
}

// handleVotingStarted processes voting session started events
func (s *Subscriber) handleVotingStarted(_ context.Context, evt event.Event) error {
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid voting started event payload type")
		return nil
	}

	// Build SSE payload
	ssePayload := VotingStartedPayload{
		PreviousUnlock: getStringFromMap(payload, "previous_unlock"),
		AutoSelected:   false,
	}

	// Extract options from payload
	if opts, ok := payload["options"].([]map[string]interface{}); ok {
		ssePayload.Options = make([]VotingOptionInfo, 0, len(opts))
		for _, opt := range opts {
			ssePayload.Options = append(ssePayload.Options, VotingOptionInfo{
				NodeKey:        getStringFromMap(opt, "node_key"),
				DisplayName:    getStringFromMap(opt, "display_name"),
				Description:    getStringFromMap(opt, "description"),
				UnlockDuration: getStringFromMap(opt, "unlock_duration"),
			})
		}
	} else if opts, ok := payload["options"].([]interface{}); ok {
		ssePayload.Options = make([]VotingOptionInfo, 0, len(opts))
		for _, opt := range opts {
			if optMap, ok := opt.(map[string]interface{}); ok {
				ssePayload.Options = append(ssePayload.Options, VotingOptionInfo{
					NodeKey:        getStringFromMap(optMap, "node_key"),
					DisplayName:    getStringFromMap(optMap, "display_name"),
					Description:    getStringFromMap(optMap, "description"),
					UnlockDuration: getStringFromMap(optMap, "unlock_duration"),
				})
			}
		}
	}

	s.hub.Broadcast(EventTypeVotingStarted, ssePayload)

	slog.Debug(LogMsgEventBroadcast,
		"event_type", EventTypeVotingStarted,
		"options_count", len(ssePayload.Options))

	return nil
}

// handleTargetSet processes progression target set events (voting started)
func (s *Subscriber) handleTargetSet(_ context.Context, evt event.Event) error {
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid target set event payload type")
		return nil
	}

	// Only broadcast if this was auto-selected (voting not required)
	// Normal voting start is handled by cycle_completed
	if autoSelected, ok := payload["auto_selected"].(bool); ok && autoSelected {
		ssePayload := VotingStartedPayload{
			NodeKey:        getStringFromMap(payload, "node_key"),
			TargetLevel:    getIntFromMap(payload, "target_level"),
			AutoSelected:   true,
			PreviousUnlock: "",
		}

		s.hub.Broadcast(EventTypeVotingStarted, ssePayload)

		slog.Debug(LogMsgEventBroadcast,
			"event_type", EventTypeVotingStarted,
			"node_key", ssePayload.NodeKey,
			"auto_selected", true)
	}

	return nil
}

// handleAllUnlocked processes progression all unlocked events
func (s *Subscriber) handleAllUnlocked(_ context.Context, evt event.Event) error {
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid all unlocked event payload type")
		return nil
	}

	ssePayload := AllUnlockedPayload{
		Message: getStringFromMap(payload, "message"),
	}

	s.hub.Broadcast(EventTypeAllUnlocked, ssePayload)

	slog.Debug(LogMsgEventBroadcast,
		"event_type", EventTypeAllUnlocked)

	return nil
}

// Helper functions for type conversion

func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getIntFromMap(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func extractNodeInfo(v interface{}) NodeInfo {
	if v == nil {
		return NodeInfo{}
	}

	// Check if it's already a domain.ProgressionNode
	switch node := v.(type) {
	case *domain.ProgressionNode:
		return NodeInfo{
			NodeKey:     node.NodeKey,
			DisplayName: node.DisplayName,
			Description: node.Description,
		}
	case domain.ProgressionNode:
		return NodeInfo{
			NodeKey:     node.NodeKey,
			DisplayName: node.DisplayName,
			Description: node.Description,
		}
	case map[string]interface{}:
		return NodeInfo{
			NodeKey:     getStringFromMap(node, "node_key"),
			DisplayName: getStringFromMap(node, "display_name"),
			Description: getStringFromMap(node, "description"),
		}
	}

	return NodeInfo{}
}

// handleTimeoutApplied processes timeout applied events
func (s *Subscriber) handleTimeoutApplied(_ context.Context, evt event.Event) error {
	// Try typed payload first
	if payload, ok := evt.Payload.(event.TimeoutPayloadV1); ok {
		ssePayload := TimeoutPayload{
			Platform:        payload.Platform,
			Username:        payload.Username,
			Action:          payload.Action,
			DurationSeconds: payload.DurationSeconds,
			Reason:          payload.Reason,
		}

		s.hub.Broadcast(EventTypeTimeoutApplied, ssePayload)

		slog.Debug(LogMsgEventBroadcast,
			"event_type", EventTypeTimeoutApplied,
			"platform", payload.Platform,
			"username", payload.Username,
			"duration_seconds", payload.DurationSeconds)

		return nil
	}

	// Fall back to map parsing
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid timeout applied event payload type")
		return nil
	}

	ssePayload := TimeoutPayload{
		Platform:        getStringFromMap(payload, "platform"),
		Username:        getStringFromMap(payload, "username"),
		Action:          getStringFromMap(payload, "action"),
		DurationSeconds: getIntFromMap(payload, "duration_seconds"),
		Reason:          getStringFromMap(payload, "reason"),
	}

	s.hub.Broadcast(EventTypeTimeoutApplied, ssePayload)

	slog.Debug(LogMsgEventBroadcast,
		"event_type", EventTypeTimeoutApplied,
		"platform", ssePayload.Platform,
		"username", ssePayload.Username)

	return nil
}

// handleTimeoutCleared processes timeout cleared events
func (s *Subscriber) handleTimeoutCleared(_ context.Context, evt event.Event) error {
	// Try typed payload first
	if payload, ok := evt.Payload.(event.TimeoutPayloadV1); ok {
		ssePayload := TimeoutPayload{
			Platform:        payload.Platform,
			Username:        payload.Username,
			Action:          payload.Action,
			DurationSeconds: 0,
		}

		s.hub.Broadcast(EventTypeTimeoutCleared, ssePayload)

		slog.Debug(LogMsgEventBroadcast,
			"event_type", EventTypeTimeoutCleared,
			"platform", payload.Platform,
			"username", payload.Username)

		return nil
	}

	// Fall back to map parsing
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid timeout cleared event payload type")
		return nil
	}

	ssePayload := TimeoutPayload{
		Platform:        getStringFromMap(payload, "platform"),
		Username:        getStringFromMap(payload, "username"),
		Action:          "cleared",
		DurationSeconds: 0,
	}

	s.hub.Broadcast(EventTypeTimeoutCleared, ssePayload)

	slog.Debug(LogMsgEventBroadcast,
		"event_type", EventTypeTimeoutCleared,
		"platform", ssePayload.Platform,
		"username", ssePayload.Username)

	return nil
}

// handleExpeditionStarted processes expedition start events
func (s *Subscriber) handleExpeditionStarted(_ context.Context, evt event.Event) error {
	if exp, ok := evt.Payload.(*domain.Expedition); ok {
		ssePayload := ExpeditionStartedPayload{
			ExpeditionID: exp.ID.String(),
			JoinDeadline: exp.JoinDeadline.Format("2006-01-02 15:04:05"),
		}
		s.hub.Broadcast(EventTypeExpeditionStarted, ssePayload)
		slog.Debug(LogMsgEventBroadcast, "event_type", EventTypeExpeditionStarted, "expedition_id", exp.ID)
		return nil
	}

	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid expedition started event payload type")
		return nil
	}

	ssePayload := ExpeditionStartedPayload{
		ExpeditionID: getStringFromMap(payload, "expedition_id"),
		JoinDeadline: getStringFromMap(payload, "join_deadline"),
	}
	s.hub.Broadcast(EventTypeExpeditionStarted, ssePayload)
	return nil
}

// handleExpeditionTurn processes expedition turn events
func (s *Subscriber) handleExpeditionTurn(_ context.Context, evt event.Event) error {
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid expedition turn event payload type")
		return nil
	}

	ssePayload := ExpeditionTurnPayload{
		ExpeditionID: getStringFromMap(payload, "expedition_id"),
		TurnNumber:   getIntFromMap(payload, "turn_number"),
		Narrative:    getStringFromMap(payload, "narrative"),
		Fatigue:      getIntFromMap(payload, "fatigue"),
		Purse:        getIntFromMap(payload, "purse"),
	}

	s.hub.Broadcast(EventTypeExpeditionTurn, ssePayload)
	return nil
}

// handleExpeditionCompleted processes expedition completion events
func (s *Subscriber) handleExpeditionCompleted(_ context.Context, evt event.Event) error {
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid expedition completed event payload type")
		return nil
	}

	ssePayload := ExpeditionCompletedPayload{
		ExpeditionID: getStringFromMap(payload, "expedition_id"),
		TotalTurns:   getIntFromMap(payload, "total_turns"),
		Won:          getBoolFromMap(payload, "won"),
		AllKO:        getBoolFromMap(payload, "all_ko"),
	}

	s.hub.Broadcast(EventTypeExpeditionCompleted, ssePayload)
	slog.Debug(LogMsgEventBroadcast, "event_type", EventTypeExpeditionCompleted, "expedition_id", ssePayload.ExpeditionID)
	return nil
}

// handleGambleCompleted processes gamble completion events
func (s *Subscriber) handleGambleCompleted(_ context.Context, evt event.Event) error {
	payload, ok := evt.Payload.(event.GambleCompletedPayloadV1)
	if !ok {
		// Fallback for untyped payload
		payloadMap, ok := evt.Payload.(map[string]interface{})
		if !ok {
			slog.Warn("Invalid gamble completed event payload type")
			return nil
		}
		payload = event.GambleCompletedPayloadV1{
			GambleID:         getStringFromMap(payloadMap, "gamble_id"),
			WinnerID:         getStringFromMap(payloadMap, "winner_id"),
			TotalValue:       int64(getIntFromMap(payloadMap, "total_value")),
			ParticipantCount: getIntFromMap(payloadMap, "participant_count"),
			Timestamp:        int64(getIntFromMap(payloadMap, "timestamp")),
		}
	}

	ssePayload := GambleCompletedPayload{
		GambleID:         payload.GambleID,
		WinnerID:         payload.WinnerID,
		TotalValue:       payload.TotalValue,
		ParticipantCount: payload.ParticipantCount,
		Timestamp:        payload.Timestamp,
	}

	s.hub.Broadcast(EventTypeGambleCompleted, ssePayload)

	slog.Debug(LogMsgEventBroadcast,
		"event_type", EventTypeGambleCompleted,
		"gamble_id", ssePayload.GambleID,
		"winner_id", ssePayload.WinnerID)

	return nil
}

// handleSubscriptionEvent processes subscription lifecycle events
func (s *Subscriber) handleSubscriptionEvent(_ context.Context, evt event.Event) error {
	// Try typed payload first
	if payload, ok := evt.Payload.(event.SubscriptionPayloadV1); ok {
		ssePayload := SubscriptionPayload{
			UserID:    payload.UserID,
			Platform:  payload.Platform,
			TierName:  payload.TierName,
			EventType: string(evt.Type),
			Timestamp: payload.Timestamp,
		}

		s.hub.Broadcast(EventTypeSubscription, ssePayload)

		slog.Debug(LogMsgEventBroadcast,
			"event_type", EventTypeSubscription,
			"sub_event", evt.Type,
			"user_id", payload.UserID,
			"platform", payload.Platform,
			"tier", payload.TierName)

		return nil
	}

	// Fall back to map parsing
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid subscription event payload type")
		return nil
	}

	ssePayload := SubscriptionPayload{
		UserID:    getStringFromMap(payload, "user_id"),
		Platform:  getStringFromMap(payload, "platform"),
		TierName:  getStringFromMap(payload, "tier_name"),
		EventType: string(evt.Type),
		Timestamp: int64(getIntFromMap(payload, "timestamp")),
	}

	s.hub.Broadcast(EventTypeSubscription, ssePayload)

	slog.Debug(LogMsgEventBroadcast,
		"event_type", EventTypeSubscription,
		"sub_event", evt.Type,
		"user_id", ssePayload.UserID,
		"platform", ssePayload.Platform)

	return nil
}
