package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleRecordEvent(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockStatsService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: RecordEventRequest{
				UserID:    "testuser",
				EventType: domain.EventTypeItemUsed,
				EventData: map[string]interface{}{"item": "potion"},
			},
			setupMock: func(m *mocks.MockStatsService) {
				m.On("RecordUserEvent", mock.Anything, "testuser", domain.EventType(domain.EventTypeItemUsed), mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Event recorded successfully",
		},
		{
			name: "Invalid Request - Missing UserID",
			requestBody: RecordEventRequest{
				EventType: domain.EventTypeItemUsed,
			},
			setupMock:      func(m *mocks.MockStatsService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Service Error",
			requestBody: RecordEventRequest{
				UserID:    "testuser",
				EventType: domain.EventTypeItemUsed,
			},
			setupMock: func(m *mocks.MockStatsService) {
				m.On("RecordUserEvent", mock.Anything, "testuser", domain.EventType(domain.EventTypeItemUsed), mock.Anything).Return(errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockStatsService(t)
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
		platform       string
		platformID     string
		period         string
		setupMock      func(*mocks.MockStatsService, *mocks.MockRepositoryUser)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:       "Success with platform_id",
			platform:   domain.PlatformTwitch,
			platformID: "test123",
			period:     domain.PeriodDaily,
			setupMock: func(svc *mocks.MockStatsService, repo *mocks.MockRepositoryUser) {
				user := &domain.User{ID: "user123"}
				repo.On("GetUserByPlatformID", mock.Anything, domain.PlatformTwitch, "test123").Return(user, nil)
				summary := &domain.StatsSummary{TotalEvents: 10}
				svc.On("GetUserStats", mock.Anything, "user123", domain.PeriodDaily).Return(summary, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"total_events":10`,
		},
		{
			name:       "Missing platform",
			platform:   "",
			platformID: "test123",
			setupMock:  func(svc *mocks.MockStatsService, repo *mocks.MockRepositoryUser) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing platform",
		},
		{
			name:       "Missing platform_id and username",
			platform:   domain.PlatformTwitch,
			platformID: "",
			setupMock:  func(svc *mocks.MockStatsService, repo *mocks.MockRepositoryUser) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Either platform_id or username is required",
		},
		{
			name:       "Service Error",
			platform:   domain.PlatformTwitch,
			platformID: "test123",
			setupMock: func(svc *mocks.MockStatsService, repo *mocks.MockRepositoryUser) {
				user := &domain.User{ID: "user123"}
				repo.On("GetUserByPlatformID", mock.Anything, domain.PlatformTwitch, "test123").Return(user, nil)
				svc.On("GetUserStats", mock.Anything, "user123", domain.PeriodDaily).Return(nil, errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockStatsService(t)
			mockUserRepo := mocks.NewMockRepositoryUser(t)
			tt.setupMock(mockSvc, mockUserRepo)

			statsHandler := NewStatsHandler(mockSvc, mockUserRepo)
			handler := statsHandler.HandleGetUserStats()

			url := "/stats/user"
			if tt.platform != "" {
				url += "?platform=" + tt.platform
			}
			if tt.platformID != "" {
				url += "&platform_id=" + tt.platformID
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
			mockUserRepo.AssertExpectations(t)
		})
	}
}

func TestHandleGetSystemStats(t *testing.T) {
	tests := []struct {
		name           string
		period         string
		setupMock      func(*mocks.MockStatsService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Success",
			period: domain.PeriodDaily,
			setupMock: func(m *mocks.MockStatsService) {
				summary := &domain.StatsSummary{TotalEvents: 100}
				m.On("GetSystemStats", mock.Anything, domain.PeriodDaily).Return(summary, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"total_events":100`,
		},
		{
			name:   "Service Error",
			period: domain.PeriodDaily,
			setupMock: func(m *mocks.MockStatsService) {
				m.On("GetSystemStats", mock.Anything, domain.PeriodDaily).Return(nil, errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockStatsService(t)
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
		setupMock      func(*mocks.MockStatsService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "Success",
			eventType: domain.EventTypeItemUsed,
			limit:     "5",
			setupMock: func(m *mocks.MockStatsService) {
				entries := []domain.LeaderboardEntry{{UserID: "user1", Count: 10}}
				m.On("GetLeaderboard", mock.Anything, domain.EventType(domain.EventTypeItemUsed), domain.PeriodDaily, 5).Return(entries, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"user_id":"user1"`,
		},
		{
			name:           "Missing EventType",
			eventType:      "",
			setupMock:      func(m *mocks.MockStatsService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing event_type",
		},
		{
			name:           "Invalid Limit",
			eventType:      domain.EventTypeItemUsed,
			limit:          "invalid",
			setupMock:      func(m *mocks.MockStatsService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid limit",
		},
		{
			name:      "Service Error",
			eventType: domain.EventTypeItemUsed,
			setupMock: func(m *mocks.MockStatsService) {
				m.On("GetLeaderboard", mock.Anything, domain.EventType(domain.EventTypeItemUsed), domain.PeriodDaily, 10).Return(nil, errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockStatsService(t)
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
