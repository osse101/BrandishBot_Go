package eventlog_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
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
	t.Parallel()

	mockRepo := mocks.NewMockRepository(t)
	service := eventlog.NewService(mockRepo)
	mockBus := new(MockEventBus)

	// Expect subscription to all event types
	eventTypes := []event.Type{
		domain.EventTypeItemSold,
		domain.EventTypeItemBought,
		domain.EventTypeItemUpgraded,
		domain.EventTypeItemDisassembled,
		domain.EventTypeItemUsed,
		domain.EventTypeSearchPerformed,
		domain.EventTypeEngagement,
	}

	for _, et := range eventTypes {
		mockBus.On("Subscribe", et, mock.Anything).Return()
	}

	err := service.Subscribe(mockBus)
	assert.NoError(t, err)
	mockBus.AssertExpectations(t)
}

func TestService_HandleEvent(t *testing.T) {
	t.Parallel()

	userID := "user123"

	tests := []struct {
		name        string
		event       event.Event
		mockSetup   func(repo *mocks.MockRepository)
		expectedErr error
	}{
		{
			name: "Success - Event Logged With User ID",
			event: event.Event{
				Type: domain.EventTypeItemSold,
				Payload: map[string]interface{}{
					eventlog.PayloadKeyUserID: userID,
					"item_name":               "sword",
				},
				Metadata: map[string]interface{}{"source": "api"},
			},
			mockSetup: func(repo *mocks.MockRepository) {
				repo.On("LogEvent", mock.Anything, string(domain.EventTypeItemSold), &userID, map[string]interface{}{
					eventlog.PayloadKeyUserID: userID,
					"item_name":               "sword",
				}, map[string]interface{}{"source": "api"}).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "Success - Event Logged Without User ID",
			event: event.Event{
				Type: domain.EventTypeItemSold,
				Payload: map[string]interface{}{
					"item_name": "sword",
				},
			},
			mockSetup: func(repo *mocks.MockRepository) {
				repo.On("LogEvent", mock.Anything, string(domain.EventTypeItemSold), (*string)(nil), map[string]interface{}{
					"item_name": "sword",
				}, mock.Anything).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "Success - Event Logged With Non-Map Payload",
			event: event.Event{
				Type:    domain.EventTypeItemSold,
				Payload: "just a string payload",
			},
			mockSetup: func(repo *mocks.MockRepository) {
				repo.On("LogEvent", mock.Anything, string(domain.EventTypeItemSold), (*string)(nil), "just a string payload", mock.Anything).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "Failure - Repo Returns Error",
			event: event.Event{
				Type: domain.EventTypeItemSold,
				Payload: map[string]interface{}{
					eventlog.PayloadKeyUserID: userID,
					"item_name":               "sword",
				},
			},
			mockSetup: func(repo *mocks.MockRepository) {
				repo.On("LogEvent", mock.Anything, string(domain.EventTypeItemSold), &userID, map[string]interface{}{
					eventlog.PayloadKeyUserID: userID,
					"item_name":               "sword",
				}, mock.Anything).Return(errors.New("db error"))
			},
			expectedErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := mocks.NewMockRepository(t)
			tt.mockSetup(mockRepo)

			service := eventlog.NewService(mockRepo)
			hooks := eventlog.NewTestHooks(service)

			err := hooks.HandleEvent(context.Background(), tt.event)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_CleanupOldEvents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		retentionDays int
		mockSetup     func(repo *mocks.MockRepository)
		expectedCount int64
		expectedErr   error
	}{
		{
			name:          "Success - Events Cleaned Up",
			retentionDays: 10,
			mockSetup: func(repo *mocks.MockRepository) {
				repo.On("CleanupOldEvents", mock.Anything, 10).Return(int64(5), nil)
			},
			expectedCount: 5,
			expectedErr:   nil,
		},
		{
			name:          "Failure - Repo Returns Error",
			retentionDays: 10,
			mockSetup: func(repo *mocks.MockRepository) {
				repo.On("CleanupOldEvents", mock.Anything, 10).Return(int64(0), errors.New("db error"))
			},
			expectedCount: 0,
			expectedErr:   errors.New("db error"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := mocks.NewMockRepository(t)
			tt.mockSetup(mockRepo)

			service := eventlog.NewService(mockRepo)

			count, err := service.CleanupOldEvents(context.Background(), tt.retentionDays)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, count)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_GetEvents(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testUserID := "user123"
	testEventType := string(domain.EventTypeItemSold)

	tests := []struct {
		name           string
		filter         eventlog.EventFilter
		mockSetup      func(repo *mocks.MockRepository)
		expectedEvents []eventlog.Event
		expectedErr    error
	}{
		{
			name: "Success - Retrieve Events",
			filter: eventlog.EventFilter{
				UserID:    &testUserID,
				EventType: &testEventType,
				Limit:     10,
			},
			mockSetup: func(repo *mocks.MockRepository) {
				repo.On("GetEvents", mock.Anything, eventlog.EventFilter{
					UserID:    &testUserID,
					EventType: &testEventType,
					Limit:     10,
				}).Return([]eventlog.Event{
					{
						ID:        1,
						EventType: string(domain.EventTypeItemSold),
						UserID:    &testUserID,
						Payload:   map[string]interface{}{"item": "sword"},
						CreatedAt: now,
					},
				}, nil)
			},
			expectedEvents: []eventlog.Event{
				{
					ID:        1,
					EventType: string(domain.EventTypeItemSold),
					UserID:    &testUserID,
					Payload:   map[string]interface{}{"item": "sword"},
					CreatedAt: now,
				},
			},
			expectedErr: nil,
		},
		{
			name: "Failure - Repo Returns Error",
			filter: eventlog.EventFilter{
				Limit: 10,
			},
			mockSetup: func(repo *mocks.MockRepository) {
				repo.On("GetEvents", mock.Anything, eventlog.EventFilter{
					Limit: 10,
				}).Return(nil, errors.New("db error"))
			},
			expectedEvents: nil,
			expectedErr:    errors.New("db error"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := mocks.NewMockRepository(t)
			tt.mockSetup(mockRepo)

			service := eventlog.NewService(mockRepo)

			events, err := service.GetEvents(context.Background(), tt.filter)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedEvents, events)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
