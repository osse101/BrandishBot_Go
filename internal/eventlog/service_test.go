package eventlog_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/eventlog"
	"github.com/osse101/BrandishBot_Go/internal/eventlog/mocks"
)

// MockEventBus is a mock implementation of event.Bus
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(ctx context.Context, evt event.Event) error {
	args := m.Called(ctx, evt)
	return args.Error(0)
}

func (m *MockEventBus) Subscribe(eventType event.Type, handler event.Handler) {
	m.Called(eventType, handler)
}

func TestService_Subscribe(t *testing.T) {
	mockRepo := mocks.NewMockRepository(t)
	service := eventlog.NewService(mockRepo)
	mockBus := new(MockEventBus)

	// Expect subscription to all event types
	eventTypes := []event.Type{
		"item.sold",
		"item.bought",
		"item.upgraded",
		"item.disassembled",
		"item.used",
		"search.performed",
		"engagement",
	}

	for _, et := range eventTypes {
		mockBus.On("Subscribe", et, mock.Anything).Return()
	}

	err := service.Subscribe(mockBus)
	assert.NoError(t, err)
	mockBus.AssertExpectations(t)
}

func TestService_HandleEvent(t *testing.T) {
	mockRepo := mocks.NewMockRepository(t)
	service := eventlog.NewService(mockRepo)

	// Use test hooks to access private method
	hooks := eventlog.NewTestHooks(service)

	ctx := context.Background()
	userID := "user123"
	payload := map[string]interface{}{
		"user_id":   userID,
		"item_name": "sword",
	}
	evt := event.Event{
		Type:    "item.sold",
		Payload: payload,
	}

	// Expect LogEvent to be called
	mockRepo.On("LogEvent", ctx, "item.sold", &userID, payload, mock.Anything).Return(nil)

	err := hooks.HandleEvent(ctx, evt)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_CleanupOldEvents(t *testing.T) {
	mockRepo := mocks.NewMockRepository(t)
	service := eventlog.NewService(mockRepo)
	ctx := context.Background()

	mockRepo.On("CleanupOldEvents", ctx, 10).Return(int64(5), nil)

	count, err := service.CleanupOldEvents(ctx, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
	mockRepo.AssertExpectations(t)
}
