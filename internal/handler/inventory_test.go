package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/user"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleAddItem(t *testing.T) {
	// Initialize validator
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: AddItemByUsernameRequest{
				Platform: domain.PlatformTwitch,
				Username: "testuser",
				ItemName: domain.ItemBlaster,
				Quantity: 1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("AddItemByUsername", mock.Anything, domain.PlatformTwitch, "testuser", domain.ItemBlaster, 1).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Item added successfully"}`,
		},
		{
			name: "Invalid Request - Missing Username",
			requestBody: AddItemByUsernameRequest{
				ItemName: domain.ItemBlaster,
				Quantity: 1,
			},
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Service Error",
			requestBody: AddItemByUsernameRequest{
				Platform: domain.PlatformTwitch,
				Username: "testuser",
				ItemName: domain.ItemBlaster,
				Quantity: 1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("AddItemByUsername", mock.Anything, domain.PlatformTwitch, "testuser", domain.ItemBlaster, 1).Return(errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockUserService(t)
			tt.setupMock(mockSvc)

			handler := HandleAddItemByUsername(mockSvc)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/item/add-by-username", bytes.NewBuffer(body))
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

func TestHandleSellItem(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockEconomyService, *mocks.MockProgressionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: SellItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemBlaster,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
				e.On("SellItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.ItemBlaster, 1).Return(100, 1, nil)
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
				ItemName:   domain.ItemBlaster,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
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
				ItemName:   domain.ItemBlaster,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(false, domain.ErrDatabaseError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid json",
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
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
				ItemName:   domain.ItemBlaster,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
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
				ItemName:   domain.ItemBlaster,
				Quantity:   0,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
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
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
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
				ItemName:   domain.ItemBlaster,
				Quantity:   100,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
				e.On("SellItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.ItemBlaster, 100).
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
			tt.setupMock(mockEco, mockProg)
			// Allow event publishing
			mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
				return evt.Type == "item.sold" || evt.Type == event.EventTypeEngagement
			})).Return(nil).Maybe()

			handler := HandleSellItem(mockEco, mockProg, mockBus)

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

func TestHandleRemoveItem(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: RemoveItemByUsernameRequest{
				Platform: domain.PlatformTwitch,
				Username: "testuser",
				ItemName: domain.ItemBlaster,
				Quantity: 1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("RemoveItemByUsername", mock.Anything, domain.PlatformTwitch, "testuser", domain.ItemBlaster, 1).Return(1, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"removed":1`,
		},
		{
			name: "Service Error",
			requestBody: RemoveItemByUsernameRequest{
				Platform: domain.PlatformTwitch,
				Username: "testuser",
				ItemName: domain.ItemBlaster,
				Quantity: 1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("RemoveItemByUsername", mock.Anything, domain.PlatformTwitch, "testuser", domain.ItemBlaster, 1).Return(0, errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockUserService(t)
			tt.setupMock(mockSvc)

			handler := HandleRemoveItemByUsername(mockSvc)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/item/remove-by-username", bytes.NewBuffer(body))
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

func TestHandleGiveItem(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: GiveItemRequest{
				OwnerPlatform:    domain.PlatformTwitch,
				OwnerPlatformID:  "owner-id",
				Owner:            "owner",
				ReceiverPlatform: domain.PlatformTwitch,
				Receiver:         "receiver",
				ItemName:         domain.ItemBlaster,
				Quantity:         1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("GiveItem", mock.Anything, domain.PlatformTwitch, "owner-id", "owner", domain.PlatformTwitch, "receiver", domain.ItemBlaster, 1).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Item transferred successfully"}`,
		},
		{
			name: "Service Error",
			requestBody: GiveItemRequest{
				OwnerPlatform:    domain.PlatformTwitch,
				OwnerPlatformID:  "owner-id",
				Owner:            "owner",
				ReceiverPlatform: domain.PlatformTwitch,
				Receiver:         "receiver",
				ItemName:         domain.ItemBlaster,
				Quantity:         1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("GiveItem", mock.Anything, domain.PlatformTwitch, "owner-id", "owner", domain.PlatformTwitch, "receiver", domain.ItemBlaster, 1).Return(errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockUserService(t)
			tt.setupMock(mockSvc)

			handler := HandleGiveItem(mockSvc)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/item/give", bytes.NewBuffer(body))
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

func TestHandleBuyItem(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockEconomyService, *mocks.MockProgressionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: BuyItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.ItemBlaster,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
				p.On("RecordEngagement", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
				e.On("BuyItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.ItemBlaster, 1).Return(1, nil)
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
				ItemName:   domain.ItemBlaster,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
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
				ItemName:   domain.ItemBlaster,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(false, errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid json",
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
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
				ItemName: domain.ItemBlaster,
				Quantity: 1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
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
				ItemName:   domain.ItemBlaster,
				Quantity:   0,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
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
				ItemName:   domain.ItemBlaster,
				Quantity:   1,
			},
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
				e.On("BuyItem", mock.Anything, domain.PlatformTwitch, "test-id", "pooruser", domain.ItemBlaster, 1).
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
			setupMock: func(e *mocks.MockEconomyService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureEconomy).Return(true, nil)
				e.On("BuyItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", "RareItem", 1).
					Return(0, errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEco := mocks.NewMockEconomyService(t)
			mockProg := mocks.NewMockProgressionService(t)
			mockBus := mocks.NewMockEventBus(t)
			tt.setupMock(mockEco, mockProg)
			// Allow event publishing
			mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
				return evt.Type == "item.bought" || evt.Type == event.EventTypeEngagement
			})).Return(nil).Maybe()

			handler := HandleBuyItem(mockEco, mockProg, mockBus)

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

func TestHandleUseItem(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockUserService, *mocks.MockEventBus)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: UseItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.PublicNameMissile,
				Quantity:   1,
			},
			setupMock: func(u *mocks.MockUserService, e *mocks.MockEventBus) {
				// Mock should return what the real blaster handler would return
				u.On("UseItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.PublicNameMissile, 1, "").
					Return("testuser has BLASTED target 1 times! They are timed out for 1m0s.", nil)
				// Expect both engagement and item.used events
				e.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
					return evt.Type == "engagement" || evt.Type == "item.used"
				})).Return(nil).Maybe()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"message":"testuser has BLASTED target 1 times`,
		},
		{
			name: "Service Error",
			requestBody: UseItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   domain.PublicNameMissile,
				Quantity:   1,
			},
			setupMock: func(u *mocks.MockUserService, e *mocks.MockEventBus) {
				u.On("UseItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.PublicNameMissile, 1, "").Return("", errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUser := mocks.NewMockUserService(t)
			mockProg := mocks.NewMockProgressionService(t)
			mockBus := mocks.NewMockEventBus(t)
			mockProg.On("RecordEngagement", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			tt.setupMock(mockUser, mockBus)

			handler := HandleUseItem(mockUser, mockProg, mockBus)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/item/use", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mockUser.AssertExpectations(t)
			mockBus.AssertExpectations(t)
		})
	}
}

func TestHandleGetInventory(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		username       string
		platform       string
		platformID     string
		filter         string
		setupMock      func(*mocks.MockUserService, *mocks.MockProgressionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:       "Success",
			username:   "testuser",
			platform:   domain.PlatformDiscord,
			platformID: "test-platformid",
			filter:     "",
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				items := []user.InventoryItem{
					{Name: domain.ItemBlaster, Quantity: 1},
				}
				m.On("GetInventory", mock.Anything, domain.PlatformDiscord, "test-platformid", "testuser", "").Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"items":[{"name":"weapon_blaster","quantity":1}]`,
		},
		{
			name:       "Success with Filter",
			username:   "testuser",
			platform:   domain.PlatformDiscord,
			platformID: "test-platformid",
			filter:     domain.FilterTypeUpgrade,
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				items := []user.InventoryItem{
					{Name: domain.ItemLootbox0, Quantity: 1},
				}
				p.On("IsFeatureUnlocked", mock.Anything, "feature_filter_upgrade").Return(true, nil)
				m.On("GetInventory", mock.Anything, domain.PlatformDiscord, "test-platformid", "testuser", domain.FilterTypeUpgrade).Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"items":[{"name":"lootbox_tier0","quantity":1}]`,
		},
		{
			name:       "Filter Locked",
			username:   "testuser",
			platform:   domain.PlatformDiscord,
			platformID: "test-platformid",
			filter:     domain.FilterTypeUpgrade,
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_filter_upgrade").Return(false, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Filter 'upgrade' is locked",
		},
		{
			name:           "Missing Username",
			username:       "",
			platform:       domain.PlatformDiscord,
			platformID:     "test-platformid",
			filter:         "",
			setupMock:      func(m *mocks.MockUserService, p *mocks.MockProgressionService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing username query parameter",
		},
		{
			name:       "Service Error",
			username:   "testuser",
			platform:   domain.PlatformDiscord,
			platformID: "test-platformid",
			filter:     "",
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				m.On("GetInventory", mock.Anything, domain.PlatformDiscord, "test-platformid", "testuser", "").Return(nil, errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
		{
			name:       "Sellable Filter - Unlocked",
			username:   "testuser",
			platform:   domain.PlatformDiscord,
			platformID: "test-platformid",
			filter:     domain.FilterTypeSellable,
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				items := []user.InventoryItem{
					{Name: domain.ItemLootbox1, Quantity: 5},
				}
				p.On("IsFeatureUnlocked", mock.Anything, "feature_filter_sellable").Return(true, nil)
				m.On("GetInventory", mock.Anything, domain.PlatformDiscord, "test-platformid", "testuser", domain.FilterTypeSellable).Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"items":[{"name":"lootbox_tier1","quantity":5}]`,
		},
		{
			name:       "Sellable Filter - Locked",
			username:   "testuser",
			platform:   domain.PlatformDiscord,
			platformID: "test-platformid",
			filter:     domain.FilterTypeSellable,
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_filter_sellable").Return(false, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Filter 'sellable' is locked",
		},
		{
			name:       "Consumable Filter - Unlocked",
			username:   "testuser",
			platform:   domain.PlatformDiscord,
			platformID: "test-platformid",
			filter:     domain.FilterTypeConsumable,
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				items := []user.InventoryItem{
					{Name: domain.ItemLootbox0, Quantity: 3},
				}
				p.On("IsFeatureUnlocked", mock.Anything, "feature_filter_consumable").Return(true, nil)
				m.On("GetInventory", mock.Anything, domain.PlatformDiscord, "test-platformid", "testuser", domain.FilterTypeConsumable).Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"items":[{"name":"lootbox_tier0","quantity":3}]`,
		},
		{
			name:       "Consumable Filter - Locked",
			username:   "testuser",
			platform:   domain.PlatformDiscord,
			platformID: "test-platformid",
			filter:     domain.FilterTypeConsumable,
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_filter_consumable").Return(false, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Filter 'consumable' is locked",
		},
		{
			name:       "Unknown Filter - Invalid",
			username:   "testuser",
			platform:   domain.PlatformDiscord,
			platformID: "test-platformid",
			filter:     "unknown",
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				// No expectations - validation happens before service call
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid filter type 'unknown'",
		},
		{
			name:       "Filter Check Error",
			username:   "testuser",
			platform:   domain.PlatformDiscord,
			platformID: "test-platformid",
			filter:     domain.FilterTypeUpgrade,
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_filter_upgrade").Return(false, domain.ErrDatabaseError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUser := mocks.NewMockUserService(t)
			mockProg := mocks.NewMockProgressionService(t)
			tt.setupMock(mockUser, mockProg)

			handler := HandleGetInventory(mockUser, mockProg)

			// Build URL with query parameters
			params := []string{}
			if tt.platform != "" {
				params = append(params, "platform="+tt.platform)
			}
			if tt.platformID != "" {
				params = append(params, "platform_id="+tt.platformID)
			}
			if tt.username != "" {
				params = append(params, "username="+tt.username)
			}
			if tt.filter != "" {
				params = append(params, "filter="+tt.filter)
			}
			url := "/user/inventory"
			if len(params) > 0 {
				url += "?" + strings.Join(params, "&")
			}
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mockUser.AssertExpectations(t)
			mockProg.AssertExpectations(t)
		})
	}
}
