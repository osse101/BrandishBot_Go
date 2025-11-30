package eventlog

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
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
	mockRepo := new(MockRepository)
	// We need to access the private handleEvent method, but since we are in the same package (eventlog),
	// we can test it directly if we export it or use the service instance.
	// However, handleEvent is private.
	// But we can test it by simulating the handler call if we could capture it.
	// Alternatively, we can export it for testing or just test via public methods if possible.
	// Since Subscribe registers the handler, we can't easily trigger it unless we mock the Bus to capture the handler.

	// Let's use the service instance and call the handler directly via reflection or by changing it to public?
	// Or better: make handleEvent public or internal (it is internal to package).
	// Since this test is in `package eventlog`, we can access `handleEvent`.

	svc := NewService(mockRepo).(*service)

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

	err := svc.handleEvent(ctx, evt)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_CleanupOldEvents(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	mockRepo.On("CleanupOldEvents", ctx, 10).Return(int64(5), nil)

	count, err := service.CleanupOldEvents(ctx, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
	mockRepo.AssertExpectations(t)
}
