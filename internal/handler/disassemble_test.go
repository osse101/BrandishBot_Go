package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCraftingService mocks the crafting.Service interface
type MockCraftingService struct {
	mock.Mock
}

func (m *MockCraftingService) DisassembleItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (map[string]int, int, error) {
	args := m.Called(ctx, platform, platformID, username, itemName, quantity)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).(map[string]int), args.Int(1), args.Error(2)
}

func (m *MockCraftingService) UpgradeItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (string, int, error) {
	args := m.Called(ctx, platform, platformID, username, itemName, quantity)
	return args.String(0), args.Int(1), args.Error(2)
}

func (m *MockCraftingService) GetRecipe(ctx context.Context, itemName, platform, platformID, username string) (*crafting.RecipeInfo, error) {
	args := m.Called(ctx, itemName, platform, platformID, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*crafting.RecipeInfo), args.Error(1)
}

func (m *MockCraftingService) GetUnlockedRecipes(ctx context.Context, platform, platformID, username string) ([]crafting.UnlockedRecipeInfo, error) {
	args := m.Called(ctx, platform, platformID, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]crafting.UnlockedRecipeInfo), args.Error(1)
}

func TestHandleDisassembleItem(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockCraftingService, *MockProgressionService, *MockEventBus)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: DisassembleItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox1",
				Quantity:   2,
			},
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
				c.On("DisassembleItem", mock.Anything, "twitch", "test-id", "testuser", "lootbox1", 2).
					Return(map[string]int{"lootbox0": 2}, 2, nil)
				e.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
					return evt.Type == "item.disassembled"
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"quantity_processed":2`,
		},
		{
			name: "Feature Locked",
			requestBody: DisassembleItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox1",
				Quantity:   1,
			},
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(false, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "not yet unlocked",
		},
		{
			name: "Feature Check Error",
			requestBody: DisassembleItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox1",
				Quantity:   1,
			},
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).
					Return(false, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to check feature availability",
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid json",
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name: "Missing Platform",
			requestBody: DisassembleItemRequest{
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox1",
				Quantity:   1,
			},
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Missing PlatformID",
			requestBody: DisassembleItemRequest{
				Platform: "twitch",
				Username: "testuser",
				Item:     "lootbox1",
				Quantity: 1,
			},
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Missing Username",
			requestBody: DisassembleItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Item:       "lootbox1",
				Quantity:   1,
			},
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Missing Item",
			requestBody: DisassembleItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Quantity:   1,
			},
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Zero Quantity",
			requestBody: DisassembleItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox1",
				Quantity:   0,
			},
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Negative Quantity",
			requestBody: DisassembleItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox1",
				Quantity:   -1,
			},
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Invalid Platform",
			requestBody: DisassembleItemRequest{
				Platform:   "invalid-platform",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox1",
				Quantity:   1,
			},
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Service Error - Item Not Found",
			requestBody: DisassembleItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "unknown-item",
				Quantity:   1,
			},
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
				c.On("DisassembleItem", mock.Anything, "twitch", "test-id", "testuser", "unknown-item", 1).
					Return(nil, 0, errors.New("item not found"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "item not found",
		},
		{
			name: "Service Error - Insufficient Items",
			requestBody: DisassembleItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox1",
				Quantity:   100,
			},
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
				c.On("DisassembleItem", mock.Anything, "twitch", "test-id", "testuser", "lootbox1", 100).
					Return(nil, 0, errors.New("insufficient items"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "insufficient items",
		},
		{
			name: "Event Publish Failure - Still Returns Success",
			requestBody: DisassembleItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox1",
				Quantity:   1,
			},
			setupMock: func(c *MockCraftingService, p *MockProgressionService, e *MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
				c.On("DisassembleItem", mock.Anything, "twitch", "test-id", "testuser", "lootbox1", 1).
					Return(map[string]int{"lootbox0": 1}, 1, nil)
				e.On("Publish", mock.Anything, mock.Anything).Return(errors.New("event bus error"))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"quantity_processed":1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCrafting := &MockCraftingService{}
			mockProgression := &MockProgressionService{}
			mockBus := &MockEventBus{}
			tt.setupMock(mockCrafting, mockProgression, mockBus)

			handler := HandleDisassembleItem(mockCrafting, mockProgression, mockBus)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest("POST", "/user/item/disassemble", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			mockCrafting.AssertExpectations(t)
			mockProgression.AssertExpectations(t)
			mockBus.AssertExpectations(t)
		})
	}
}
