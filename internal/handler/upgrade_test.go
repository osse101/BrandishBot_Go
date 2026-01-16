package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
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
				p.On("IsFeatureUnlocked", mock.Anything, "feature_upgrade").Return(true, nil)
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
				p.On("IsFeatureUnlocked", mock.Anything, "feature_upgrade").Return(false, nil)
				// When locked, it tries to get required nodes
				// IMPORTANT: Must return []*domain.ProgressionNode (slice of pointers)
				p.On("GetRequiredNodes", mock.Anything, "feature_upgrade").Return([]*domain.ProgressionNode{}, nil)
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
				p.On("IsFeatureUnlocked", mock.Anything, "feature_upgrade").Return(true, nil)
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
				p.On("IsFeatureUnlocked", mock.Anything, "feature_upgrade").Return(true, nil)
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
				p.On("IsFeatureUnlocked", mock.Anything, "feature_upgrade").Return(true, nil)
				c.On("UpgradeItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.PublicNameJunkbox, 10).
					Return(&crafting.Result{
						ItemName:      domain.PublicNameLootbox,
						Quantity:      20, // Doubled
						IsMasterwork:  true,
						BonusQuantity: 10,
					}, nil)

				b.On("Publish", mock.Anything, mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"MASTERWORK! Critical success! You received 20x lootbox (Bonus: +10)","new_item":"lootbox","quantity_upgraded":20,"is_masterwork":true,"bonus_quantity":10}`,
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
