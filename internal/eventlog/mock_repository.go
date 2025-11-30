package eventlog

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) LogEvent(ctx context.Context, eventType string, userID *string, payload, metadata map[string]interface{}) error {
	args := m.Called(ctx, eventType, userID, payload, metadata)
	return args.Error(0)
}

func (m *MockRepository) GetEvents(ctx context.Context, filter EventFilter) ([]Event, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]Event), args.Error(1)
}

func (m *MockRepository) GetEventsByUser(ctx context.Context, userID string, limit int) ([]Event, error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]Event), args.Error(1)
}

func (m *MockRepository) GetEventsByType(ctx context.Context, eventType string, limit int) ([]Event, error) {
	args := m.Called(ctx, eventType, limit)
	return args.Get(0).([]Event), args.Error(1)
}

func (m *MockRepository) CleanupOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	args := m.Called(ctx, retentionDays)
	return args.Get(0).(int64), args.Error(1)
}
