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

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockEconomyService, *mocks.MockProgressionService, *mocks.MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
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
			expectedBody:   `"items_bought":1`,
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
			expectedStatus: http.StatusForbidden,
			expectedBody:   "LOCKED_NODES: Buy System",
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
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid json",
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
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
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Zero Quantity",
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
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
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
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgNotEnoughMoneyError,
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
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
		{
			name: "Negative Quantity",
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
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Quantity Too Large",
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
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mockEco.AssertExpectations(t)
			mockProg.AssertExpectations(t)
		})
	}
}
