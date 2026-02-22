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
		{
			name: "Invalid Platform",
			requestBody: AddItemByUsernameRequest{
				Platform: "invalid_platform",
				Username: "testuser",
				ItemName: domain.ItemBlaster,
				Quantity: 1,
			},
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Negative Quantity",
			requestBody: AddItemByUsernameRequest{
				Platform: domain.PlatformTwitch,
				Username: "testuser",
				ItemName: domain.ItemBlaster,
				Quantity: -1,
			},
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Empty Item Name",
			requestBody: AddItemByUsernameRequest{
				Platform: domain.PlatformTwitch,
				Username: "testuser",
				ItemName: "",
				Quantity: 1,
			},
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
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
		{
			name: "Invalid Platform",
			requestBody: RemoveItemByUsernameRequest{
				Platform: "invalid_platform",
				Username: "testuser",
				ItemName: domain.ItemBlaster,
				Quantity: 1,
			},
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Negative Quantity",
			requestBody: RemoveItemByUsernameRequest{
				Platform: domain.PlatformTwitch,
				Username: "testuser",
				ItemName: domain.ItemBlaster,
				Quantity: -1,
			},
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
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
					{InternalName: domain.ItemBlaster, PublicName: "missile", Quantity: 1, QualityLevel: "COMMON"},
				}
				m.On("GetInventory", mock.Anything, domain.PlatformDiscord, "test-platformid", "testuser", "").Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"items":[{"item_name":"weapon_blaster","public_name":"missile","quantity":1,"quality_level":"COMMON"}]`,
		},
		{
			name:       "Success with Filter",
			username:   "testuser",
			platform:   domain.PlatformDiscord,
			platformID: "test-platformid",
			filter:     domain.FilterTypeUpgrade,
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				items := []user.InventoryItem{
					{InternalName: domain.ItemLootbox0, PublicName: "junkbox", Quantity: 1, QualityLevel: "COMMON"},
				}
				p.On("IsFeatureUnlocked", mock.Anything, "feature_filter_upgrade").Return(true, nil)
				m.On("GetInventory", mock.Anything, domain.PlatformDiscord, "test-platformid", "testuser", domain.FilterTypeUpgrade).Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"items":[{"item_name":"lootbox_tier0","public_name":"junkbox","quantity":1,"quality_level":"COMMON"}]`,
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
			name:           "Missing Platform",
			username:       "testuser",
			platform:       "",
			platformID:     "test-platformid",
			filter:         "",
			setupMock:      func(m *mocks.MockUserService, p *mocks.MockProgressionService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing platform query parameter",
		},
		{
			name:           "Missing PlatformID",
			username:       "testuser",
			platform:       domain.PlatformDiscord,
			platformID:     "",
			filter:         "",
			setupMock:      func(m *mocks.MockUserService, p *mocks.MockProgressionService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing platform_id query parameter",
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
					{InternalName: domain.ItemLootbox1, PublicName: "lootbox", Quantity: 5, QualityLevel: "COMMON"},
				}
				p.On("IsFeatureUnlocked", mock.Anything, "feature_filter_sellable").Return(true, nil)
				m.On("GetInventory", mock.Anything, domain.PlatformDiscord, "test-platformid", "testuser", domain.FilterTypeSellable).Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"items":[{"item_name":"lootbox_tier1","public_name":"lootbox","quantity":5,"quality_level":"COMMON"}]`,
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
					{InternalName: domain.ItemLootbox0, PublicName: "junkbox", Quantity: 3, QualityLevel: "COMMON"},
				}
				p.On("IsFeatureUnlocked", mock.Anything, "feature_filter_consumable").Return(true, nil)
				m.On("GetInventory", mock.Anything, domain.PlatformDiscord, "test-platformid", "testuser", domain.FilterTypeConsumable).Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"items":[{"item_name":"lootbox_tier0","public_name":"junkbox","quantity":3,"quality_level":"COMMON"}]`,
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

func TestHandleGetInventoryByUsername(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		username       string
		platform       string
		filter         string
		setupMock      func(*mocks.MockUserService, *mocks.MockProgressionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:     "Success",
			username: "testuser",
			platform: domain.PlatformDiscord,
			filter:   "",
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				items := []user.InventoryItem{
					{InternalName: domain.ItemBlaster, PublicName: "missile", Quantity: 1, QualityLevel: "COMMON"},
				}
				m.On("GetInventoryByUsername", mock.Anything, domain.PlatformDiscord, "testuser", "").Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"items":[{"item_name":"weapon_blaster","public_name":"missile","quantity":1,"quality_level":"COMMON"}]`,
		},
		{
			name:     "Success with Filter",
			username: "testuser",
			platform: domain.PlatformDiscord,
			filter:   domain.FilterTypeUpgrade,
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				items := []user.InventoryItem{
					{InternalName: domain.ItemLootbox0, PublicName: "junkbox", Quantity: 1, QualityLevel: "COMMON"},
				}
				p.On("IsFeatureUnlocked", mock.Anything, "feature_filter_upgrade").Return(true, nil)
				m.On("GetInventoryByUsername", mock.Anything, domain.PlatformDiscord, "testuser", domain.FilterTypeUpgrade).Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"items":[{"item_name":"lootbox_tier0","public_name":"junkbox","quantity":1,"quality_level":"COMMON"}]`,
		},
		{
			name:     "Locked Filter",
			username: "testuser",
			platform: domain.PlatformDiscord,
			filter:   domain.FilterTypeUpgrade,
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_filter_upgrade").Return(false, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Filter 'upgrade' is locked",
		},
		{
			name:     "Invalid Filter",
			username: "testuser",
			platform: domain.PlatformDiscord,
			filter:   "invalid_filter",
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				// No interactions expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid filter type",
		},
		{
			name:           "Missing Username",
			username:       "",
			platform:       domain.PlatformDiscord,
			filter:         "",
			setupMock:      func(m *mocks.MockUserService, p *mocks.MockProgressionService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing username query parameter",
		},
		{
			name:     "Service Error",
			username: "testuser",
			platform: domain.PlatformDiscord,
			filter:   "",
			setupMock: func(m *mocks.MockUserService, p *mocks.MockProgressionService) {
				m.On("GetInventoryByUsername", mock.Anything, domain.PlatformDiscord, "testuser", "").Return(nil, errors.New(ErrMsgGenericServerError))
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

			handler := HandleGetInventoryByUsername(mockUser, mockProg)

			// Build URL with query parameters
			params := []string{}
			if tt.platform != "" {
				params = append(params, "platform="+tt.platform)
			}
			if tt.username != "" {
				params = append(params, "username="+tt.username)
			}
			if tt.filter != "" {
				params = append(params, "filter="+tt.filter)
			}
			url := "/user/inventory-by-username"
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
