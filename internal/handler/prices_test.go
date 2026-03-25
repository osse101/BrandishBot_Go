package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleGetPrices(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockEconomyService)
		expectedStatus int
		verifyBody     func(*testing.T, string)
	}{
		{
			name: "Success - Multiple Items",
			setupMock: func(m *mocks.MockEconomyService) {
				sellPrice1 := 4
				sellPrice2 := 8
				items := []domain.Item{
					{ID: 1, InternalName: "Item1", BaseValue: 10, SellPrice: &sellPrice1},
					{ID: 2, InternalName: "Item2", BaseValue: 20, SellPrice: &sellPrice2},
				}
				m.On("GetSellablePrices", mock.Anything).Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body string) {
				var items []domain.Item
				err := json.Unmarshal([]byte(body), &items)
				require.NoError(t, err)
				assert.Len(t, items, 2)
				assert.Equal(t, "Item1", items[0].InternalName)
				assert.Equal(t, 10, items[0].BaseValue)
				require.NotNil(t, items[0].SellPrice)
				assert.Equal(t, 4, *items[0].SellPrice)

				assert.Equal(t, "Item2", items[1].InternalName)
				assert.Equal(t, 20, items[1].BaseValue)
				require.NotNil(t, items[1].SellPrice)
				assert.Equal(t, 8, *items[1].SellPrice)
			},
		},
		{
			name: "Success - Empty List",
			setupMock: func(m *mocks.MockEconomyService) {
				m.On("GetSellablePrices", mock.Anything).Return([]domain.Item{}, nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body string) {
				var items []domain.Item
				err := json.Unmarshal([]byte(body), &items)
				require.NoError(t, err)
				assert.Len(t, items, 0)
			},
		},
		{
			name: "Service Error",
			setupMock: func(m *mocks.MockEconomyService) {
				m.On("GetSellablePrices", mock.Anything).Return(nil, domain.ErrDatabaseError)
			},
			expectedStatus: http.StatusInternalServerError,
			verifyBody: func(t *testing.T, body string) {
				var errResp ErrorResponse
				err := json.Unmarshal([]byte(body), &errResp)
				require.NoError(t, err)
				assert.Equal(t, ErrMsgGenericServerError, errResp.Error)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockSvc := mocks.NewMockEconomyService(t)
			tt.setupMock(mockSvc)

			handler := HandleGetPrices(mockSvc)

			req := httptest.NewRequest("GET", "/prices", nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.verifyBody(t, rec.Body.String())
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandleGetBuyPrices(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockEconomyService)
		expectedStatus int
		verifyBody     func(*testing.T, string)
	}{
		{
			name: "Success - Multiple Items",
			setupMock: func(m *mocks.MockEconomyService) {
				items := []domain.Item{
					{ID: 2, InternalName: "Item2", BaseValue: 20},
					{ID: 3, InternalName: "Item3", BaseValue: 50},
				}
				m.On("GetBuyablePrices", mock.Anything).Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body string) {
				var items []domain.Item
				err := json.Unmarshal([]byte(body), &items)
				require.NoError(t, err)
				assert.Len(t, items, 2)
				assert.Equal(t, "Item2", items[0].InternalName)
				assert.Equal(t, 20, items[0].BaseValue)
				assert.Nil(t, items[0].SellPrice)

				assert.Equal(t, "Item3", items[1].InternalName)
				assert.Equal(t, 50, items[1].BaseValue)
				assert.Nil(t, items[1].SellPrice)
			},
		},
		{
			name: "Success - Empty List",
			setupMock: func(m *mocks.MockEconomyService) {
				m.On("GetBuyablePrices", mock.Anything).Return([]domain.Item{}, nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body string) {
				var items []domain.Item
				err := json.Unmarshal([]byte(body), &items)
				require.NoError(t, err)
				assert.Len(t, items, 0)
			},
		},
		{
			name: "Service Error",
			setupMock: func(m *mocks.MockEconomyService) {
				m.On("GetBuyablePrices", mock.Anything).Return(nil, domain.ErrDatabaseError)
			},
			expectedStatus: http.StatusInternalServerError,
			verifyBody: func(t *testing.T, body string) {
				var errResp ErrorResponse
				err := json.Unmarshal([]byte(body), &errResp)
				require.NoError(t, err)
				assert.Equal(t, ErrMsgGenericServerError, errResp.Error)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockSvc := mocks.NewMockEconomyService(t)
			tt.setupMock(mockSvc)

			handler := HandleGetBuyPrices(mockSvc)

			req := httptest.NewRequest("GET", "/prices/buy", nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.verifyBody(t, rec.Body.String())
			mockSvc.AssertExpectations(t)
		})
	}
}
