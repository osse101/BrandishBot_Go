package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/slots"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestSlotsHandler_HandleSpinSlots(t *testing.T) {
	t.Parallel()

	const betAmount = 100
	const payoutMultiplier = 2.0
	const expectedPayout = int(float64(betAmount) * payoutMultiplier)
	const expectedMessage = "You won!"

	tests := []struct {
		name           string
		reqBody        interface{}
		setupMocks     func(*mocks.MockProgressionService, *mocks.MockSlotsService)
		expectedStatus int
		expectedError  error
		expectedBody   func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Best Case - Successful Spin",
			reqBody: handler.SpinSlotsRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  betAmount,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
				slotsMock.On("SpinSlots", mock.Anything, domain.PlatformDiscord, "user123", "testuser", betAmount).
					Return(&domain.SlotsResult{
						UserID:           "uuid-123",
						Username:         "testuser",
						Reel1:            slots.SymbolCherry,
						Reel2:            slots.SymbolCherry,
						Reel3:            slots.SymbolCherry,
						BetAmount:        betAmount,
						PayoutAmount:     expectedPayout,
						PayoutMultiplier: payoutMultiplier,
						IsWin:            true,
						IsNearMiss:       false,
						TriggerType:      slots.TriggerNormal,
						Message:          expectedMessage,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp handler.SlotsResult
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)

				assert.Equal(t, "uuid-123", resp.UserID)
				assert.Equal(t, "testuser", resp.Username)
				assert.Equal(t, slots.SymbolCherry, resp.Reel1)
				assert.Equal(t, slots.SymbolCherry, resp.Reel2)
				assert.Equal(t, slots.SymbolCherry, resp.Reel3)
				assert.Equal(t, betAmount, resp.BetAmount)
				assert.Equal(t, expectedPayout, resp.PayoutAmount)
				assert.Equal(t, payoutMultiplier, resp.PayoutMultiplier)
				assert.True(t, resp.IsWin)
				assert.False(t, resp.IsNearMiss)
				assert.Equal(t, slots.TriggerNormal, resp.TriggerType)
				assert.Equal(t, expectedMessage, resp.Message)
			},
		},
		{
			name: "Error Case - Feature Locked",
			reqBody: handler.SpinSlotsRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  100,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(false, nil)
				progMock.On("GetRequiredNodes", mock.Anything, progression.FeatureSlots).Return([]*domain.ProgressionNode{}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  domain.ErrFeatureLocked,
		},
		{
			name: "Error Case - Progression check failed",
			reqBody: handler.SpinSlotsRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  100,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(false, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  domain.ErrDatabaseError,
		},
		{
			name:    "Error Case - Invalid Request Body",
			reqBody: "invalid-json",
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  domain.ErrInvalidInput,
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
			expectedError:  domain.ErrInvalidInput,
		},
		{
			name: "Error Case - Validation Error (Bet Amount Too Low)",
			reqBody: handler.SpinSlotsRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  5,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  domain.ErrInvalidInput,
		},
		{
			name: "Error Case - Validation Error (Bet Amount Too High)",
			reqBody: handler.SpinSlotsRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  15000,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  domain.ErrInvalidInput,
		},
		{
			name: "Error Case - Insufficient Funds",
			reqBody: handler.SpinSlotsRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  100,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
				slotsMock.On("SpinSlots", mock.Anything, domain.PlatformDiscord, "user123", "testuser", 100).
					Return(nil, errors.New("insufficient funds. You have 50 money"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  domain.ErrInsufficientFunds,
		},
		{
			name: "Error Case - Slots feature not yet unlocked from service layer",
			reqBody: handler.SpinSlotsRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  100,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
				slotsMock.On("SpinSlots", mock.Anything, domain.PlatformDiscord, "user123", "testuser", 100).
					Return(nil, errors.New("slots feature is not yet unlocked"))
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  domain.ErrFeatureLocked,
		},
		{
			name: "Error Case - Minimum bet error from service layer",
			reqBody: handler.SpinSlotsRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  10, // Validation passes, but service layer logic kicks in
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
				slotsMock.On("SpinSlots", mock.Anything, domain.PlatformDiscord, "user123", "testuser", 10).
					Return(nil, errors.New("minimum bet is 20 money"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  domain.ErrInvalidInput,
		},
		{
			name: "Error Case - Maximum bet error from service layer",
			reqBody: handler.SpinSlotsRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  10000,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
				slotsMock.On("SpinSlots", mock.Anything, domain.PlatformDiscord, "user123", "testuser", 10000).
					Return(nil, errors.New("maximum bet is 5000 money"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  domain.ErrInvalidInput,
		},
		{
			name: "Error Case - Generic Internal Error",
			reqBody: handler.SpinSlotsRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user123",
				Username:   "testuser",
				BetAmount:  100,
			},
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
				slotsMock.On("SpinSlots", mock.Anything, domain.PlatformDiscord, "user123", "testuser", 100).
					Return(nil, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  domain.ErrInternalError,
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

			if tt.expectedError != nil {
				assert.Contains(t, w.Body.String(), tt.expectedError.Error())
			} else if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}

			// Verify mock expectations
			mockProgSvc.AssertExpectations(t)
			mockSlotsSvc.AssertExpectations(t)
		})
	}
}
