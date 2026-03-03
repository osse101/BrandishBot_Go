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
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleGiveItem(t *testing.T) {
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
			requestBody: GiveItemRequest{
				OwnerPlatform:    domain.PlatformTwitch,
				OwnerPlatformID:  "owner-id",
				Owner:            "owner",
				ReceiverPlatform: domain.PlatformTwitch,
				Receiver:         "receiver",
				ItemName:         domain.ItemMissile,
				Quantity:         1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("GiveItem", mock.Anything, domain.PlatformTwitch, "owner-id", "owner", domain.PlatformTwitch, "receiver", domain.ItemMissile, 1).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Item transferred successfully"}`,
		},
		{
			name: "Service Error",
			requestBody: GiveItemRequest{
				OwnerPlatform:    domain.PlatformTwitch,
				OwnerPlatformID:  "owner-id",
				Owner:            "owner",
				ReceiverPlatform: domain.PlatformTwitch,
				Receiver:         "receiver",
				ItemName:         domain.ItemMissile,
				Quantity:         1,
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("GiveItem", mock.Anything, domain.PlatformTwitch, "owner-id", "owner", domain.PlatformTwitch, "receiver", domain.ItemMissile, 1).Return(errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockUserService(t)
			tt.setupMock(mockSvc)

			handler := HandleGiveItem(mockSvc)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/item/give", bytes.NewBuffer(body))
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
