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
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleBuyItem(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		requestBody      interface{}
		setupMock        func(*mocks.MockEconomyService, *mocks.MockProgressionService, *mocks.MockUserService)
		expectedStatus   int
		expectedErrorMsg string
		expectedItems    int
	}{
		{
			name: "Success (Typical Quantity)",
			requestBody: BuyItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemMissile,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
				p.On("RecordEngagement", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
				u.On("GetUserIDByPlatformID", mock.Anything, domain.PlatformTwitch, "test-id").Return("", nil)
				e.On("BuyItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.ItemMissile, 1).Return(1, nil)
			},
			expectedStatus: http.StatusOK,
			expectedItems:  1,
		},
		{
			name: "Quantity Boundary: Just Inside (1)",
			requestBody: BuyItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemMissile,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
				p.On("RecordEngagement", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
				u.On("GetUserIDByPlatformID", mock.Anything, domain.PlatformTwitch, "test-id").Return("", nil)
				e.On("BuyItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.ItemMissile, 1).Return(1, nil)
			},
			expectedStatus: http.StatusOK,
			expectedItems:  1,
		},
		{
			name: "Quantity Boundary: Max Value (10000)",
			requestBody: BuyItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemMissile,
				Quantity:   10000,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
				p.On("RecordEngagement", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
				u.On("GetUserIDByPlatformID", mock.Anything, domain.PlatformTwitch, "test-id").Return("", nil)
				e.On("BuyItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.ItemMissile, 10000).Return(10000, nil)
			},
			expectedStatus: http.StatusOK,
			expectedItems:  10000,
		},
		{
			name: "Quantity Boundary: On Lower Boundary (0)",
			requestBody: BuyItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemMissile,
				Quantity:   0,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorMsg: "Invalid request",
		},
		{
			name: "Quantity Boundary: Negative (Beyond Lower, -1)",
			requestBody: BuyItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemMissile,
				Quantity:   -1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorMsg: "Invalid request",
		},
		{
			name: "Quantity Boundary: Beyond Upper (10001)",
			requestBody: BuyItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemMissile,
				Quantity:   10001,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorMsg: "Invalid request",
		},
		{
			name: "Feature Locked",
			requestBody: BuyItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemMissile,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(false, nil)
				p.On("GetRequiredNodes", mock.Anything, progression.FeatureEconomy).Return([]*domain.ProgressionNode{
					{DisplayName: "Buy System"},
				}, nil)
			},
			expectedStatus:   http.StatusForbidden,
			expectedErrorMsg: "LOCKED_NODES: Buy System",
		},
		{
			name: "Feature Check Error",
			requestBody: BuyItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemMissile,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(false, errors.New(ErrMsgGenericServerError))
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedErrorMsg: ErrMsgGenericServerError,
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid json",
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorMsg: "Invalid request body",
		},
		{
			name: "Hostile Case: Control characters in username",
			requestBody: BuyItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "test\nuser",
				ItemName:   domain.ItemMissile,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorMsg: "Invalid request",
		},
		{
			name: "Hostile Case: Null byte in username",
			requestBody: BuyItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "test\x00user",
				ItemName:   domain.ItemMissile,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorMsg: "Invalid request",
		},
		{
			name: "Missing PlatformID",
			requestBody: BuyItemRequest{
				Platform: domain.PlatformTwitch,
				Username: "testuser",
				ItemName: domain.ItemMissile,
				Quantity: 1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorMsg: "Invalid request",
		},
		{
			name: "Service Error - Insufficient Money",
			requestBody: BuyItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "pooruser",
				ItemName:   domain.ItemMissile,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
				e.On("BuyItem", mock.Anything, domain.PlatformTwitch, "test-id", "pooruser", domain.ItemMissile, 1).
					Return(0, errors.New(ErrMsgNotEnoughMoneyError))
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedErrorMsg: ErrMsgNotEnoughMoneyError,
		},
		{
			name: "Service Error - Item Not Available",
			requestBody: BuyItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "RareItem",
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
				e.On("BuyItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", "RareItem", 1).
					Return(0, errors.New(ErrMsgGenericServerError))
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedErrorMsg: ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockEco := mocks.NewMockEconomyService(t)
			mockProg := mocks.NewMockProgressionService(t)
			mockBus := mocks.NewMockEventBus(t)
			mockUser := mocks.NewMockUserService(t)
			tt.setupMock(mockEco, mockProg, mockUser)
			// Allow event publishing
			mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
				return evt.Type == "item.bought" || evt.Type == event.EventTypeEngagement
			})).Return(nil).Maybe()

			handler := HandleBuyItem(mockEco, mockUser, mockProg, mockBus)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/item/buy", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedErrorMsg != "" {
				var errResp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errResp)
				if err == nil && errResp.Error != "" {
					assert.Contains(t, errResp.Error, tt.expectedErrorMsg)
				} else {
					// Fallback for cases where response might not be standard ErrorResponse JSON
					assert.Contains(t, w.Body.String(), tt.expectedErrorMsg)
				}
			} else {
				var successResp BuyItemResponse
				err := json.Unmarshal(w.Body.Bytes(), &successResp)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedItems, successResp.ItemsBought)
			}

			mockEco.AssertExpectations(t)
			mockProg.AssertExpectations(t)
		})
	}
}
