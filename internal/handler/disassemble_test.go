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

func TestHandleDisassembleItem(t *testing.T) {
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
				Item:       "lootbox_tier1",
				Quantity:   2,
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_disassemble").Return(true, nil)
				c.On("DisassembleItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", "lootbox_tier1", 2).
					Return(&crafting.DisassembleResult{
						Outputs:           map[string]int{"lootbox_tier0": 4},
						QuantityProcessed: 2,
						IsPerfectSalvage:  false,
						Multiplier:        1.0,
					}, nil)

				b.On("Publish", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
					return true
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Disassembled 2 items into: 4x lootbox_tier0","outputs":{"lootbox_tier0":4},"quantity_processed":2,"is_perfect_salvage":false,"multiplier":1}`,
		},
		{
			name: "Feature Locked",
			requestBody: CraftingActionRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox_tier1",
				Quantity:   1,
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_disassemble").Return(false, nil)
				// When locked, it tries to get required nodes to show helpful message
				// IMPORTANT: Must return []*domain.ProgressionNode (slice of pointers)
				p.On("GetRequiredNodes", mock.Anything, "feature_disassemble").Return([]*domain.ProgressionNode{}, nil)
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
				Item:       "lootbox_tier1",
				Quantity:   1,
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_disassemble").Return(true, nil)
				c.On("DisassembleItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", "lootbox_tier1", 1).
					Return(nil, fmt.Errorf(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
		{
			name: "Success Perfect Salvage",
			requestBody: CraftingActionRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "lootbox_tier1",
				Quantity:   10,
			},
			mockSetup: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, b *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_disassemble").Return(true, nil)
				c.On("DisassembleItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", "lootbox_tier1", 10).
					Return(&crafting.DisassembleResult{
						Outputs:           map[string]int{"lootbox_tier0": 30}, // 20 * 1.5
						QuantityProcessed: 10,
						IsPerfectSalvage:  true,
						Multiplier:        1.5,
					}, nil)

				b.On("Publish", mock.Anything, mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"PERFECT SALVAGE! You efficiently recovered more materials! (+50% Bonus): 30x lootbox_tier0","outputs":{"lootbox_tier0":30},"quantity_processed":10,"is_perfect_salvage":true,"multiplier":1.5}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockCrafting := new(mocks.MockCraftingService)
			mockProgression := new(mocks.MockProgressionService)
			mockBus := new(mocks.MockEventBus)

			tc.mockSetup(mockCrafting, mockProgression, mockBus)

			handler := HandleDisassembleItem(mockCrafting, mockProgression, mockBus)

			body, _ := json.Marshal(tc.requestBody)
			req, _ := http.NewRequest("POST", "/user/item/disassemble", bytes.NewBuffer(body))
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
