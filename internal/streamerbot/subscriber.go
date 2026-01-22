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

	slog.Info("Streamer.bot subscriber registered for event types",
		"types", []string{
			string(domain.EventJobLevelUp),
			string(event.ProgressionVotingStarted),
			string(event.ProgressionCycleCompleted),
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
		"session_id":      fmt.Sprintf("%d", getIntFromMap(payload, "session_id")),
		"previous_unlock": getStringFromMap(payload, "previous_unlock"),
	}

	// Extract options
	if opts, ok := payload["options"].([]interface{}); ok {
		args["options_count"] = fmt.Sprintf("%d", len(opts))
		for i, opt := range opts {
			if optMap, ok := opt.(map[string]interface{}); ok {
				displayName := getStringFromMap(optMap, "display_name")
				if displayName == "" {
					displayName = getStringFromMap(optMap, "node_key")
				}
				args[fmt.Sprintf("option_%d", i+1)] = displayName
				args[fmt.Sprintf("option_%d_key", i+1)] = getStringFromMap(optMap, "node_key")
			}
		}
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
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		slog.Warn("Invalid cycle completed event payload type")
		return nil
	}

	args := map[string]string{}

	// Extract unlocked node info
	if unlockedNode := payload["unlocked_node"]; unlockedNode != nil {
		switch node := unlockedNode.(type) {
		case *domain.ProgressionNode:
			args["unlocked_node_key"] = node.NodeKey
			args["unlocked_node_name"] = node.DisplayName
		case domain.ProgressionNode:
			args["unlocked_node_key"] = node.NodeKey
			args["unlocked_node_name"] = node.DisplayName
		case map[string]interface{}:
			args["unlocked_node_key"] = getStringFromMap(node, "node_key")
			args["unlocked_node_name"] = getStringFromMap(node, "display_name")
		}
	}

	// Extract voting session info
	if votingSession := payload["voting_session"]; votingSession != nil {
		switch session := votingSession.(type) {
		case *domain.ProgressionVotingSession:
			args["new_session_id"] = fmt.Sprintf("%d", session.ID)
			args["options_count"] = fmt.Sprintf("%d", len(session.Options))
		case map[string]interface{}:
			args["new_session_id"] = fmt.Sprintf("%d", getIntFromMap(session, "id"))
			if opts, ok := session["options"].([]interface{}); ok {
				args["options_count"] = fmt.Sprintf("%d", len(opts))
			}
		}
	}

	slog.Debug(LogMsgEventReceived, "event_type", event.ProgressionCycleCompleted, "args", args)

	if err := s.client.DoAction(ActionCycleCompleted, args); err != nil {
		// Use Debug level - Streamer.bot being unavailable is expected
		slog.Debug("Failed to send cycle completed to Streamer.bot", "error", err)
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
