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
	s.bus.Subscribe(event.Type(domain.EventJobLevelUp), s.handleJobLevelUp)

	// Subscribe to progression voting started events
	s.bus.Subscribe(event.ProgressionVotingStarted, s.handleVotingStarted)

	// Subscribe to progression cycle completed events
	s.bus.Subscribe(event.ProgressionCycleCompleted, s.handleCycleCompleted)

	// Subscribe to progression all unlocked events
	s.bus.Subscribe(event.ProgressionAllUnlocked, s.handleAllUnlocked)

	// Subscribe to gamble completed events
	s.bus.Subscribe(event.Type(domain.EventGambleCompleted), s.handleGambleCompleted)

	slog.Info("Streamer.bot subscriber registered for event types",
		"types", []string{
			string(domain.EventJobLevelUp),
			string(event.ProgressionVotingStarted),
			string(event.ProgressionCycleCompleted),
			string(event.ProgressionAllUnlocked),
			string(domain.EventGambleCompleted),
		})
}

// handleJobLevelUp sends a DoAction for job level up events
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

	args := map[string]string{
		"user_id":   getStringFromMap(payload, "user_id"),
		"job_key":   getStringFromMap(payload, "job_key"),
		"old_level": fmt.Sprintf("%d", getIntFromMap(payload, "old_level")),
		"new_level": fmt.Sprintf("%d", getIntFromMap(payload, "new_level")),
		"source":    source,
	}

	slog.Debug(LogMsgEventReceived, "event_type", domain.EventJobLevelUp, "args", args)

	if err := s.client.DoAction(ActionJobLevelUp, args); err != nil {
		// Use Debug level - Streamer.bot being unavailable is expected
		slog.Debug("Failed to send job level up to Streamer.bot", "error", err)
	}

	return nil
}

// handleVotingStarted sends a DoAction for voting session start events
func (s *Subscriber) handleVotingStarted(_ context.Context, evt event.Event) error {
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid voting started event payload type")
		return nil
	}

	args := map[string]string{
		"previous_unlock": getStringFromMap(payload, "previous_unlock"),
	}

	// Extract options - handle both []map[string]interface{} and []interface{}
	var optsRaw interface{}
	if optsRaw, ok = payload["options"]; !ok {
		slog.Warn("Voting started event missing options field", "sessionID", evt.Metadata["session_id"])
	}

	extractedOptions := false
	if opts, ok := optsRaw.([]map[string]interface{}); ok {
		args["options_count"] = fmt.Sprintf("%d", len(opts))
		for i, optMap := range opts {
			s.populateOptionArgs(args, i+1, optMap)
		}
		extractedOptions = true
	} else if opts, ok := optsRaw.([]interface{}); ok {
		args["options_count"] = fmt.Sprintf("%d", len(opts))
		for i, opt := range opts {
			if optMap, ok := opt.(map[string]interface{}); ok {
				s.populateOptionArgs(args, i+1, optMap)
			} else {
				slog.Warn("Individual option is not a map", "index", i, "type", fmt.Sprintf("%T", opt))
			}
		}
		extractedOptions = true
	} else if optsRaw != nil {
		slog.Warn("Invalid options type in voting started event", "type", fmt.Sprintf("%T", optsRaw))
	}

	if !extractedOptions {
		slog.Warn("Failed to extract options for Streamer.bot", "payload_keys", getMapKeys(payload))
	}

	slog.Debug(LogMsgEventReceived, "event_type", event.ProgressionVotingStarted, "args", args)

	if err := s.client.DoAction(ActionVotingStarted, args); err != nil {
		// Use Debug level - Streamer.bot being unavailable is expected
		slog.Debug("Failed to send voting started to Streamer.bot", "error", err)
	}

	return nil
}

func (s *Subscriber) populateOptionArgs(args map[string]string, index int, optMap map[string]interface{}) {
	displayName := getStringFromMap(optMap, "display_name")
	if displayName == "" {
		displayName = getStringFromMap(optMap, "node_key")
	}
	args[fmt.Sprintf("option_%d", index)] = displayName
	args[fmt.Sprintf("option_%d_key", index)] = getStringFromMap(optMap, "node_key")
	args[fmt.Sprintf("option_%d_description", index)] = getStringFromMap(optMap, "description")
	args[fmt.Sprintf("option_%d_duration", index)] = getStringFromMap(optMap, "unlock_duration")
}

// handleCycleCompleted sends a DoAction for progression cycle completion events
func (s *Subscriber) handleCycleCompleted(_ context.Context, evt event.Event) error {
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid cycle completed event payload type")
		return nil
	}

	args := map[string]string{}

	// Extract unlocked node info
	if unlockedNode := payload["unlocked_node"]; unlockedNode != nil {
		if node, ok := unlockedNode.(map[string]interface{}); ok {
			args["node_key"] = getStringFromMap(node, "node_key")
			args["display_name"] = getStringFromMap(node, "display_name")
			args["description"] = getStringFromMap(node, "description")
		}
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
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid all unlocked event payload type")
		return nil
	}

	args := map[string]string{
		"message": getStringFromMap(payload, "message"),
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
		}
	}

	args := map[string]string{
		"gamble_id":         payload.GambleID,
		"winner_id":         payload.WinnerID,
		"total_value":       fmt.Sprintf("%d", payload.TotalValue),
		"participant_count": fmt.Sprintf("%d", payload.ParticipantCount),
		"has_winner":        fmt.Sprintf("%t", payload.WinnerID != ""),
	}

	slog.Debug(LogMsgEventReceived, "event_type", domain.EventGambleCompleted, "args", args)

	if err := s.client.DoAction(ActionGambleCompleted, args); err != nil {
		slog.Debug("Failed to send gamble completed to Streamer.bot", "error", err)
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

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
