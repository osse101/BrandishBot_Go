package progression

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

// MockBus is a mock implementation of the event.Bus interface
type MockBus struct {
	mock.Mock
}

func (m *MockBus) Publish(ctx context.Context, evt event.Event) error {
	args := m.Called(ctx, evt)
	return args.Error(0)
}

func (m *MockBus) Subscribe(eventType event.Type, handler event.Handler) {
	m.Called(eventType, handler)
	// No-op for tests - just allow the subscription
}

func TestStartVotingSession_AutoSelect_PublishesEvent(t *testing.T) {
	mockRepo := NewMockRepository()
	ctx := context.Background()
	mockBus := new(MockBus)

	// Service subscribes to events for cache invalidation
	mockBus.On("Subscribe", event.Type("progression.node_unlocked"), mock.Anything).Return()
	mockBus.On("Subscribe", event.Type("progression.node_relocked"), mock.Anything).Return()

	// Setup tree with single available option
	// 1. Root - Unlocked
	root := &domain.ProgressionNode{
		ID:          1,
		NodeKey:     "root_node",
		NodeType:    "feature",
		DisplayName: "Root Node",
		MaxLevel:    1,
		UnlockCost:  0,
		CreatedAt:   time.Now(),
	}
	mockRepo.nodes[1] = root
	mockRepo.nodesByKey["root_node"] = root
	mockRepo.UnlockNode(ctx, 1, 1, "system", 0)

	// 2. Child - Locked (Single Option)
	child := &domain.ProgressionNode{
		ID:          2,
		NodeKey:     "child_node",
		NodeType:    "feature",
		DisplayName: "Child Node",
		MaxLevel:    1,
		UnlockCost:  100,
		CreatedAt:   time.Now(),
	}
	mockRepo.nodes[2] = child
	mockRepo.nodesByKey["child_node"] = child

	// Initialize service with mock bus
	service := NewService(mockRepo, NewMockUser(), mockBus)

	// Expect Publish to be called with ProgressionTargetSet event
	mockBus.On("Publish", ctx, mock.MatchedBy(func(evt event.Event) bool {
		if evt.Type != event.ProgressionTargetSet {
			return false
		}

		payload, ok := evt.Payload.(map[string]interface{})
		if !ok {
			return false
		}

		// Verify payload content
		if payload["node_key"] != "child_node" {
			return false
		}
		if payload["target_level"] != 1 {
			return false
		}
		if payload["auto_selected"] != true {
			return false
		}

		return true
	})).Return(nil)

	mockBus.On("Publish", ctx, mock.MatchedBy(func(evt event.Event) bool {
		return evt.Type == event.ProgressionVotingStarted
	})).Return(nil)

	// Act
	err := service.StartVotingSession(ctx, nil)

	// Assert
	assert.NoError(t, err)

	// Verify bus calls
	mockBus.AssertExpectations(t)

	session, err := mockRepo.GetActiveSession(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, session, "Session should exist in 'voting' status for auto-select")
}
