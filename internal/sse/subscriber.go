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

	slog.Info("SSE subscriber registered for event types",
		"types", []string{
			string(domain.EventJobLevelUp),
			string(event.ProgressionCycleCompleted),
			string(event.ProgressionVotingStarted),
			string(event.ProgressionTargetSet),
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
		UnlockedNode:  extractNodeInfo(payload["unlocked_node"]),
		VotingSession: extractSessionInfo(payload["voting_session"]),
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
		SessionID:      getIntFromMap(payload, "session_id"),
		PreviousUnlock: getStringFromMap(payload, "previous_unlock"),
		AutoSelected:   false,
	}

	// Extract options from payload
	if opts, ok := payload["options"].([]map[string]interface{}); ok {
		ssePayload.Options = make([]VotingOptionInfo, 0, len(opts))
		for _, opt := range opts {
			ssePayload.Options = append(ssePayload.Options, VotingOptionInfo{
				NodeKey:     getStringFromMap(opt, "node_key"),
				DisplayName: getStringFromMap(opt, "display_name"),
			})
		}
	} else if opts, ok := payload["options"].([]interface{}); ok {
		ssePayload.Options = make([]VotingOptionInfo, 0, len(opts))
		for _, opt := range opts {
			if optMap, ok := opt.(map[string]interface{}); ok {
				ssePayload.Options = append(ssePayload.Options, VotingOptionInfo{
					NodeKey:     getStringFromMap(optMap, "node_key"),
					DisplayName: getStringFromMap(optMap, "display_name"),
				})
			}
		}
	}

	s.hub.Broadcast(EventTypeVotingStarted, ssePayload)

	slog.Debug(LogMsgEventBroadcast,
		"event_type", EventTypeVotingStarted,
		"session_id", ssePayload.SessionID,
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
			SessionID:      0, // No session when auto-selected
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
		}
	case domain.ProgressionNode:
		return NodeInfo{
			NodeKey:     node.NodeKey,
			DisplayName: node.DisplayName,
		}
	case map[string]interface{}:
		return NodeInfo{
			NodeKey:     getStringFromMap(node, "node_key"),
			DisplayName: getStringFromMap(node, "display_name"),
		}
	}

	return NodeInfo{}
}

func extractSessionInfo(v interface{}) *VotingSessionInfo {
	if v == nil {
		return nil
	}

	switch session := v.(type) {
	case *domain.ProgressionVotingSession:
		info := &VotingSessionInfo{
			SessionID: session.ID,
			Options:   make([]VotingOptionInfo, 0, len(session.Options)),
		}
		for _, opt := range session.Options {
			optInfo := VotingOptionInfo{
				NodeKey: "",
			}
			if opt.NodeDetails != nil {
				optInfo.NodeKey = opt.NodeDetails.NodeKey
				optInfo.DisplayName = opt.NodeDetails.DisplayName
			}
			info.Options = append(info.Options, optInfo)
		}
		return info
	case map[string]interface{}:
		info := &VotingSessionInfo{
			SessionID: getIntFromMap(session, "id"),
		}
		if opts, ok := session["options"].([]interface{}); ok {
			info.Options = make([]VotingOptionInfo, 0, len(opts))
			for _, opt := range opts {
				if optMap, ok := opt.(map[string]interface{}); ok {
					info.Options = append(info.Options, VotingOptionInfo{
						NodeKey:     getStringFromMap(optMap, "node_key"),
						DisplayName: getStringFromMap(optMap, "display_name"),
					})
				}
			}
		}
		return info
	}

	return nil
}
