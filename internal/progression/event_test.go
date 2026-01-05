package progression

import (
	"context"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBus is a mock implementation of the event.Bus interface
type MockBus struct {
	mock.Mock
}

func (m *MockBus) Publish(ctx context.Context, evt event.Event) error {
	args := m.Called(ctx, evt)
	return args.Error(0)
}

func (m *MockBus) Subscribe(topic event.Type, handler event.Handler) {
	m.Called(topic, handler)
}

func TestStartVotingSession_AutoSelect_PublishesEvent(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()
	mockBus := new(MockBus)

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
	repo.nodes[1] = root
	repo.nodesByKey["root_node"] = root
	repo.UnlockNode(ctx, 1, 1, "system", 0)

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
	repo.nodes[2] = child
	repo.nodesByKey["child_node"] = child

	// Initialize service with mock bus
	service := NewService(repo, mockBus)

	// Expect Publish to be called with specific event
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

	// Act
	err := service.StartVotingSession(ctx, nil)

	// Assert
	assert.NoError(t, err)

	// Verify bus call
	mockBus.AssertExpectations(t)

	// Verify no session created (core logic check)
	session, err := repo.GetActiveSession(ctx)
	assert.NoError(t, err)
	assert.Nil(t, session)
}
