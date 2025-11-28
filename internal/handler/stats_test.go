package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStatsService mocks the stats.Service interface
type MockStatsService struct {
	mock.Mock
}

func (m *MockStatsService) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error {
	args := m.Called(ctx, userID, eventType, metadata)
	return args.Error(0)
}

func (m *MockStatsService) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) {
	args := m.Called(ctx, userID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StatsSummary), args.Error(1)
}

func (m *MockStatsService) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	args := m.Called(ctx, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StatsSummary), args.Error(1)
}

func (m *MockStatsService) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	args := m.Called(ctx, eventType, period, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.LeaderboardEntry), args.Error(1)
}

func TestHandleRecordEvent(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockStatsService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: RecordEventRequest{
				UserID:    "testuser",
				EventType: "item_used",
				EventData: map[string]interface{}{"item": "potion"},
			},
			setupMock: func(m *MockStatsService) {
				m.On("RecordUserEvent", mock.Anything, "testuser", domain.EventType("item_used"), mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Event recorded successfully",
		},
		{
			name: "Invalid Request - Missing UserID",
			requestBody: RecordEventRequest{
				EventType: "item_used",
			},
			setupMock:      func(m *MockStatsService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Service Error",
			requestBody: RecordEventRequest{
				UserID:    "testuser",
				EventType: "item_used",
			},
			setupMock: func(m *MockStatsService) {
				m.On("RecordUserEvent", mock.Anything, "testuser", domain.EventType("item_used"), mock.Anything).Return(errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockStatsService{}
			tt.setupMock(mockSvc)

			handler := HandleRecordEvent(mockSvc)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/stats/event", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandleGetUserStats(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		period         string
		setupMock      func(*MockStatsService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Success",
			userID: "testuser",
			period: "daily",
			setupMock: func(m *MockStatsService) {
				summary := &domain.StatsSummary{TotalEvents: 10}
				m.On("GetUserStats", mock.Anything, "testuser", "daily").Return(summary, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"total_events":10`,
		},
		{
			name:           "Missing UserID",
			userID:         "",
			setupMock:      func(m *MockStatsService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing user_id",
		},
		{
			name:   "Service Error",
			userID: "testuser",
			setupMock: func(m *MockStatsService) {
				m.On("GetUserStats", mock.Anything, "testuser", "daily").Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockStatsService{}
			tt.setupMock(mockSvc)

			handler := HandleGetUserStats(mockSvc)

			url := "/stats/user"
			if tt.userID != "" {
				url += "?user_id=" + tt.userID
			}
			if tt.period != "" {
				url += "&period=" + tt.period
			}
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandleGetSystemStats(t *testing.T) {
	tests := []struct {
		name           string
		period         string
		setupMock      func(*MockStatsService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Success",
			period: "daily",
			setupMock: func(m *MockStatsService) {
				summary := &domain.StatsSummary{TotalEvents: 100}
				m.On("GetSystemStats", mock.Anything, "daily").Return(summary, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"total_events":100`,
		},
		{
			name:   "Service Error",
			period: "daily",
			setupMock: func(m *MockStatsService) {
				m.On("GetSystemStats", mock.Anything, "daily").Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockStatsService{}
			tt.setupMock(mockSvc)

			handler := HandleGetSystemStats(mockSvc)

			url := "/stats/system"
			if tt.period != "" {
				url += "?period=" + tt.period
			}
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandleGetLeaderboard(t *testing.T) {
	tests := []struct {
		name           string
		eventType      string
		limit          string
		setupMock      func(*MockStatsService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "Success",
			eventType: "item_used",
			limit:     "5",
			setupMock: func(m *MockStatsService) {
				entries := []domain.LeaderboardEntry{{UserID: "user1", Count: 10}}
				m.On("GetLeaderboard", mock.Anything, domain.EventType("item_used"), "daily", 5).Return(entries, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"user_id":"user1"`,
		},
		{
			name:           "Missing EventType",
			eventType:      "",
			setupMock:      func(m *MockStatsService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing event_type",
		},
		{
			name:           "Invalid Limit",
			eventType:      "item_used",
			limit:          "invalid",
			setupMock:      func(m *MockStatsService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid limit",
		},
		{
			name:      "Service Error",
			eventType: "item_used",
			setupMock: func(m *MockStatsService) {
				m.On("GetLeaderboard", mock.Anything, domain.EventType("item_used"), "daily", 10).Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockStatsService{}
			tt.setupMock(mockSvc)

			handler := HandleGetLeaderboard(mockSvc)

			url := "/stats/leaderboard"
			if tt.eventType != "" {
				url += "?event_type=" + tt.eventType
			}
			if tt.limit != "" {
				url += "&limit=" + tt.limit
			}
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}
