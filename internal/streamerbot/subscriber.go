package streamerbot

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

// Subscriber bridges internal events to Streamer.bot DoAction commands
type Subscriber struct {
	client *Client
	bus    event.Bus
}

// NewSubscriber creates a new Streamer.bot event subscriber
func NewSubscriber(client *Client, bus event.Bus) *Subscriber {
	return &Subscriber{
		client: client,
		bus:    bus,
	}
}

// Subscribe registers handlers for relevant event types
func (s *Subscriber) Subscribe() {
	// Subscribe to job level up events
	s.bus.Subscribe(event.Type(domain.EventTypeJobLevelUp), s.handleJobLevelUp)

	// Subscribe to progression voting started events
	s.bus.Subscribe(event.ProgressionVotingStarted, s.handleVotingStarted)

	// Subscribe to progression cycle completed events
	s.bus.Subscribe(event.ProgressionCycleCompleted, s.handleCycleCompleted)

	// Subscribe to progression all unlocked events
	s.bus.Subscribe(event.ProgressionAllUnlocked, s.handleAllUnlocked)

	// Subscribe to gamble completed events
	s.bus.Subscribe(event.Type(domain.EventGambleCompleted), s.handleGambleCompleted)

	// Subscribe to slots completed events
	s.bus.Subscribe(event.Type(domain.EventSlotsCompleted), s.handleSlotsCompleted)

	// Subscribe to timeout events
	s.bus.Subscribe(event.TimeoutApplied, s.handleTimeoutUpdate)
	s.bus.Subscribe(event.TimeoutCleared, s.handleTimeoutUpdate)

	// Subscribe to subscription events
	s.bus.Subscribe(event.SubscriptionActivated, s.handleSubscriptionUpdate)
	s.bus.Subscribe(event.SubscriptionRenewed, s.handleSubscriptionUpdate)
	s.bus.Subscribe(event.SubscriptionUpgraded, s.handleSubscriptionUpdate)
	s.bus.Subscribe(event.SubscriptionDowngraded, s.handleSubscriptionUpdate)
	s.bus.Subscribe(event.SubscriptionExpired, s.handleSubscriptionUpdate)
	s.bus.Subscribe(event.SubscriptionCancelled, s.handleSubscriptionUpdate)

	// Subscribe to item used events
	s.bus.Subscribe(event.Type(domain.EventTypeItemUsed), s.handleItemUsed)

	slog.Info("Streamer.bot subscriber registered for event types",
		"types", []string{
			string(domain.EventTypeJobLevelUp),
			string(event.ProgressionVotingStarted),
			string(event.ProgressionCycleCompleted),
			string(event.ProgressionAllUnlocked),
			string(domain.EventGambleCompleted),
			string(domain.EventSlotsCompleted),
			string(event.TimeoutApplied),
			string(event.TimeoutCleared),
			string(event.SubscriptionActivated),
			string(event.SubscriptionRenewed),
			string(event.SubscriptionExpired),
			string(event.SubscriptionCancelled),
			string(domain.EventTypeItemUsed),
		})
}

// handleJobLevelUp sends a DoAction for job level up events
func (s *Subscriber) handleJobLevelUp(_ context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[event.JobLevelUpPayloadV1](evt.Payload)
	if err != nil {
		slog.Warn("Invalid job level up event payload type", "error", err)
		return nil
	}

	// Extract source from payload or metadata
	source := payload.Source
	if source == "" {
		if src, ok := evt.GetMetadataValue("source").(string); ok {
			source = src
		}
	}

	// Only send to Streamerbot for Twitch platform
	if payload.Platform != "" && payload.Platform != domain.PlatformTwitch {
		return nil
	}

	args := map[string]string{
		"username":  payload.Username,
		"job_key":   payload.JobKey,
		"new_level": fmt.Sprintf("%d", payload.NewLevel),
		"source":    source,
	}

	slog.Debug(LogMsgEventReceived, "event_type", domain.EventTypeJobLevelUp, "args", args)

	if err := s.client.DoAction(ActionJobLevelUp, args); err != nil {
		// Use Debug level - Streamer.bot being unavailable is expected
		slog.Debug("Failed to send job level up to Streamer.bot", "error", err)
	}

	return nil
}

// handleVotingStarted sends a DoAction for voting session start events
func (s *Subscriber) handleVotingStarted(_ context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[event.ProgressionVotingStartedPayloadV1](evt.Payload)
	if err != nil {
		slog.Warn("Invalid voting started event payload type", "error", err)
		return nil
	}

	args := map[string]string{
		"previous_unlock": payload.PreviousUnlock,
		"options_count":   fmt.Sprintf("%d", len(payload.Options)),
	}

	for i, opt := range payload.Options {
		idx := i + 1
		displayName := opt.DisplayName
		if displayName == "" {
			displayName = opt.NodeKey
		}
		args[fmt.Sprintf("option_%d", idx)] = displayName
		args[fmt.Sprintf("option_%d_key", idx)] = opt.NodeKey
		args[fmt.Sprintf("option_%d_description", idx)] = opt.Description
		args[fmt.Sprintf("option_%d_duration", idx)] = opt.UnlockDuration
	}

	slog.Debug(LogMsgEventReceived, "event_type", event.ProgressionVotingStarted, "args", args)

	if err := s.client.DoAction(ActionVotingStarted, args); err != nil {
		// Use Debug level - Streamer.bot being unavailable is expected
		slog.Debug("Failed to send voting started to Streamer.bot", "error", err)
	}

	return nil
}

// handleCycleCompleted sends a DoAction for progression cycle completion events
func (s *Subscriber) handleCycleCompleted(_ context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[event.ProgressionCycleCompletedPayloadV1](evt.Payload)
	if err != nil {
		slog.Warn("Invalid cycle completed event payload type", "error", err)
		return nil
	}

	args := map[string]string{
		"node_key":     payload.UnlockedNode.NodeKey,
		"display_name": payload.UnlockedNode.DisplayName,
		"description":  payload.UnlockedNode.Description,
	}

	slog.Debug(LogMsgEventReceived, "event_type", event.ProgressionCycleCompleted, "args", args)

	if err := s.client.DoAction(ActionCycleCompleted, args); err != nil {
		// Use Debug level - Streamer.bot being unavailable is expected
		slog.Debug("Failed to send cycle completed to Streamer.bot", "error", err)
	}

	return nil
}

// handleAllUnlocked sends a DoAction when all progression nodes are unlocked
func (s *Subscriber) handleAllUnlocked(_ context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[event.ProgressionAllUnlockedPayloadV1](evt.Payload)
	if err != nil {
		slog.Warn("Invalid all unlocked event payload type", "error", err)
		return nil
	}

	args := map[string]string{
		"message": payload.Message,
	}

	slog.Debug(LogMsgEventReceived, "event_type", event.ProgressionAllUnlocked, "args", args)

	if err := s.client.DoAction(ActionAllUnlocked, args); err != nil {
		// Use Debug level - Streamer.bot being unavailable is expected
		slog.Debug("Failed to send all unlocked to Streamer.bot", "error", err)
	}

	return nil
}

// handleGambleCompleted sends a DoAction when a gamble completes
func (s *Subscriber) handleGambleCompleted(_ context.Context, evt event.Event) error {
	var totalValue int64
	var participantCount int
	var winnerUsername string
	groupedItems := []string{}

	payloadMap, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid gamble completed event payload type")
		return nil
	}
	winnerUsername = getStringFromMap(payloadMap, "winner_username")
	totalValue = int64(getIntFromMap(payloadMap, "total_value"))
	participantCount = getIntFromMap(payloadMap, "participant_count")

	args := map[string]string{
		"winner_username":   winnerUsername,
		"total_value":       fmt.Sprintf("%d", totalValue),
		"participant_count": fmt.Sprintf("%d", participantCount),
		"items":             fmt.Sprintf("%v", groupedItems),
		"has_winner":        fmt.Sprintf("%t", winnerUsername != ""),
	}

	slog.Debug(LogMsgEventReceived, "event_type", domain.EventGambleCompleted, "args", args)

	if err := s.client.DoAction(ActionGambleCompleted, args); err != nil {
		slog.Debug("Failed to send gamble completed to Streamer.bot", "error", err)
	}

	return nil
}

// handleSlotsCompleted sends a DoAction when a slots spin completes
func (s *Subscriber) handleSlotsCompleted(_ context.Context, evt event.Event) error {
	payload, ok := evt.Payload.(domain.SlotsCompletedPayload)
	if !ok {
		// Fallback for untyped payload
		payloadMap, ok := evt.Payload.(map[string]interface{})
		if !ok {
			slog.Warn("Invalid slots completed event payload type")
			return nil
		}

		// Extract fields with type assertions
		payoutMultiplier := 0.0
		if pm, ok := payloadMap["payout_multiplier"].(float64); ok {
			payoutMultiplier = pm
		}

		payload = domain.SlotsCompletedPayload{
			UserID:           getStringFromMap(payloadMap, "user_id"),
			Username:         getStringFromMap(payloadMap, "username"),
			BetAmount:        getIntFromMap(payloadMap, "bet_amount"),
			Reel1:            getStringFromMap(payloadMap, "reel1"),
			Reel2:            getStringFromMap(payloadMap, "reel2"),
			Reel3:            getStringFromMap(payloadMap, "reel3"),
			PayoutAmount:     getIntFromMap(payloadMap, "payout_amount"),
			PayoutMultiplier: payoutMultiplier,
			TriggerType:      getStringFromMap(payloadMap, "trigger_type"),
			IsWin:            getBoolFromMap(payloadMap, "is_win"),
			IsNearMiss:       getBoolFromMap(payloadMap, "is_near_miss"),
		}
	}

	args := map[string]string{
		"user_id":           payload.UserID,
		"username":          payload.Username,
		"bet_amount":        fmt.Sprintf("%d", payload.BetAmount),
		"reel1":             payload.Reel1,
		"reel2":             payload.Reel2,
		"reel3":             payload.Reel3,
		"payout_amount":     fmt.Sprintf("%d", payload.PayoutAmount),
		"payout_multiplier": fmt.Sprintf("%.2f", payload.PayoutMultiplier),
		"trigger_type":      payload.TriggerType,
		"is_win":            fmt.Sprintf("%t", payload.IsWin),
		"is_near_miss":      fmt.Sprintf("%t", payload.IsNearMiss),
	}

	slog.Debug(LogMsgEventReceived, "event_type", domain.EventSlotsCompleted, "args", args)

	if err := s.client.DoAction(ActionSlotsResult, args); err != nil {
		slog.Debug("Failed to send slots result to Streamer.bot", "error", err)
	}

	return nil
}

// handleTimeoutUpdate sends a DoAction for timeout applied or cleared events
func (s *Subscriber) handleTimeoutUpdate(_ context.Context, evt event.Event) error {
	var payload event.TimeoutPayloadV1

	// Attempt to extract typed payload
	if p, ok := evt.Payload.(event.TimeoutPayloadV1); ok {
		payload = p
	} else {
		// Fallback to map parsing (useful if events are unmarshaled from JSON)
		pMap, ok := evt.Payload.(map[string]interface{})
		if !ok {
			slog.Warn("Invalid timeout event payload type", "type", fmt.Sprintf("%T", evt.Payload))
			return nil
		}
		payload = event.TimeoutPayloadV1{
			Platform:        getStringFromMap(pMap, "platform"),
			Username:        getStringFromMap(pMap, "username"),
			Action:          getStringFromMap(pMap, "action"),
			DurationSeconds: getIntFromMap(pMap, "duration_seconds"),
			Reason:          getStringFromMap(pMap, "reason"),
		}
	}

	args := map[string]string{
		"platform":         payload.Platform,
		"username":         payload.Username,
		"action":           payload.Action,
		"duration_seconds": fmt.Sprintf("%d", payload.DurationSeconds),
		"reason":           payload.Reason,
	}

	slog.Debug(LogMsgEventReceived, "event_type", evt.Type, "args", args)

	if err := s.client.DoAction(ActionTimeoutUpdate, args); err != nil {
		// Streamer.bot being offline is expected, use debug level
		slog.Debug("Failed to send timeout update to Streamer.bot", "error", err)
	}

	return nil
}

// handleSubscriptionUpdate sends a DoAction for subscription lifecycle events
func (s *Subscriber) handleSubscriptionUpdate(_ context.Context, evt event.Event) error {
	// Try typed payload first
	var payload event.SubscriptionPayloadV1
	if p, ok := evt.Payload.(event.SubscriptionPayloadV1); ok {
		payload = p
	} else {
		// Fall back to map parsing
		pMap, ok := evt.Payload.(map[string]interface{})
		if !ok {
			slog.Warn("Invalid subscription event payload type")
			return nil
		}
		payload = event.SubscriptionPayloadV1{
			UserID:    getStringFromMap(pMap, "user_id"),
			Platform:  getStringFromMap(pMap, "platform"),
			TierName:  getStringFromMap(pMap, "tier_name"),
			Timestamp: int64(getIntFromMap(pMap, "timestamp")),
		}
	}

	args := map[string]string{
		"user_id":    payload.UserID,
		"platform":   payload.Platform,
		"tier_name":  payload.TierName,
		"event_type": string(evt.Type),
		"timestamp":  fmt.Sprintf("%d", payload.Timestamp),
	}

	slog.Debug(LogMsgEventReceived, "event_type", evt.Type, "args", args)

	if err := s.client.DoAction(ActionSubscriptionUpdate, args); err != nil {
		// Streamer.bot being offline is expected, use debug level
		slog.Debug("Failed to send subscription update to Streamer.bot", "error", err)
	}

	return nil
}

// handleItemUsed sends a DoAction when an item is used
func (s *Subscriber) handleItemUsed(_ context.Context, evt event.Event) error {
	// Try typed payload first
	var payload domain.ItemUsedPayload
	if p, ok := evt.Payload.(domain.ItemUsedPayload); ok {
		payload = p
	} else {
		// Fall back to map parsing
		pMap, ok := evt.Payload.(map[string]interface{})
		if !ok {
			slog.Warn("Invalid item used event payload type")
			return nil
		}
		payload = domain.ItemUsedPayload{
			UserID:    getStringFromMap(pMap, "user_id"),
			ItemName:  getStringFromMap(pMap, "item_name"),
			Quantity:  getIntFromMap(pMap, "quantity"),
			Timestamp: int64(getIntFromMap(pMap, "timestamp")),
			Metadata:  pMap["metadata"],
		}
	}

	args := map[string]string{
		"user_id":   payload.UserID,
		"item_name": payload.ItemName,
		"quantity":  fmt.Sprintf("%d", payload.Quantity),
	}

	// Try extracting the "target" from the generic metadata field
	if mapMeta, ok := payload.Metadata.(map[string]interface{}); ok {
		if t, tgOk := mapMeta["target"].(string); tgOk && t != "" {
			args["target"] = t
		}
	}

	slog.Debug(LogMsgEventReceived, "event_type", evt.Type, "args", args)

	if err := s.client.DoAction(ActionItemUsed, args); err != nil {
		// Streamer.bot being offline is expected, use debug level
		slog.Debug("Failed to send item used to Streamer.bot", "error", err)
	}

	return nil
}

// Helper functions for type conversion

func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
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

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
