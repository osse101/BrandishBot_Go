package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/event"

	"github.com/osse101/BrandishBot_Go/internal/user"
	"github.com/osse101/BrandishBot_Go/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)


func TestHandleGetInventoryByUsername(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		setupMock      func(*mocks.MockUserService)
		expectedStatus int
		verifyBody     func(*testing.T, string)
	}{
		{
			name:        "Success",
			queryParams: map[string]string{"platform": "discord", "username": "user1"},
			setupMock: func(mUser *mocks.MockUserService) {
				items := []user.UserInventoryItem{{Name: "Item1", Quantity: 5}}
				mUser.On("GetInventoryByUsername", mock.Anything, "discord", "user1", "").Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body string) {
				var resp GetInventoryResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Len(t, resp.Items, 1)
				assert.Equal(t, "Item1", resp.Items[0].Name)
			},
		},
		{
			name:        "Missing Username",
			queryParams: map[string]string{"platform": "discord"},
			setupMock:   func(mUser *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			verifyBody:     func(t *testing.T, body string) {},
		},
		{
			name:        "Service Error",
			queryParams: map[string]string{"username": "user1"},
			setupMock: func(mUser *mocks.MockUserService) {
				mUser.On("GetInventoryByUsername", mock.Anything, "discord", "user1", "").Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			verifyBody:     func(t *testing.T, body string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := mocks.NewMockUserService(t)

			tt.setupMock(mockUserSvc)

			handler := HandleGetInventoryByUsername(mockUserSvc)

			req := httptest.NewRequest("GET", "/user/inventory-by-username", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			rec := httptest.NewRecorder()

			handler(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.verifyBody(t, rec.Body.String())
		})
	}
}

func TestHandleAddItemByUsername(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockUserService)
		expectedStatus int
	}{
		{
			name: "Success",
			requestBody: AddItemByUsernameRequest{
				Platform: "discord",
				Username: "user1",
				ItemName: "item1",
				Quantity: 1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("AddItemByUsername", mock.Anything, "discord", "user1", "item1", 1).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Service Error",
			requestBody: AddItemByUsernameRequest{
				Platform: "discord",
				Username: "user1",
				ItemName: "item1",
				Quantity: 1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("AddItemByUsername", mock.Anything, "discord", "user1", "item1", 1).Return(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := mocks.NewMockUserService(t)
			tt.setupMock(mockUserSvc)

			handler := HandleAddItemByUsername(mockUserSvc)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/item/add-by-username", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockUserSvc.AssertExpectations(t)
		})
	}
}

func TestHandleRemoveItemByUsername(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockUserService)
		expectedStatus int
	}{
		{
			name: "Success",
			requestBody: RemoveItemByUsernameRequest{
				Platform: "discord",
				Username: "user1",
				ItemName: "item1",
				Quantity: 1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("RemoveItemByUsername", mock.Anything, "discord", "user1", "item1", 1).Return(1, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Service Error",
			requestBody: RemoveItemByUsernameRequest{
				Platform: "discord",
				Username: "user1",
				ItemName: "item1",
				Quantity: 1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("RemoveItemByUsername", mock.Anything, "discord", "user1", "item1", 1).Return(0, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := mocks.NewMockUserService(t)
			tt.setupMock(mockUserSvc)

			handler := HandleRemoveItemByUsername(mockUserSvc)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/item/remove-by-username", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockUserSvc.AssertExpectations(t)
		})
	}
}

func TestHandleUseItemByUsername(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockUserService, *mocks.MockEventBus)
		expectedStatus int
	}{
		{
			name: "Success",
			requestBody: UseItemByUsernameRequest{
				Platform: "discord",
				Username: "user1",
				ItemName: "item1",
				Quantity: 1,
			},
			setupMock: func(u *mocks.MockUserService, e *mocks.MockEventBus) {
				u.On("UseItemByUsername", mock.Anything, "discord", "user1", "item1", 1, "").Return("Used item", nil)
				e.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
					return evt.Type == "item.used" || evt.Type == "engagement"
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Service Error",
			requestBody: UseItemByUsernameRequest{
				Platform: "discord",
				Username: "user1",
				ItemName: "item1",
				Quantity: 1,
			},
			setupMock: func(u *mocks.MockUserService, e *mocks.MockEventBus) {
				u.On("UseItemByUsername", mock.Anything, "discord", "user1", "item1", 1, "").Return("", errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := mocks.NewMockUserService(t)
			mockBus := mocks.NewMockEventBus(t)
			tt.setupMock(mockUserSvc, mockBus)

			handler := HandleUseItemByUsername(mockUserSvc, mockBus)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/item/use-by-username", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockUserSvc.AssertExpectations(t)
			mockBus.AssertExpectations(t)
		})
	}
}

func TestHandleGiveItemByUsername(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockUserService)
		expectedStatus int
	}{
		{
			name: "Success",
			requestBody: GiveItemByUsernameRequest{
				FromPlatform: "discord",
				FromUsername: "user1",
				ToPlatform:   "discord",
				ToUsername:   "user2",
				ItemName:     "item1",
				Quantity:     1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("GiveItemByUsername", mock.Anything, "discord", "user1", "discord", "user2", "item1", 1).Return("Given item", nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Service Error",
			requestBody: GiveItemByUsernameRequest{
				FromPlatform: "discord",
				FromUsername: "user1",
				ToPlatform:   "discord",
				ToUsername:   "user2",
				ItemName:     "item1",
				Quantity:     1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("GiveItemByUsername", mock.Anything, "discord", "user1", "discord", "user2", "item1", 1).Return("", errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := mocks.NewMockUserService(t)
			tt.setupMock(mockUserSvc)

			handler := HandleGiveItemByUsername(mockUserSvc)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/item/give-by-username", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockUserSvc.AssertExpectations(t)
		})
	}
}
