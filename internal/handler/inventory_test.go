package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserService mocks the user.Service interface
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) AddItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) error {
	args := m.Called(ctx, platform, platformID, username, itemName, quantity)
	return args.Error(0)
}

func (m *MockUserService) RemoveItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, error) {
	args := m.Called(ctx, platform, platformID, username, itemName, quantity)
	return args.Int(0), args.Error(1)
}

func (m *MockUserService) GiveItem(ctx context.Context, ownerPlatform, ownerPlatformID, ownerUsername, receiverPlatform, receiverPlatformID, receiverUsername, itemName string, quantity int) error {
	args := m.Called(ctx, ownerPlatform, ownerPlatformID, ownerUsername, receiverPlatform, receiverPlatformID, receiverUsername, itemName, quantity)
	return args.Error(0)
}

func (m *MockUserService) GetInventory(ctx context.Context, platform, platformID, username string) ([]user.UserInventoryItem, error) {
	args := m.Called(ctx, platform, platformID, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]user.UserInventoryItem), args.Error(1)
}

func (m *MockUserService) UseItem(ctx context.Context, platform, platformID, username, itemName string, quantity int, targetUsername string) (string, error) {
	args := m.Called(ctx, platform, platformID, username, itemName, quantity, targetUsername)
	return args.String(0), args.Error(1)
}

func (m *MockUserService) HandleIncomingMessage(ctx context.Context, platform, platformID, username, message string) (*domain.MessageResult, error) {
	args := m.Called(ctx, platform, platformID, username, message)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MessageResult), args.Error(1)
}

func (m *MockUserService) FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) GetUser(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) UpdatePlatformID(ctx context.Context, userID, platform, platformID string) error {
	args := m.Called(ctx, userID, platform, platformID)
	return args.Error(0)
}

func (m *MockUserService) RegisterUser(ctx context.Context, user domain.User) (domain.User, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(domain.User), args.Error(1)
}

func (m *MockUserService) TimeoutUser(ctx context.Context, username string, duration time.Duration, reason string) error {
	args := m.Called(ctx, username, duration, reason)
	return args.Error(0)
}

func (m *MockUserService) LoadLootTables(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockUserService) HandleSearch(ctx context.Context, platform, platformID, username string) (string, error) {
	args := m.Called(ctx, platform, platformID, username)
	return args.String(0), args.Error(1)
}

// MockEconomyService mocks the economy.Service interface
type MockEconomyService struct {
	mock.Mock
}

func (m *MockEconomyService) SellItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, int, error) {
	args := m.Called(ctx, platform, platformID, username, itemName, quantity)
	return args.Int(0), args.Int(1), args.Error(2)
}

func (m *MockEconomyService) BuyItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, error) {
	args := m.Called(ctx, platform, platformID, username, itemName, quantity)
	return args.Int(0), args.Error(1)
}

func (m *MockEconomyService) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Item), args.Error(1)
}

// MockProgressionService mocks the progression.Service interface
type MockProgressionService struct {
	mock.Mock
}

func (m *MockProgressionService) IsFeatureUnlocked(ctx context.Context, feature string) (bool, error) {
	args := m.Called(ctx, feature)
	return args.Bool(0), args.Error(1)
}

func (m *MockProgressionService) IsItemUnlocked(ctx context.Context, itemName string) (bool, error) {
	args := m.Called(ctx, itemName)
	return args.Bool(0), args.Error(1)
}

func (m *MockProgressionService) RecordEngagement(ctx context.Context, userID, eventType string, count int) error {
	args := m.Called(ctx, userID, eventType, count)
	return args.Error(0)
}

// MockEventBus mocks the event.Bus interface
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(ctx context.Context, evt event.Event) error {
	args := m.Called(ctx, evt)
	return args.Error(0)
}

func (m *MockEventBus) Subscribe(eventType event.Type, handler event.Handler) {
	m.Called(eventType, handler)
}

func (m *MockProgressionService) GetProgressionTree(ctx context.Context) ([]*domain.ProgressionTreeNode, error) {
	return nil, nil
}
func (m *MockProgressionService) GetAvailableUnlocks(ctx context.Context) ([]*domain.ProgressionNode, error) {
	return nil, nil
}
func (m *MockProgressionService) VoteForUnlock(ctx context.Context, userID, nodeKey string) error {
	return nil
}
func (m *MockProgressionService) GetProgressionStatus(ctx context.Context) (*domain.ProgressionStatus, error) {
	return nil, nil
}
func (m *MockProgressionService) GetEngagementScore(ctx context.Context) (int, error) {
	return 0, nil
}
func (m *MockProgressionService) GetUserEngagement(ctx context.Context, userID string) (*domain.ContributionBreakdown, error) {
	return nil, nil
}
func (m *MockProgressionService) CheckAndUnlockCriteria(ctx context.Context) (*domain.ProgressionUnlock, error) {
	return nil, nil
}
func (m *MockProgressionService) ForceInstantUnlock(ctx context.Context) (*domain.ProgressionUnlock, error) {
	return nil, nil
}
func (m *MockProgressionService) AdminUnlock(ctx context.Context, feature string, level int) error {
	return nil
}
func (m *MockProgressionService) AdminRelock(ctx context.Context, feature string, level int) error {
	return nil
}
func (m *MockProgressionService) AdminInstantUnlock(ctx context.Context, feature string) error {
	return nil
}
func (m *MockProgressionService) ResetProgressionTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error {
	return nil
}

func TestHandleAddItem(t *testing.T) {
	// Initialize validator
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: AddItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   1,
			},
			setupMock: func(m *MockUserService) {
				m.On("AddItem", mock.Anything, "twitch", "test-id", "testuser", "Sword", 1).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Item added successfully"}`,
		},
		{
			name: "Invalid Request - Missing Username",
			requestBody: AddItemRequest{
				ItemName: "Sword",
				Quantity: 1,
			},
			setupMock:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Service Error",
			requestBody: AddItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   1,
			},
			setupMock: func(m *MockUserService) {
				m.On("AddItem", mock.Anything, "twitch", "test-id", "testuser", "Sword", 1).Return(errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to add item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockUserService{}
			tt.setupMock(mockSvc)

			handler := HandleAddItem(mockSvc)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/item/add", bytes.NewBuffer(body))
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
		setupMock      func(*MockEconomyService, *MockProgressionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: SellItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   1,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSell).Return(true, nil)
				e.On("SellItem", mock.Anything, "twitch", "test-id", "testuser", "Sword", 1).Return(100, 1, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"money_gained":100,"items_sold":1}`,
		},
		{
			name: "Feature Locked",
			requestBody: SellItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   1,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSell).Return(false, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Sell feature is not yet unlocked",
		},
		{
			name: "Feature Check Error",
			requestBody: SellItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   1,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSell).Return(false, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to check feature availability",
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid json",
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSell).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name: "Missing Platform",
			requestBody: SellItemRequest{
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   1,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSell).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Zero Quantity",
			requestBody: SellItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   0,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSell).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Service Error - Item Not Found",
			requestBody: SellItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "UnknownItem",
				Quantity:   1,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSell).Return(true, nil)
				e.On("SellItem", mock.Anything, "twitch", "test-id", "testuser", "UnknownItem", 1).
					Return(0, 0, errors.New("item not found"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "item not found",
		},
		{
			name: "Service Error - Insufficient Items",
			requestBody: SellItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   100,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSell).Return(true, nil)
				e.On("SellItem", mock.Anything, "twitch", "test-id", "testuser", "Sword", 100).
					Return(0, 0, errors.New("insufficient items"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "insufficient items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEco := &MockEconomyService{}
			mockProg := &MockProgressionService{}
			mockBus := &MockEventBus{}
			tt.setupMock(mockEco, mockProg)
			// Allow event publishing
			mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
				return evt.Type == "item.sold"
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
		setupMock      func(*MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: RemoveItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   1,
			},
			setupMock: func(m *MockUserService) {
				m.On("RemoveItem", mock.Anything, "twitch", "test-id", "testuser", "Sword", 1).Return(1, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"removed":1}`,
		},
		{
			name: "Service Error",
			requestBody: RemoveItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   1,
			},
			setupMock: func(m *MockUserService) {
				m.On("RemoveItem", mock.Anything, "twitch", "test-id", "testuser", "Sword", 1).Return(0, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockUserService{}
			tt.setupMock(mockSvc)

			handler := HandleRemoveItem(mockSvc)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/item/remove", bytes.NewBuffer(body))
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
		setupMock      func(*MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: GiveItemRequest{
				OwnerPlatform:      "twitch",
				OwnerPlatformID:    "owner-id",
				Owner:              "owner",
				ReceiverPlatform:   "twitch",
				ReceiverPlatformID: "receiver-id",
				Receiver:           "receiver",
				ItemName:           "Sword",
				Quantity:           1,
			},
			setupMock: func(m *MockUserService) {
				m.On("GiveItem", mock.Anything, "twitch", "owner-id", "owner", "twitch", "receiver-id", "receiver", "Sword", 1).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Item transferred successfully"}`,
		},
		{
			name: "Service Error",
			requestBody: GiveItemRequest{
				OwnerPlatform:      "twitch",
				OwnerPlatformID:    "owner-id",
				Owner:              "owner",
				ReceiverPlatform:   "twitch",
				ReceiverPlatformID: "receiver-id",
				Receiver:           "receiver",
				ItemName:           "Sword",
				Quantity:           1,
			},
			setupMock: func(m *MockUserService) {
				m.On("GiveItem", mock.Anything, "twitch", "owner-id", "owner", "twitch", "receiver-id", "receiver", "Sword", 1).Return(errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockUserService{}
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
		setupMock      func(*MockEconomyService, *MockProgressionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: BuyItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   1,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureBuy).Return(true, nil)
				e.On("BuyItem", mock.Anything, "twitch", "test-id", "testuser", "Sword", 1).Return(1, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"items_bought":1}`,
		},
		{
			name: "Feature Locked",
			requestBody: BuyItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   1,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureBuy).Return(false, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Buy feature is not yet unlocked",
		},
		{
			name: "Feature Check Error",
			requestBody: BuyItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   1,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureBuy).Return(false, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to check feature availability",
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid json",
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureBuy).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name: "Missing PlatformID",
			requestBody: BuyItemRequest{
				Platform: "twitch",
				Username: "testuser",
				ItemName: "Sword",
				Quantity: 1,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureBuy).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Zero Quantity",
			requestBody: BuyItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Sword",
				Quantity:   0,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureBuy).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Service Error - Insufficient Money",
			requestBody: BuyItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "pooruser",
				ItemName:   "Sword",
				Quantity:   1,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureBuy).Return(true, nil)
				e.On("BuyItem", mock.Anything, "twitch", "test-id", "pooruser", "Sword", 1).
					Return(0, errors.New("insufficient money"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "insufficient money",
		},
		{
			name: "Service Error - Item Not Available",
			requestBody: BuyItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "RareItem",
				Quantity:   1,
			},
			setupMock: func(e *MockEconomyService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureBuy).Return(true, nil)
				e.On("BuyItem", mock.Anything, "twitch", "test-id", "testuser", "RareItem", 1).
					Return(0, errors.New("item not available for purchase"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "item not available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEco := &MockEconomyService{}
			mockProg := &MockProgressionService{}
			mockBus := &MockEventBus{}
			tt.setupMock(mockEco, mockProg)
			// Allow event publishing
			mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
				return evt.Type == "item.bought"
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
		setupMock      func(*MockUserService, *MockEventBus)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: UseItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Potion",
				Quantity:   1,
			},
			setupMock: func(u *MockUserService, e *MockEventBus) {
				u.On("UseItem", mock.Anything, "twitch", "test-id", "testuser", "Potion", 1, "").Return("Used potion", nil)
				// Expect both engagement and item.used events
				e.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
					return evt.Type == "engagement" || evt.Type == "item.used"
				})).Return(nil).Maybe()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Used potion"}`,
		},
		{
			name: "Service Error",
			requestBody: UseItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "Potion",
				Quantity:   1,
			},
			setupMock: func(u *MockUserService, e *MockEventBus) {
				u.On("UseItem", mock.Anything, "twitch", "test-id", "testuser", "Potion", 1, "").Return("", errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUser := &MockUserService{}
			mockBus := &MockEventBus{}
			tt.setupMock(mockUser, mockBus)

			handler := HandleUseItem(mockUser, mockBus)

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
		setupMock      func(*MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:       "Success",
			username:   "testuser",
			platform:   "discord",
			platformID: "test-platformid",
			setupMock: func(m *MockUserService) {
				items := []user.UserInventoryItem{
					{Name: "Sword", Quantity: 1},
				}
				m.On("GetInventory", mock.Anything, "discord", "test-platformid", "testuser").Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"name":"Sword"`,
		},
		{
			name:           "Missing Username",
			username:       "",
			platform:       "discord",
			platformID:     "", // Missing platformID
			setupMock:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing", // Will fail on platform_id
		},
		{
			name:       "Service Error with Platform Params", // Changed name for clarity
			username:   "testuser",
			platform:   "discord",
			platformID: "test-platformid",
			setupMock: func(m *MockUserService) {
				// Updated mock expectation to match platform and platformID
				m.On("GetInventory", mock.Anything, "discord", "test-platformid", "testuser").Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockUserService{}
			tt.setupMock(mockSvc)

			handler := HandleGetInventory(mockSvc)

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
			url := "/user/inventory"
			if len(params) > 0 {
				url += "?" + strings.Join(params, "&")
			}
			req := httptest.NewRequest("GET", url, nil)
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
