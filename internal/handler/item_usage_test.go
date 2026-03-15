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
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleUseItem(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockUserService, *mocks.MockProgressionService, *mocks.MockEventBus)
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
			setupMock: func(u *mocks.MockUserService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				// Resolve internal name
				u.On("GetItemByName", mock.Anything, domain.PublicNameMissile).Return(&domain.Item{InternalName: domain.ItemMissile}, nil)
				// Unlock the feature
				p.On("IsFeatureUnlocked", mock.Anything, "weapon_missile").Return(true, nil)
				// Mock should return what the real blaster handler would return
				u.On("UseItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.PublicNameMissile, 1, "").
					Return("testuser has BLASTED target 1 times! They are timed out for 1m0s.", nil)
				u.On("GetUserIDByPlatformID", mock.Anything, domain.PlatformTwitch, "test-id").Return("", nil)
				// Expect ONLY engagement event (item.used is now service-side and not using this mock bus in this test setup)
				// Wait, the handler STILL publishes engagement if points > 0
				e.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
					return evt.Type == "engagement"
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
			setupMock: func(u *mocks.MockUserService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				u.On("GetItemByName", mock.Anything, domain.PublicNameMissile).Return(&domain.Item{InternalName: domain.ItemMissile}, nil)
				p.On("IsFeatureUnlocked", mock.Anything, "weapon_missile").Return(true, nil)
				u.On("UseItem", mock.Anything, domain.PlatformTwitch, "test-id", "testuser", domain.PublicNameMissile, 1, "").Return("", errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
		{
			name: "Progression Locked",
			requestBody: UseItemRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
				ItemName:   "filter", // Public name
				Quantity:   1,
			},
			setupMock: func(u *mocks.MockUserService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				// Resolve internal name
				u.On("GetItemByName", mock.Anything, "filter").Return(&domain.Item{InternalName: domain.ItemVideoFilter}, nil)
				// Lock the feature
				p.On("IsFeatureUnlocked", mock.Anything, "item_video_filter").Return(false, nil)
				// Mock required nodes for error message
				p.On("GetRequiredNodes", mock.Anything, "item_video_filter").Return([]*domain.ProgressionNode{}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Feature locked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUser := mocks.NewMockUserService(t)
			mockProg := mocks.NewMockProgressionService(t)
			mockBus := mocks.NewMockEventBus(t)
			mockProg.On("RecordEngagement", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			tt.setupMock(mockUser, mockProg, mockBus)

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
