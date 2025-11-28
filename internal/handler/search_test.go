package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleSearch(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockUserService, *MockProgressionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: SearchRequest{
				Username: "testuser",
				Platform: "twitch",
			},
			setupMock: func(u *MockUserService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSearch).Return(true, nil)
				u.On("HandleSearch", mock.Anything, "testuser", "twitch").Return("Found a sword!", nil)
				// Allow RecordEngagement (async)
				p.On("RecordEngagement", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"message":"Found a sword!"`,
		},
		{
			name: "Feature Locked",
			requestBody: SearchRequest{
				Username: "testuser",
				Platform: "twitch",
			},
			setupMock: func(u *MockUserService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSearch).Return(false, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Search feature is not yet unlocked",
		},
		{
			name: "Service Error",
			requestBody: SearchRequest{
				Username: "testuser",
				Platform: "twitch",
			},
			setupMock: func(u *MockUserService, p *MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, progression.FeatureSearch).Return(true, nil)
				u.On("HandleSearch", mock.Anything, "testuser", "twitch").Return("", errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to perform search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUser := &MockUserService{}
			mockProg := &MockProgressionService{}
			tt.setupMock(mockUser, mockProg)

			handler := HandleSearch(mockUser, mockProg)

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
		})
	}
}
