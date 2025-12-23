package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleUpgradeItem(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockCraftingService, *mocks.MockProgressionService, *mocks.MockEventBus)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: UpgradeItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox0",
				Quantity:   2,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
				p.On("AddContribution", mock.Anything, mock.Anything).Return(nil)
				c.On("UpgradeItem", mock.Anything, "twitch", "test-id", "testuser", "lootbox0", 2).
					Return("lootbox1", 2, nil)
				e.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
					return evt.Type == "item.upgraded"
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"new_item":"lootbox1"`,
		},
		{
			name: "Feature Locked",
			requestBody: UpgradeItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox0",
				Quantity:   1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(false, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "not yet unlocked",
		},
		{
			name: "Feature Check Error",
			requestBody: UpgradeItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox0",
				Quantity:   1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).
					Return(false, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to check feature availability",
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid json",
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name: "Missing Platform",
			requestBody: UpgradeItemRequest{
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox0",
				Quantity:   1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Zero Quantity",
			requestBody: UpgradeItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox0",
				Quantity:   0,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Service Error - Insufficient Materials",
			requestBody: UpgradeItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox0",
				Quantity:   100,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
				c.On("UpgradeItem", mock.Anything, "twitch", "test-id", "testuser", "lootbox0", 100).
					Return("", 0, errors.New("insufficient materials"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "insufficient materials",
		},
		{
			name: "Event Publish Failure - Still Returns Success",
			requestBody: UpgradeItemRequest{
				Platform:   "twitch",
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox0",
				Quantity:   1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
				p.On("AddContribution", mock.Anything, mock.Anything).Return(nil)
				c.On("UpgradeItem", mock.Anything, "twitch", "test-id", "testuser", "lootbox0", 1).
					Return("lootbox1", 1, nil)
				e.On("Publish", mock.Anything, mock.Anything).Return(errors.New("event bus error"))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"quantity_upgraded":1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCrafting := mocks.NewMockCraftingService(t)
			mockProgression := mocks.NewMockProgressionService(t)
			mockBus := mocks.NewMockEventBus(t)
			tt.setupMock(mockCrafting, mockProgression, mockBus)

			handler := HandleUpgradeItem(mockCrafting, mockProgression, mockBus)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest("POST", "/user/item/upgrade", bytes.NewBuffer(body))
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

func TestHandleGetRecipes(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*mocks.MockCraftingService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "Get Recipe - Without User",
			queryParams: "?item=lootbox1",
			setupMock: func(c *mocks.MockCraftingService) {
				c.On("GetRecipe", mock.Anything, "lootbox1", "", "", "").
					Return(&crafting.RecipeInfo{
						ItemName: "lootbox1",
						Locked:   false,
						BaseCost: []domain.RecipeCost{{ItemID: 1, Quantity: 1}},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"item_name":"lootbox1"`,
		},
		{
			name:        "Get Recipe - With User",
			queryParams: "?item=lootbox1&user=testuser&platform=twitch&platform_id=test-id",
			setupMock: func(c *mocks.MockCraftingService) {
				c.On("GetRecipe", mock.Anything, "lootbox1", "twitch", "test-id", "testuser").
					Return(&crafting.RecipeInfo{
						ItemName: "lootbox1",
						Locked:   true,
						BaseCost: []domain.RecipeCost{{ItemID: 1, Quantity: 1}},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"locked":true`,
		},
		{
			name:        "Get Unlocked Recipes - Success",
			queryParams: "?user=testuser&platform=twitch&platform_id=test-id",
			setupMock: func(c *mocks.MockCraftingService) {
				c.On("GetUnlockedRecipes", mock.Anything, "twitch", "test-id", "testuser").
					Return([]crafting.UnlockedRecipeInfo{
						{ItemName: "lootbox1", ItemID: 1},
						{ItemName: "lootbox2", ItemID: 2},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"recipes"`,
		},
		{
			name:           "Get Unlocked Recipes - Missing Platform",
			queryParams:    "?user=testuser&platform_id=test-id",
			setupMock:      func(c *mocks.MockCraftingService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing platform",
		},
		{
			name:           "Get Unlocked Recipes - Missing PlatformID",
			queryParams:    "?user=testuser&platform=twitch",
			setupMock:      func(c *mocks.MockCraftingService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing platform",
		},
		{
			name:           "No Parameters Provided",
			queryParams:    "",
			setupMock:      func(c *mocks.MockCraftingService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Must provide either",
		},
		{
			name:        "Service Error - Get Recipe",
			queryParams: "?item=unknown",
			setupMock: func(c *mocks.MockCraftingService) {
				c.On("GetRecipe", mock.Anything, "unknown", "", "", "").
					Return(nil, errors.New("recipe not found"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "recipe not found",
		},
		{
			name:        "Service Error - Get Unlocked Recipes",
			queryParams: "?user=testuser&platform=twitch&platform_id=test-id",
			setupMock: func(c *mocks.MockCraftingService) {
				c.On("GetUnlockedRecipes", mock.Anything, "twitch", "test-id", "testuser").
					Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCrafting := mocks.NewMockCraftingService(t)
			tt.setupMock(mockCrafting)

			handler := HandleGetRecipes(mockCrafting)

			req := httptest.NewRequest("GET", "/recipes"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			mockCrafting.AssertExpectations(t)
		})
	}
}
