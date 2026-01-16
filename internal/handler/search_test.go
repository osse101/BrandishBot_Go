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
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleSearch(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockUserService, *mocks.MockProgressionService, *mocks.MockEventBus)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: SearchRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
			},
			setupMock: func(u *mocks.MockUserService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSearch).Return(true, nil)

				u.On("HandleSearch", mock.Anything, domain.PlatformTwitch, "test-id", "testuser").Return("Found a sword!", nil)
				// Expect both engagement and search.performed events
				e.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
					return evt.Type == "engagement" || evt.Type == "search.performed"
				})).Return(nil).Maybe()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"message":"Found a sword!"`,
		},
		{
			name: "Feature Locked",
			requestBody: SearchRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
			},
			setupMock: func(u *mocks.MockUserService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSearch).Return(false, nil)
				p.On("GetRequiredNodes", mock.Anything, progression.FeatureSearch).Return([]*domain.ProgressionNode{
					{DisplayName: "Progression System"},
				}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "LOCKED_NODES: Progression System",
		},
		{
			name: "Service Error",
			requestBody: SearchRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "test-id",
				Username:   "testuser",
			},
			setupMock: func(u *mocks.MockUserService, p *mocks.MockProgressionService, e *mocks.MockEventBus) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSearch).Return(true, nil)
				u.On("HandleSearch", mock.Anything, domain.PlatformTwitch, "test-id", "testuser").Return("", errors.New(ErrMsgGenericServerError))
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
			tt.setupMock(mockUser, mockProg, mockBus)

			handler := HandleSearch(mockUser, mockProg, mockBus)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/search", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mockUser.AssertExpectations(t)
			mockProg.AssertExpectations(t)
			mockBus.AssertExpectations(t)
		})
	}
}
