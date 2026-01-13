package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/user"
	"github.com/osse101/BrandishBot_Go/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleGetCacheStats(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockUserService)
		expectedStatus int
		expectedStats  user.CacheStats
	}{
		{
			name: "Success",
			setupMock: func(m *mocks.MockUserService) {
				stats := user.CacheStats{
					Hits:      100,
					Misses:    50,
					Evictions: 10,
					Size:      500,
				}
				m.On("GetCacheStats").Return(stats)
			},
			expectedStatus: http.StatusOK,
			expectedStats: user.CacheStats{
				Hits:      100,
				Misses:    50,
				Evictions: 10,
				Size:      500,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserService := mocks.NewMockUserService(t)
			tt.setupMock(mockUserService)

			handler := NewAdminCacheHandler(mockUserService)

			req := httptest.NewRequest("GET", "/api/v1/admin/cache/stats", nil)
			w := httptest.NewRecorder()

			handler.HandleGetCacheStats(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response user.CacheStats
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStats.Hits, response.Hits)
			assert.Equal(t, tt.expectedStats.Misses, response.Misses)
			assert.Equal(t, tt.expectedStats.Evictions, response.Evictions)
			assert.Equal(t, tt.expectedStats.Size, response.Size)

			mockUserService.AssertExpectations(t)
		})
	}
}
