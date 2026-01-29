package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleStartGamble(t *testing.T) {
	tests := []struct {
		name           string
		reqBody        interface{}
		setupMocks     func(*mocks.MockGambleService, *mocks.MockProgressionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Feature Locked",
			reqBody: StartGambleRequest{
				Platform:   "discord",
				PlatformID: "123",
				Username:   "testuser",
				Bets:       []domain.LootboxBet{},
			},
			setupMocks: func(mg *mocks.MockGambleService, mp *mocks.MockProgressionService) {
				mp.On("IsFeatureUnlocked", mock.Anything, progression.FeatureGamble).Return(false, nil)
				mp.On("GetRequiredNodes", mock.Anything, progression.FeatureGamble).Return([]*domain.ProgressionNode{{DisplayName: "Gamble Node"}}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "LOCKED_NODES: Gamble Node",
		},
		{
			name:    "Invalid JSON",
			reqBody: "invalid json",
			setupMocks: func(mg *mocks.MockGambleService, mp *mocks.MockProgressionService) {
				mp.On("IsFeatureUnlocked", mock.Anything, progression.FeatureGamble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name: "Service Error",
			reqBody: StartGambleRequest{
				Platform:   "discord",
				PlatformID: "123",
				Username:   "testuser",
				Bets:       []domain.LootboxBet{},
			},
			setupMocks: func(mg *mocks.MockGambleService, mp *mocks.MockProgressionService) {
				mp.On("IsFeatureUnlocked", mock.Anything, progression.FeatureGamble).Return(true, nil)
				mg.On("StartGamble", mock.Anything, domain.PlatformDiscord, "123", "testuser", mock.Anything).Return(nil, errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
		{
			name: "Success",
			reqBody: StartGambleRequest{
				Platform:   "discord",
				PlatformID: "123",
				Username:   "testuser",
				Bets:       []domain.LootboxBet{},
			},
			setupMocks: func(mg *mocks.MockGambleService, mp *mocks.MockProgressionService) {
				mp.On("IsFeatureUnlocked", mock.Anything, progression.FeatureGamble).Return(true, nil)
				mg.On("StartGamble", mock.Anything, "discord", "123", "testuser", mock.Anything).Return(&domain.Gamble{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `"gamble_id":"00000000-0000-0000-0000-000000000001"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGamble := mocks.NewMockGambleService(t)
			mockProgression := mocks.NewMockProgressionService(t)
			mockEventBus := mocks.NewMockEventBus(t)
			mockEventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			handler := NewGambleHandler(mockGamble, mockProgression, mockEventBus)

			mockProgression.On("RecordEngagement", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			if tt.setupMocks != nil {
				tt.setupMocks(mockGamble, mockProgression)
			}

			var body []byte
			if s, ok := tt.reqBody.(string); ok && s == "invalid json" {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tt.reqBody)
			}

			req := httptest.NewRequest("POST", "/gamble/start", bytes.NewBuffer(body))
			rec := httptest.NewRecorder()

			handler.HandleStartGamble(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, rec.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestHandleJoinGamble(t *testing.T) {
	validUUID := uuid.New()
	tests := []struct {
		name           string
		queryID        string
		reqBody        interface{}
		setupMocks     func(*mocks.MockGambleService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Missing ID",
			queryID:        "",
			reqBody:        JoinGambleRequest{},
			setupMocks:     nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing id query parameter",
		},
		{
			name:           "Invalid ID",
			queryID:        "invalid-uuid",
			reqBody:        JoinGambleRequest{},
			setupMocks:     nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid gamble ID",
		},
		{
			name:           "Invalid JSON",
			queryID:        validUUID.String(),
			reqBody:        "invalid json",
			setupMocks:     nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name:    "Service Error",
			queryID: validUUID.String(),
			reqBody: JoinGambleRequest{
				Platform:   "discord",
				PlatformID: "123",
				Username:   "testuser",
			},
			setupMocks: func(mg *mocks.MockGambleService) {
				mg.On("JoinGamble", mock.Anything, validUUID, domain.PlatformDiscord, "123", "testuser").Return(errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
		{
			name:    "Success",
			queryID: validUUID.String(),
			reqBody: JoinGambleRequest{
				Platform:   "discord",
				PlatformID: "123",
				Username:   "testuser",
			},
			setupMocks: func(mg *mocks.MockGambleService) {
				mg.On("JoinGamble", mock.Anything, validUUID, domain.PlatformDiscord, "123", "testuser").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Successfully joined gamble",
		},
	}

			mockEventBus := mocks.NewMockEventBus(t)
			mockProg := mocks.NewMockProgressionService(t)
			mockProg.On("RecordEngagement", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			mockEventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGamble := mocks.NewMockGambleService(t)
			// Progression service is not used in JoinGamble, so we can pass nil or a mock
			handler := NewGambleHandler(mockGamble, mockProg, mockEventBus)

			if tt.setupMocks != nil {
				tt.setupMocks(mockGamble)
			}

			var body []byte
			if s, ok := tt.reqBody.(string); ok && s == "invalid json" {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tt.reqBody)
			}

			req := httptest.NewRequest("POST", "/gamble/join?id="+tt.queryID, bytes.NewBuffer(body))
			rec := httptest.NewRecorder()

			handler.HandleJoinGamble(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, rec.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestHandleGetGamble(t *testing.T) {
	validUUID := uuid.New()
	tests := []struct {
		name           string
		queryID        string
		setupMocks     func(*mocks.MockGambleService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Missing ID",
			queryID:        "",
			setupMocks:     nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing id query parameter",
		},
		{
			name:           "Invalid ID",
			queryID:        "invalid-uuid",
			setupMocks:     nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid gamble ID",
		},
		{
			name:    "Service Error",
			queryID: validUUID.String(),
			setupMocks: func(mg *mocks.MockGambleService) {
				mg.On("GetGamble", mock.Anything, validUUID).Return(nil, errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
		{
			name:    "Not Found",
			queryID: validUUID.String(),
			setupMocks: func(mg *mocks.MockGambleService) {
				mg.On("GetGamble", mock.Anything, validUUID).Return(nil, nil)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Gamble not found",
		},
		{
			name:    "Success",
			queryID: validUUID.String(),
			setupMocks: func(mg *mocks.MockGambleService) {
				mg.On("GetGamble", mock.Anything, validUUID).Return(&domain.Gamble{ID: validUUID}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   validUUID.String(),
		},
	}

	for _, tt := range tests {
			mockProg := mocks.NewMockProgressionService(t)
			mockEventBus := mocks.NewMockEventBus(t)
		t.Run(tt.name, func(t *testing.T) {
			mockGamble := mocks.NewMockGambleService(t)
			handler := NewGambleHandler(mockGamble, mockProg, mockEventBus)

			if tt.setupMocks != nil {
				tt.setupMocks(mockGamble)
			}

			req := httptest.NewRequest("GET", "/gamble?id="+tt.queryID, nil)
			rec := httptest.NewRecorder()

			handler.HandleGetGamble(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, rec.Body.String(), tt.expectedBody)
			}
		})
	}
}
