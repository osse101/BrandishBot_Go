package streamerbot

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

// mockEventBus implements event.Bus for testing
type mockEventBus struct {
	subscribedTypes []event.Type
	handlers        map[event.Type]event.Handler
}

func newMockEventBus() *mockEventBus {
	return &mockEventBus{
		subscribedTypes: []event.Type{},
		handlers:        make(map[event.Type]event.Handler),
	}
}

func (m *mockEventBus) Subscribe(t event.Type, h event.Handler) {
	m.subscribedTypes = append(m.subscribedTypes, t)
	m.handlers[t] = h
}

func (m *mockEventBus) Publish(ctx context.Context, e event.Event) error {
	return nil
}

func (m *mockEventBus) PublishBatch(ctx context.Context, events []event.Event) error {
	return nil
}

func (m *mockEventBus) HasSubscribers(t event.Type) bool {
	_, ok := m.handlers[t]
	return ok
}

// TestSubscriber_NewSubscriber verifies correct initialization
func TestSubscriber_NewSubscriber(t *testing.T) {
	client := NewClient("", "")
	bus := newMockEventBus()
	sub := NewSubscriber(client, bus)

	assert.NotNil(t, sub)
	assert.Equal(t, client, sub.client)
	assert.Equal(t, bus, sub.bus)
}

// TestSubscriber_Subscribe verifies all expected events are subscribed to
func TestSubscriber_Subscribe(t *testing.T) {
	client := NewClient("", "")
	bus := newMockEventBus()
	sub := NewSubscriber(client, bus)

	sub.Subscribe()

	expectedSubscriptions := []event.Type{
		event.Type(domain.EventTypeJobLevelUp),
		event.ProgressionVotingStarted,
		event.ProgressionCycleCompleted,
		event.ProgressionAllUnlocked,
		event.Type(domain.EventGambleCompleted),
		event.Type(domain.EventSlotsCompleted),
		event.TimeoutApplied,
		event.TimeoutCleared,
		event.SubscriptionActivated,
		event.SubscriptionRenewed,
		event.SubscriptionUpgraded,
		event.SubscriptionDowngraded,
		event.SubscriptionExpired,
		event.SubscriptionCancelled,
		event.Type(domain.EventTypeItemUsed),
	}

	assert.ElementsMatch(t, expectedSubscriptions, bus.subscribedTypes)
}

// TestSubscriber_Handlers_InvalidPayloads verifies handlers don't crash on invalid data
func TestSubscriber_Handlers_InvalidPayloads(t *testing.T) {
	client := NewClient("", "")
	bus := newMockEventBus()
	sub := NewSubscriber(client, bus)
	ctx := context.Background()

	invalidPayloads := []interface{}{
		nil,
		"string",
		123,
		[]int{1, 2, 3},
	}

	handlersToTest := []struct {
		name    string
		handler func(context.Context, event.Event) error
	}{
		{"handleJobLevelUp", sub.handleJobLevelUp},
		{"handleVotingStarted", sub.handleVotingStarted},
		{"handleCycleCompleted", sub.handleCycleCompleted},
		{"handleAllUnlocked", sub.handleAllUnlocked},
		{"handleGambleCompleted", sub.handleGambleCompleted},
		{"handleSlotsCompleted", sub.handleSlotsCompleted},
		{"handleTimeoutUpdate", sub.handleTimeoutUpdate},
		{"handleSubscriptionUpdate", sub.handleSubscriptionUpdate},
		{"handleItemUsed", sub.handleItemUsed},
	}

	for _, h := range handlersToTest {
		t.Run(h.name, func(t *testing.T) {
			for _, payload := range invalidPayloads {
				evt := event.Event{Payload: payload}
				err := h.handler(ctx, evt)
				assert.NoError(t, err, "Handler %s should handle invalid payload gracefully", h.name)
			}
		})
	}
}

// TestSubscriber_handleJobLevelUp verifies extracting payload
func TestSubscriber_handleJobLevelUp_Valid(t *testing.T) {
	client := NewClient("", "")
	bus := newMockEventBus()
	sub := NewSubscriber(client, bus)
	ctx := context.Background()

	// Only Twitch platform should cause further action, but since we have no mock client DoAction
	// We just ensure it doesn't panic when encountering valid data

	// Type 1: event.JobLevelUpPayloadV1
	evt := event.Event{
		Payload: event.JobLevelUpPayloadV1{
			UserID:   "user123",
			Platform: "twitch",
			JobKey:   "Miner",
			NewLevel: 2,
			OldLevel: 1,
			Source:   "test",
		},
	}
	err := sub.handleJobLevelUp(ctx, evt)
	assert.NoError(t, err) // DoAction fails but logs and returns nil

	// Valid payload but wrong platform (should return nil silently)
	evt2 := event.Event{
		Payload: event.JobLevelUpPayloadV1{
			UserID:   "user123",
			Platform: "discord",
			JobKey:   "Miner",
			NewLevel: 2,
			OldLevel: 1,
			Source:   "test",
		},
	}
	err2 := sub.handleJobLevelUp(ctx, evt2)
	assert.NoError(t, err2)
}

// TestSubscriber_handleTimeoutUpdate verifies extracting map/typed payloads
func TestSubscriber_handleTimeoutUpdate_Valid(t *testing.T) {
	client := NewClient("", "")
	bus := newMockEventBus()
	sub := NewSubscriber(client, bus)
	ctx := context.Background()

	// Type 1: Typed Payload
	evt := event.Event{
		Payload: event.TimeoutPayloadV1{
			Platform:        "twitch",
			Username:        "testuser",
			DurationSeconds: 300,
			Reason:          "spam",
			Action:          "applied",
			Timestamp:       time.Now().Unix(),
		},
	}
	err := sub.handleTimeoutUpdate(ctx, evt)
	assert.NoError(t, err) // Client not connected

	// Type 2: Map Payload
	evtMap := event.Event{
		Payload: map[string]interface{}{
			"platform":         "twitch",
			"username":         "testuser",
			"duration_seconds": float64(300),
			"reason":           "spam",
			"action":           "applied",
			"timestamp":        float64(time.Now().Unix()),
		},
	}
	errMap := sub.handleTimeoutUpdate(ctx, evtMap)
	assert.NoError(t, errMap) // Client not connected
}

func TestSubscriber_handleItemUsed_Valid(t *testing.T) {
	client := NewClient("", "")
	bus := newMockEventBus()
	sub := NewSubscriber(client, bus)
	ctx := context.Background()

	// Type 1: Typed Payload
	evt := event.Event{
		Payload: domain.ItemUsedPayload{
			UserID:    "user1",
			ItemName:  "Grenade",
			Quantity:  1,
			Timestamp: time.Now().Unix(),
		},
	}
	err := sub.handleItemUsed(ctx, evt)
	assert.NoError(t, err)

	// Type 2: Map Payload
	evtMap := event.Event{
		Payload: map[string]interface{}{
			"user_id":   "user1",
			"item_name": "Grenade",
			"quantity":  float64(1),
		},
	}
	errMap := sub.handleItemUsed(ctx, evtMap)
	assert.NoError(t, errMap)
}

func TestSubscriber_handleGambleCompleted_Valid(t *testing.T) {
	client := NewClient("", "")
	bus := newMockEventBus()
	sub := NewSubscriber(client, bus)
	ctx := context.Background()

	// Type 1: domain.GambleCompletedPayloadV2
	evt1 := event.Event{
		Payload: domain.GambleCompletedPayloadV2{
			GambleID:         "g1",
			WinnerID:         "w1",
			TotalValue:       1000,
			ParticipantCount: 5,
		},
	}
	err1 := sub.handleGambleCompleted(ctx, evt1)
	assert.NoError(t, err1)

	// Type 2: map
	evt2 := event.Event{
		Payload: map[string]interface{}{
			"gamble_id":         "g1",
			"winner_id":         "w1",
			"total_value":       float64(1000),
			"participant_count": float64(5),
		},
	}
	err2 := sub.handleGambleCompleted(ctx, evt2)
	assert.NoError(t, err2)
}

func TestSubscriber_handleSlotsCompleted_Valid(t *testing.T) {
	client := NewClient("", "")
	bus := newMockEventBus()
	sub := NewSubscriber(client, bus)
	ctx := context.Background()

	// Type 1: Typed Payload
	evt1 := event.Event{
		Payload: domain.SlotsCompletedPayload{
			UserID:           "u1",
			Username:         "tester",
			BetAmount:        100,
			PayoutAmount:     200,
			PayoutMultiplier: 2.0,
			Reel1:            "CHERRY",
			Reel2:            "CHERRY",
			Reel3:            "CHERRY",
		},
	}
	err1 := sub.handleSlotsCompleted(ctx, evt1)
	assert.NoError(t, err1)

	// Type 2: Map Payload
	evt2 := event.Event{
		Payload: map[string]interface{}{
			"user_id":           "u1",
			"username":          "tester",
			"bet_amount":        float64(100),
			"payout_amount":     float64(200),
			"payout_multiplier": 2.0,
			"reel1":             "CHERRY",
			"reel2":             "CHERRY",
			"reel3":             "CHERRY",
		},
	}
	err2 := sub.handleSlotsCompleted(ctx, evt2)
	assert.NoError(t, err2)
}
