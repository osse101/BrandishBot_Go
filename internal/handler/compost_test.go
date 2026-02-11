package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/compost"
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
			requestBody: CompostDepositRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "user123",
				Items: []compost.DepositItem{
					{ItemName: "herb", Quantity: 3},
				},
			},
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(true, nil)

				readyAt := time.Now().Add(2 * time.Hour)
				bin := &domain.CompostBin{
					ID:        "bin-1",
					UserID:    "user123",
					Status:    domain.CompostBinStatusComposting,
					Capacity:  5,
					ItemCount: 3,
					ReadyAt:   &readyAt,
				}
				m.On("Deposit", mock.Anything, domain.PlatformTwitch, "user123", []compost.DepositItem{
					{ItemName: "herb", Quantity: 3},
				}).Return(bin, nil)

				pm.On("RecordEngagement", mock.Anything, "user123", "compost_deposit", 1).
					Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   MsgCompostDepositSuccess,
		},
		{
			name: "Feature Locked",
			requestBody: CompostDepositRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "user123",
				Items: []compost.DepositItem{
					{ItemName: "herb", Quantity: 1},
				},
			},
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(false, nil)
				pm.On("GetRequiredNodes", mock.Anything, progression.FeatureCompost).
					Return([]*domain.ProgressionNode{}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   ErrMsgFeatureLocked,
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid-json",
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Service Error",
			requestBody: CompostDepositRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "user123",
				Items: []compost.DepositItem{
					{ItemName: "herb", Quantity: 1},
				},
			},
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(true, nil)

				m.On("Deposit", mock.Anything, domain.PlatformTwitch, "user123", []compost.DepositItem{
					{ItemName: "herb", Quantity: 1},
				}).Return(nil, errors.New("service error"))
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

func TestHandleStatus(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		setupMocks     func(*mocks.MockCompostService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success - Idle Bin",
			queryParams: map[string]string{
				"platform":    domain.PlatformTwitch,
				"platform_id": "user123",
			},
			setupMocks: func(m *mocks.MockCompostService) {
				result := &domain.HarvestResult{
					Harvested: false,
					Status: &domain.CompostStatusResponse{
						Status:    domain.CompostBinStatusIdle,
						Capacity:  5,
						ItemCount: 0,
						Items:     []domain.CompostBinItem{},
						TimeLeft:  compost.MsgBinEmpty,
					},
				}
				m.On("Harvest", mock.Anything, domain.PlatformTwitch, "user123", "").
					Return(result, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"status":"idle"`,
		},
		{
			name: "Missing platform",
			queryParams: map[string]string{
				"platform_id": "user123",
			},
			setupMocks:     func(m *mocks.MockCompostService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "platform",
		},
		{
			name: "Missing platform_id",
			queryParams: map[string]string{
				"platform": domain.PlatformTwitch,
			},
			setupMocks:     func(m *mocks.MockCompostService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "platform_id",
		},
		{
			name: "Service Error",
			queryParams: map[string]string{
				"platform":    domain.PlatformTwitch,
				"platform_id": "user123",
			},
			setupMocks: func(m *mocks.MockCompostService) {
				m.On("Harvest", mock.Anything, domain.PlatformTwitch, "user123", "").
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

			h.HandleStatus(rec, req)

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
			name: "Success - Harvest Ready",
			requestBody: CompostHarvestRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "user123",
				Username:   "testuser",
			},
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(true, nil)

				result := &domain.HarvestResult{
					Harvested: true,
					Output: &domain.CompostOutput{
						Items:      map[string]int{"iron_ore": 2},
						IsSludge:   false,
						TotalValue: 20,
						Message:    compost.MsgHarvestComplete,
					},
				}
				m.On("Harvest", mock.Anything, domain.PlatformTwitch, "user123", "testuser").
					Return(result, nil)

				pm.On("RecordEngagement", mock.Anything, "user123", "compost_harvest", 5).
					Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   compost.MsgHarvestComplete,
		},
		{
			name: "Not Ready - Returns Status",
			requestBody: CompostHarvestRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "user123",
				Username:   "testuser",
			},
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(true, nil)

				result := &domain.HarvestResult{
					Harvested: false,
					Status: &domain.CompostStatusResponse{
						Status:    domain.CompostBinStatusComposting,
						Capacity:  5,
						ItemCount: 3,
						TimeLeft:  "1h 30m",
					},
				}
				m.On("Harvest", mock.Anything, domain.PlatformTwitch, "user123", "testuser").
					Return(result, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "1h 30m",
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid-json",
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Service Error",
			requestBody: CompostHarvestRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "user123",
				Username:   "testuser",
			},
			setupMocks: func(m *mocks.MockCompostService, pm *mocks.MockProgressionService) {
				pm.On("IsFeatureUnlocked", mock.Anything, progression.FeatureCompost).
					Return(true, nil)
				m.On("Harvest", mock.Anything, domain.PlatformTwitch, "user123", "testuser").
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
