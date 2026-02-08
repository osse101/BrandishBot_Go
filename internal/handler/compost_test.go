package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleDeposit(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*mocks.MockCompostService, *mocks.MockProgressionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: DepositRequest{
				Platform:   "twitch",
				PlatformID: "user123",
				ItemKey:    "item1",
				Quantity:   5,
			},
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(true, nil)

				deposit := &domain.CompostDeposit{
					ID:          uuid.New(),
					ReadyAt:     time.Now().Add(24 * time.Hour),
					DepositedAt: time.Now(),
				}
				m.On("Deposit", mock.Anything, "twitch", "user123", "item1", 5).
					Return(deposit, nil)

				pm.On("RecordEngagement", mock.Anything, "user123", "compost_deposit", 1).
					Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "Items composting!",
		},
		{
			name: "Feature Locked",
			requestBody: DepositRequest{
				Platform:   "twitch",
				PlatformID: "user123",
			},
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(false, nil)
				pm.On("GetRequiredNodes", mock.Anything, progression.FeatureCompost).
					Return([]*domain.ProgressionNode{}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Feature locked",
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid-json",
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name: "Service Error",
			requestBody: DepositRequest{
				Platform:   "twitch",
				PlatformID: "user123",
				ItemKey:    "item1",
				Quantity:   5,
			},
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(true, nil)

				m.On("Deposit", mock.Anything, "twitch", "user123", "item1", 5).
					Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockCompostService(t)
			mockProgression := mocks.NewMockProgressionService(t)

			if tt.setupMocks != nil {
				tt.setupMocks(mockService, mockProgression)
			}

			h := NewCompostHandler(mockService, mockProgression)

			var body []byte
			if s, ok := tt.requestBody.(string); ok && s == "invalid-json" {
				body = []byte(s)
			} else {
				var err error
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest("POST", "/compost/deposit", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			h.HandleDeposit(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, rec.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestHandleGetStatus(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		setupMocks     func(*mocks.MockCompostService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			queryParams: map[string]string{
				"username": "user123",
			},
			setupMocks: func(m *mocks.MockCompostService) {
				status := &domain.CompostStatus{
					ReadyCount:       2,
					TotalGemsPending: 100,
				}
				m.On("GetStatus", mock.Anything, "twitch", "user123").
					Return(status, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"ready_count":2`,
		},
		{
			name:           "Missing Username",
			queryParams:    map[string]string{},
			setupMocks:     func(m *mocks.MockCompostService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing username query parameter",
		},
		{
			name: "Service Error",
			queryParams: map[string]string{
				"username": "user123",
			},
			setupMocks: func(m *mocks.MockCompostService) {
				m.On("GetStatus", mock.Anything, "twitch", "user123").
					Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockCompostService(t)
			mockProgression := mocks.NewMockProgressionService(t)

			if tt.setupMocks != nil {
				tt.setupMocks(mockService)
			}

			h := NewCompostHandler(mockService, mockProgression)

			req := httptest.NewRequest("GET", "/compost/status", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			rec := httptest.NewRecorder()

			h.HandleGetStatus(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, rec.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestHandleHarvest(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*mocks.MockCompostService, *mocks.MockProgressionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: HarvestRequest{
				Platform:   "twitch",
				PlatformID: "user123",
			},
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(true, nil)
				m.On("Harvest", mock.Anything, "twitch", "user123").
					Return(50, nil)

				pm.On("RecordEngagement", mock.Anything, "user123", "compost_harvest", 1).
					Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `"gems_awarded":50`,
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid-json",
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name: "Service Error",
			requestBody: HarvestRequest{
				Platform:   "twitch",
				PlatformID: "user123",
			},
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(true, nil)
				m.On("Harvest", mock.Anything, "twitch", "user123").
					Return(0, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockCompostService(t)
			mockProgression := mocks.NewMockProgressionService(t)

			if tt.setupMocks != nil {
				tt.setupMocks(mockService, mockProgression)
			}

			h := NewCompostHandler(mockService, mockProgression)

			var body []byte
			if s, ok := tt.requestBody.(string); ok && s == "invalid-json" {
				body = []byte(s)
			} else {
				var err error
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest("POST", "/compost/harvest", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			h.HandleHarvest(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, rec.Body.String(), tt.expectedBody)
			}
		})
	}
}
