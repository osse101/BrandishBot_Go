package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/osse101/BrandishBot_Go/mocks"
	"github.com/stretchr/testify/assert"
)

func TestHandleReloadAliases(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockNamingResolver)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			setupMock: func(m *mocks.MockNamingResolver) {
				m.On("Reload").Return(nil)
				m.On("GetActiveTheme").Return("winter_event")
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "winter_event",
		},
		{
			name: "Failure - Reload Error",
			setupMock: func(m *mocks.MockNamingResolver) {
				m.On("Reload").Return(errors.New("failed to read file"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to reload configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockResolver := mocks.NewMockNamingResolver(t)
			tt.setupMock(mockResolver)

			handler := HandleReloadAliases(mockResolver)

			req := httptest.NewRequest("POST", "/admin/reload-aliases", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mockResolver.AssertExpectations(t)
		})
	}
}
