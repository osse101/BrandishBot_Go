package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandleGetPrices(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockEconomyService)
		expectedStatus int
		verifyBody     func(*testing.T, string)
	}{
		{
			name: "Success",
			setupMock: func(m *mocks.MockEconomyService) {
				items := []domain.Item{{ID: 1, InternalName: "Item1", BaseValue: 10}}
				m.On("GetSellablePrices", mock.Anything).Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body string) {
				var items []domain.Item
				err := json.Unmarshal([]byte(body), &items)
				require.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, "Item1", items[0].InternalName)
			},
		},
		{
			name: "Service Error",
			setupMock: func(m *mocks.MockEconomyService) {
				m.On("GetSellablePrices", mock.Anything).Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			verifyBody:     func(t *testing.T, body string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockEconomyService(t)
			tt.setupMock(mockSvc)

			handler := HandleGetPrices(mockSvc)

			req := httptest.NewRequest("GET", "/prices", nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.verifyBody(t, rec.Body.String())
		})
	}
}

func TestHandleGetBuyPrices(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockEconomyService)
		expectedStatus int
		verifyBody     func(*testing.T, string)
	}{
		{
			name: "Success",
			setupMock: func(m *mocks.MockEconomyService) {
				items := []domain.Item{{ID: 2, InternalName: "Item2", BaseValue: 20}}
				m.On("GetBuyablePrices", mock.Anything).Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body string) {
				var items []domain.Item
				err := json.Unmarshal([]byte(body), &items)
				require.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, "Item2", items[0].InternalName)
			},
		},
		{
			name: "Service Error",
			setupMock: func(m *mocks.MockEconomyService) {
				m.On("GetBuyablePrices", mock.Anything).Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			verifyBody:     func(t *testing.T, body string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockEconomyService(t)
			tt.setupMock(mockSvc)

			handler := HandleGetBuyPrices(mockSvc)

			req := httptest.NewRequest("GET", "/prices/buy", nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.verifyBody(t, rec.Body.String())
		})
	}
}
