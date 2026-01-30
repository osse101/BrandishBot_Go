package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleUpgradeItem(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    CraftingActionRequest
		mockSetup      func(*mocks.MockCraftingService, *mocks.MockProgressionService, *mocks.MockEventBus)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: CraftingActionRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameJunkbox, // Assuming "junkbox" maps to Lootbox0
				Quantity:   2,
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
				c.On("UpgradeItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.PublicNameJunkbox, 2).
					Return(&crafting.Result{
						ItemName:      domain.PublicNameLootbox, // Result is Lootbox1
						Quantity:      2,
						IsMasterwork:  false,
						BonusQuantity: 0,
					}, nil)

				b.On("Publish", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
					// Add event matching logic if needed
					return true
				})).Return(nil)
				p.On("RecordEngagement", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Successfully upgraded to 2x lootbox","new_item":"lootbox","quantity_upgraded":2,"is_masterwork":false,"bonus_quantity":0}`,
		},
		{
			name: "Feature Locked",
			requestBody: CraftingActionRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameJunkbox,
				Quantity:   1,
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(false, nil)
				// When locked, it tries to get required nodes
				// IMPORTANT: Must return []*domain.ProgressionNode (slice of pointers)
				p.On("GetRequiredNodes", mock.Anything, progression.FeatureUpgrade).Return([]*domain.ProgressionNode{}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"Feature locked"}`,
		},
		{
			name: "Service Error",
			requestBody: CraftingActionRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameJunkbox,
				Quantity:   1,
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
				c.On("UpgradeItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.PublicNameJunkbox, 1).
					Return(nil, fmt.Errorf(ErrMsgGenericServerError)) // Return nil result on error
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
		{
			name: "Invalid Request",
			requestBody: CraftingActionRequest{
				Platform: "", // Missing platform
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Success Masterwork",
			requestBody: CraftingActionRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameJunkbox,
				Quantity:   10,
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
				c.On("UpgradeItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.PublicNameJunkbox, 10).
					Return(&crafting.Result{
						ItemName:      domain.PublicNameLootbox,
						Quantity:      20, // Doubled
						IsMasterwork:  true,
						BonusQuantity: 10,
					}, nil)

				b.On("Publish", mock.Anything, mock.Anything).Return(nil)
				p.On("RecordEngagement", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"MASTERWORK! Critical success! You received 20x lootbox (Bonus: +10)","new_item":"lootbox","quantity_upgraded":20,"is_masterwork":true,"bonus_quantity":10}`,
		},
		{
			name: "Boundary Quantity Zero",
			requestBody: CraftingActionRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameJunkbox,
				Quantity:   0,
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "quantity",
		},
		{
			name: "Boundary Quantity Max",
			requestBody: CraftingActionRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameJunkbox,
				Quantity:   10000,
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
				c.On("UpgradeItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.PublicNameJunkbox, 10000).
					Return(&crafting.Result{
						ItemName:      domain.PublicNameLootbox,
						Quantity:      10000,
						IsMasterwork:  false,
						BonusQuantity: 0,
					}, nil)

				b.On("Publish", mock.Anything, mock.Anything).Return(nil)
				p.On("RecordEngagement", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Successfully upgraded to 10000x lootbox","new_item":"lootbox","quantity_upgraded":10000,"is_masterwork":false,"bonus_quantity":0}`,
		},
		{
			name: "Boundary Quantity Over Max",
			requestBody: CraftingActionRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameJunkbox,
				Quantity:   10001,
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "quantity",
		},
		{
			name: "Edge Case Username Too Long",
			requestBody: CraftingActionRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   string(make([]byte, 101)),
				Item:       domain.PublicNameJunkbox,
				Quantity:   1,
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "username",
		},
		{
			name: "Edge Case Item Name Too Long",
			requestBody: CraftingActionRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       string(make([]byte, 101)),
				Quantity:   1,
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureUpgrade).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "item",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockCrafting := new(mocks.MockCraftingService)
			mockProgression := new(mocks.MockProgressionService)
			mockBus := new(mocks.MockEventBus)

			tc.mockSetup(mockCrafting, mockProgression, mockBus)

			handler := HandleUpgradeItem(mockCrafting, mockProgression, mockBus)

			body, _ := json.Marshal(tc.requestBody)
			req, _ := http.NewRequest("POST", "/user/item/upgrade", bytes.NewBuffer(body))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			if tc.expectedBody != "" {
				if tc.expectedStatus == http.StatusOK || tc.expectedStatus == http.StatusForbidden {
					assert.JSONEq(t, tc.expectedBody, rr.Body.String())
				} else {
					assert.Contains(t, rr.Body.String(), tc.expectedBody)
				}
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
		queryParams    map[string]string
		mockSetup      func(*mocks.MockCraftingService, *mocks.MockRepositoryUser)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Get All Recipes",
			queryParams: map[string]string{},
			mockSetup: func(c *mocks.MockCraftingService, u *mocks.MockRepositoryUser) {
				c.On("GetAllRecipes", mock.Anything).Return([]repository.RecipeListItem{
					{ItemID: 1, ItemName: domain.PublicNameLootbox, Description: "A lootbox"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   fmt.Sprintf(`{"recipes":[{"item_id":1,"item_name":"%s","description":"A lootbox"}]}`, domain.PublicNameLootbox),
		},
		{
			name: "Get Recipe By Item - Happy Path",
			queryParams: map[string]string{
				"item": domain.PublicNameLootbox,
			},
			mockSetup: func(c *mocks.MockCraftingService, u *mocks.MockRepositoryUser) {
				c.On("GetRecipe", mock.Anything, domain.PublicNameLootbox, "", "", "").Return(&crafting.RecipeInfo{
					ItemName: domain.PublicNameLootbox,
					Locked:   false,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   fmt.Sprintf(`{"item_name":"%s"}`, domain.PublicNameLootbox),
		},
		{
			name: "Get Recipe By Item - Not Found",
			queryParams: map[string]string{
				"item": "invalid",
			},
			mockSetup: func(c *mocks.MockCraftingService, u *mocks.MockRepositoryUser) {
				c.On("GetRecipe", mock.Anything, "invalid", "", "", "").Return(nil, fmt.Errorf("item not found: %w", domain.ErrItemNotFound))
			},
			expectedStatus: http.StatusBadRequest, // mapServiceErrorToUserMessage maps ItemNotFound to BadRequest
			expectedBody:   "Item not found",
		},
		{
			name: "Get Unlocked Recipes - User Only (Missing Platform)",
			queryParams: map[string]string{
				"user": "testuser",
			},
			mockSetup: func(c *mocks.MockCraftingService, u *mocks.MockRepositoryUser) {
				// Should fail before service call because platform param is missing for user queries
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing platform",
		},
		{
			name: "Get Unlocked Recipes - User and Platform (Self Mode)",
			queryParams: map[string]string{
				"user":        "testuser",
				"platform":    domain.PlatformTwitch,
				"platform_id": "test-id",
			},
			mockSetup: func(c *mocks.MockCraftingService, u *mocks.MockRepositoryUser) {
				c.On("GetUnlockedRecipes", mock.Anything, domain.PlatformTwitch, "test-id", "testuser").Return([]repository.UnlockedRecipeInfo{
					{ItemID: 1, ItemName: domain.PublicNameLootbox},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   fmt.Sprintf(`{"recipes":[{"item_name":"%s","item_id":1}]}`, domain.PublicNameLootbox),
		},
		{
			name: "Get Unlocked Recipes - Target Mode (Resolve Username)",
			queryParams: map[string]string{
				"user":     "otheruser",
				"platform": domain.PlatformTwitch,
			},
			mockSetup: func(c *mocks.MockCraftingService, u *mocks.MockRepositoryUser) {
				u.On("GetUserByPlatformUsername", mock.Anything, domain.PlatformTwitch, "otheruser").Return(&domain.User{
					ID:       "user-123",
					TwitchID: "other-id",
				}, nil)
				// The handler logic calls getPlatformID(user, platform)
				// Then it calls service with resolved platformID
				c.On("GetUnlockedRecipes", mock.Anything, domain.PlatformTwitch, "other-id", "otheruser").Return([]repository.UnlockedRecipeInfo{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"recipes":[]}`,
		},
		{
			name: "Get Unlocked Recipes - Target Mode - User Not Found",
			queryParams: map[string]string{
				"user":     "unknown",
				"platform": domain.PlatformTwitch,
			},
			mockSetup: func(c *mocks.MockCraftingService, u *mocks.MockRepositoryUser) {
				u.On("GetUserByPlatformUsername", mock.Anything, domain.PlatformTwitch, "unknown").Return(nil, domain.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "User not found",
		},
		{
			name: "Service Error - GetAllRecipes",
			queryParams: map[string]string{},
			mockSetup: func(c *mocks.MockCraftingService, u *mocks.MockRepositoryUser) {
				c.On("GetAllRecipes", mock.Anything).Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "db error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockCrafting := new(mocks.MockCraftingService)
			mockUserRepo := new(mocks.MockRepositoryUser)

			tc.mockSetup(mockCrafting, mockUserRepo)

			handler := NewCraftingHandler(mockCrafting, mockUserRepo)

			req, _ := http.NewRequest("GET", "/recipes", nil)
			q := req.URL.Query()
			for k, v := range tc.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			rr := httptest.NewRecorder()

			handler.HandleGetRecipes().ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			if tc.expectedBody != "" {
				if tc.expectedStatus == http.StatusOK {
					assert.JSONEq(t, tc.expectedBody, rr.Body.String())
				} else {
					assert.Contains(t, rr.Body.String(), tc.expectedBody)
				}
			}

			mockCrafting.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t)
		})
	}
}
