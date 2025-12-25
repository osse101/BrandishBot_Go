package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)


func TestHandleDisassembleItem(t *testing.T) {
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
			requestBody: DisassembleItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameLootbox,
				Quantity:   2,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
				c.On("DisassembleItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.PublicNameLootbox, 2).
					Return(map[string]int{domain.PublicNameJunkbox: 2}, 2, nil)
				p.On("AddContribution", mock.Anything, mock.Anything).Return(nil)
				e.On("Publish", mock.Anything, mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"quantity_processed":2`,
		},
		{
			name: "Feature Locked",
			requestBody: DisassembleItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameLootbox,
				Quantity:   1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(false, nil)
				p.On("GetRequiredNodes", mock.Anything, progression.FeatureDisassemble).Return([]*domain.ProgressionNode{
					{DisplayName: "Disassemble System"},
				}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "LOCKED_NODES: Disassemble System",
		},
		{
			name: "Feature Check Error",
			requestBody: DisassembleItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameLootbox,
				Quantity:   1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).
					Return(false, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to check feature availability",
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid json",
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
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
				Item:       domain.PublicNameLootbox,
				Quantity:   1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Missing PlatformID",
			requestBody: DisassembleItemRequest{
				Platform: domain.PlatformTwitch,
				Username: "testuser",
				Item:     domain.PublicNameLootbox,
				Quantity: 1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Missing Username",
			requestBody: DisassembleItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Item:       domain.PublicNameLootbox,
				Quantity:   1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Missing Item",
			requestBody: DisassembleItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Quantity:   1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Zero Quantity",
			requestBody: DisassembleItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameLootbox,
				Quantity:   0,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Negative Quantity",
			requestBody: DisassembleItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameLootbox,
				Quantity:   -1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
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
				Item:       domain.PublicNameLootbox,
				Quantity:   1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Service Error - Item Not Found",
			requestBody: DisassembleItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       "unknown-item",
				Quantity:   1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
				c.On("DisassembleItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", "unknown-item", 1).
					Return(nil, 0, errors.New("item not found"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "item not found",
		},
		{
			name: "Service Error - Insufficient Items",
			requestBody: DisassembleItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameLootbox,
				Quantity:   100,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
				c.On("DisassembleItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.PublicNameLootbox, 100).
					Return(nil, 0, errors.New("insufficient items"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "insufficient items",
		},
		{
			name: "Event Publish Failure - Still Returns Success",
			requestBody: DisassembleItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				Item:       domain.PublicNameLootbox,
				Quantity:   1,
			},
			setupMock: func(c *mocks.MockCraftingService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureDisassemble).Return(true, nil)
				c.On("DisassembleItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.PublicNameLootbox, 1).
					Return(map[string]int{domain.PublicNameJunkbox: 1}, 1, nil)
				p.On("AddContribution", mock.Anything, mock.Anything).Return(nil)
				e.On("Publish", mock.Anything, mock.Anything).Return(errors.New("event bus error"))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"quantity_processed":1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCrafting := mocks.NewMockCraftingService(t)
			mockProgression := mocks.NewMockProgressionService(t)
			mockBus := mocks.NewMockEventBus(t)
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
