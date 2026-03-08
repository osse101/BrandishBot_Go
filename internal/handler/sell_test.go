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

func TestHandleSellItem(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockEconomyService, *mocks.MockProgressionService, *mocks.MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: SellItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemMissile,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
				u.On("GetUserIDByPlatformID", mock.Anything, domain.PlatformTwitch, "test-id").Return("", nil)
				e.On("SellItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.ItemMissile, 1).Return(100, 1, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"money_gained":100,"items_sold":1`,
		},
		{
			name: "Feature Locked",
			requestBody: SellItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemMissile,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(false, nil)
				p.On("GetRequiredNodes", mock.Anything, progression.FeatureEconomy).Return([]*domain.ProgressionNode{
					{DisplayName: "Sell System"},
				}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "LOCKED_NODES: Sell System",
		},
		{
			name: "Feature Check Error",
			requestBody: SellItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemMissile,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(false, domain.ErrDatabaseError)
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
			name: "Missing Platform",
			requestBody: SellItemRequest{
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemMissile,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Zero Quantity",
			requestBody: SellItemRequest{
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
			name: "Service Error - Item Not Found",
			requestBody: SellItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "UnknownItem",
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
				e.On("SellItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", "UnknownItem", 1).
					Return(0, 0, errors.New(ErrMsgItemNotFoundError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgItemNotFoundError,
		},
		{
			name: "Service Error - Insufficient Items",
			requestBody: SellItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemMissile,
				Quantity:   100,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService, u *mocks.MockUserService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
				e.On("SellItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.ItemMissile, 100).
					Return(0, 0, errors.New(ErrMsgInsufficientItemsErr))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgInsufficientItemsErr,
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
				return evt.Type == "item.sold" || evt.Type == event.EventTypeEngagement
			})).Return(nil).Maybe()

			handler := HandleSellItem(mockEco, mockUser, mockProg, mockBus)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/item/sell", bytes.NewBuffer(body))
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
