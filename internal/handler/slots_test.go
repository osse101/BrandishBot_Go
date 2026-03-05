package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSlotsHandler_HandleSpinSlots(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		reqBody        interface{}
		setupMocks     func(*mocks.MockProgressionService, *mocks.MockSlotsService)
		expectedStatus int
		expectedError  string
		expectedBody   string
	}{
		{
			name: "Best Case - Successful Spin",
			reqBody: handler.SpinSlotsRequest{
				Platform:   "discord",
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  100,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
				slotsMock.On("SpinSlots", mock.Anything, "discord", "user123", "testuser", 100).
					Return(&domain.SlotsResult{
						UserID:           "uuid-123",
						Username:         "testuser",
						Reel1:            "cherry",
						Reel2:            "cherry",
						Reel3:            "cherry",
						BetAmount:        100,
						PayoutAmount:     200,
						PayoutMultiplier: 2.0,
						IsWin:            true,
						IsNearMiss:       false,
						TriggerType:      "normal",
						Message:          "You won!",
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"user_id":"uuid-123","username":"testuser","reel1":"cherry","reel2":"cherry","reel3":"cherry","bet_amount":100,"payout_amount":200,"payout_multiplier":2,"is_win":true,"is_near_miss":false,"trigger_type":"normal","message":"You won!"}`,
		},
		{
			name: "Error Case - Feature Locked",
			reqBody: handler.SpinSlotsRequest{
				Platform:   "discord",
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  100,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(false, nil)
				progMock.On("GetRequiredNodes", mock.Anything, progression.FeatureSlots).Return([]*domain.ProgressionNode{}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Feature locked",
		},
		{
			name: "Error Case - Progression check failed",
			reqBody: handler.SpinSlotsRequest{
				Platform:   "discord",
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  100,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(false, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "db error",
		},
		{
			name:       "Error Case - Invalid Request Body",
			reqBody:    "invalid-json",
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name: "Error Case - Validation Error (Missing Platform)",
			reqBody: handler.SpinSlotsRequest{
				Platform:   "",
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  100,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request",
		},
		{
			name: "Error Case - Validation Error (Bet Amount Too Low)",
			reqBody: handler.SpinSlotsRequest{
				Platform:   "discord",
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  5,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request",
		},
		{
			name: "Error Case - Validation Error (Bet Amount Too High)",
			reqBody: handler.SpinSlotsRequest{
				Platform:   "discord",
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  15000,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request",
		},
		{
			name: "Error Case - Insufficient Funds",
			reqBody: handler.SpinSlotsRequest{
				Platform:   "discord",
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  100,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
				slotsMock.On("SpinSlots", mock.Anything, "discord", "user123", "testuser", 100).
					Return(nil, errors.New("insufficient funds. You have 50 money"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "insufficient funds. You have 50 money",
		},
		{
			name: "Error Case - Slots feature not yet unlocked from service layer",
			reqBody: handler.SpinSlotsRequest{
				Platform:   "discord",
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  100,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
				slotsMock.On("SpinSlots", mock.Anything, "discord", "user123", "testuser", 100).
					Return(nil, errors.New("slots feature is not yet unlocked"))
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "slots feature is not yet unlocked",
		},
		{
			name: "Error Case - Minimum bet error from service layer",
			reqBody: handler.SpinSlotsRequest{
				Platform:   "discord",
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  10, // Validation passes, but service layer logic kicks in
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
				slotsMock.On("SpinSlots", mock.Anything, "discord", "user123", "testuser", 10).
					Return(nil, errors.New("minimum bet is 20 money"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "minimum bet is 20 money",
		},
		{
			name: "Error Case - Maximum bet error from service layer",
			reqBody: handler.SpinSlotsRequest{
				Platform:   "discord",
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  10000,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
				slotsMock.On("SpinSlots", mock.Anything, "discord", "user123", "testuser", 10000).
					Return(nil, errors.New("maximum bet is 5000 money"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "maximum bet is 5000 money",
		},
		{
			name: "Error Case - Generic Internal Error",
			reqBody: handler.SpinSlotsRequest{
				Platform:   "discord",
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  100,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
				slotsMock.On("SpinSlots", mock.Anything, "discord", "user123", "testuser", 100).
					Return(nil, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to process slots spin",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup mocks
			mockProgSvc := new(mocks.MockProgressionService)
			mockSlotsSvc := new(mocks.MockSlotsService)

			if tt.setupMocks != nil {
				tt.setupMocks(mockProgSvc, mockSlotsSvc)
			}

			// Create handler
			h := handler.NewSlotsHandler(mockSlotsSvc, mockProgSvc)

			// Create request body
			var reqBodyBytes []byte
			if str, ok := tt.reqBody.(string); ok {
				reqBodyBytes = []byte(str)
			} else {
				var err error
				reqBodyBytes, err = json.Marshal(tt.reqBody)
				require.NoError(t, err)
			}

			// Execute request
			req := httptest.NewRequest(http.MethodPost, "/slots/spin", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.HandleSpinSlots(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			} else if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}

			// Verify mock expectations
			mockProgSvc.AssertExpectations(t)
			mockSlotsSvc.AssertExpectations(t)
		})
	}
}
