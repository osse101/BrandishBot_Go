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
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestSlotsHandler_HandleSpinSlots(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		reqBody        interface{}
		setupMocks     func(*mocks.MockProgressionService, *mocks.MockSlotsService)
		expectedStatus int
		expectedError  error
		expectedBody   func(t *testing.T, w *httptest.ResponseRecorder)
	}{
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
			name:    "Error Case - Invalid Request Body",
			reqBody: "invalid-json",
			setupMocks: func(progMock *mocks.MockProgressionService, slotsMock *mocks.MockSlotsService) {
				progMock.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSlots).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  errors.New(handler.ErrMsgInvalidRequest),
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
			expectedError:  errors.New(handler.ErrMsgInvalidRequestSummary),
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
			expectedError:  errors.New(handler.ErrMsgInvalidRequestSummary),
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
			expectedError:  errors.New(handler.ErrMsgInvalidRequestSummary),
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

			var resp handler.ErrorResponse
			json.Unmarshal(w.Body.Bytes(), &resp)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != nil {
				assert.Contains(t, resp.Error, tt.expectedError.Error())
			} else if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}

			// Verify mock expectations
			mockProgSvc.AssertExpectations(t)
			mockSlotsSvc.AssertExpectations(t)
		})
	}
}
