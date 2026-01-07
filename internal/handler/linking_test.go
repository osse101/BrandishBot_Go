package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/linking"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// MOCKS
// ============================================================================

type MockLinkingService struct {
	mock.Mock
}

func (m *MockLinkingService) InitiateLink(ctx context.Context, platform, platformID string) (*repository.LinkToken, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.LinkToken), args.Error(1)
}

func (m *MockLinkingService) ClaimLink(ctx context.Context, tokenStr, platform, platformID string) (*repository.LinkToken, error) {
	args := m.Called(ctx, tokenStr, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.LinkToken), args.Error(1)
}

func (m *MockLinkingService) ConfirmLink(ctx context.Context, platform, platformID string) (*linking.LinkResult, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*linking.LinkResult), args.Error(1)
}

func (m *MockLinkingService) InitiateUnlink(ctx context.Context, platform, platformID, targetPlatform string) error {
	args := m.Called(ctx, platform, platformID, targetPlatform)
	return args.Error(0)
}

func (m *MockLinkingService) ConfirmUnlink(ctx context.Context, platform, platformID, targetPlatform string) error {
	args := m.Called(ctx, platform, platformID, targetPlatform)
	return args.Error(0)
}

func (m *MockLinkingService) GetStatus(ctx context.Context, platform, platformID string) (*linking.LinkStatus, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*linking.LinkStatus), args.Error(1)
}

// ============================================================================
// REQUEST VALIDATION TESTS
// ============================================================================

func TestHandleInitiate_InvalidJSON(t *testing.T) {
	svc := new(MockLinkingService)
	handler := NewLinkingHandlers(svc)

	req := httptest.NewRequest(http.MethodPost, "/link/initiate", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handler.HandleInitiate()(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request")
}

// ============================================================================
// HTTP METHOD TESTS
// ============================================================================

func TestHandleInitiate_MethodNotAllowed(t *testing.T) {
	svc := new(MockLinkingService)
	handler := NewLinkingHandlers(svc)

	req := httptest.NewRequest(http.MethodGet, "/link/initiate", nil)
	w := httptest.NewRecorder()

	handler.HandleInitiate()(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleClaim_MethodNotAllowed(t *testing.T) {
	svc := new(MockLinkingService)
	handler := NewLinkingHandlers(svc)

	req := httptest.NewRequest(http.MethodGet, "/link/claim", nil)
	w := httptest.NewRecorder()

	handler.HandleClaim()(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ============================================================================
// HAPPY PATH TESTS
// ============================================================================

func TestHandleInitiate_Success(t *testing.T) {
	svc := new(MockLinkingService)
	handler := NewLinkingHandlers(svc)

	token := &repository.LinkToken{
		Token:            "ABC123",
		SourcePlatform:   domain.PlatformDiscord,
		SourcePlatformID: "discord-123",
		State:            linking.StatePending,
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(10 * time.Minute),
	}

	svc.On("InitiateLink", mock.Anything, domain.PlatformDiscord, "discord-123").Return(token, nil)

	body := InitiateRequest{
		Platform:   domain.PlatformDiscord,
		PlatformID: "discord-123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/link/initiate", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	handler.HandleInitiate()(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "ABC123", resp["token"])
	assert.NotNil(t, resp["expires_in"])

	svc.AssertExpectations(t)
}

func TestHandleClaim_Success(t *testing.T) {
	svc := new(MockLinkingService)
	handler := NewLinkingHandlers(svc)

	token := &repository.LinkToken{
		Token:            "ABC123",
		SourcePlatform:   domain.PlatformDiscord,
		SourcePlatformID: "discord-123",
		TargetPlatform:   domain.PlatformTwitch,
		TargetPlatformID: "twitch-456",
		State:            linking.StateClaimed,
		ExpiresAt:        time.Now().Add(10 * time.Minute),
	}

	svc.On("ClaimLink", mock.Anything, "ABC123", domain.PlatformTwitch, "twitch-456").Return(token, nil)

	body := ClaimRequest{
		Token:      "ABC123",
		Platform:   domain.PlatformTwitch,
		PlatformID: "twitch-456",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/link/claim", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	handler.HandleClaim()(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, domain.PlatformDiscord, resp["source_platform"])
	assert.Equal(t, true, resp["awaiting_confirmation"])

	svc.AssertExpectations(t)
}

func TestHandleConfirm_Success(t *testing.T) {
	svc := new(MockLinkingService)
	handler := NewLinkingHandlers(svc)

	result := &linking.LinkResult{
		Success:         true,
		LinkedPlatforms: []string{domain.PlatformDiscord, domain.PlatformTwitch},
	}

	svc.On("ConfirmLink", mock.Anything, domain.PlatformDiscord, "discord-123").Return(result, nil)

	body := ConfirmRequest{
		Platform:   domain.PlatformDiscord,
		PlatformID: "discord-123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/link/confirm", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	handler.HandleConfirm()(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp linking.LinkResult
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, []string{domain.PlatformDiscord, domain.PlatformTwitch}, resp.LinkedPlatforms)

	svc.AssertExpectations(t)
}

func TestHandleUnlink_InitiateSuccess(t *testing.T) {
	svc := new(MockLinkingService)
	handler := NewLinkingHandlers(svc)

	svc.On("InitiateUnlink", mock.Anything, domain.PlatformDiscord, "discord-123", domain.PlatformTwitch).Return(nil)

	body := UnlinkRequest{
		Platform:       domain.PlatformDiscord,
		PlatformID:     "discord-123",
		TargetPlatform: domain.PlatformTwitch,
		Confirm:        false,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/link/unlink", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	handler.HandleUnlink()(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, true, resp["awaiting_confirmation"])

	svc.AssertExpectations(t)
}

func TestHandleUnlink_ConfirmSuccess(t *testing.T) {
	svc := new(MockLinkingService)
	handler := NewLinkingHandlers(svc)

	svc.On("ConfirmUnlink", mock.Anything, domain.PlatformDiscord, "discord-123", domain.PlatformTwitch).Return(nil)

	body := UnlinkRequest{
		Platform:       domain.PlatformDiscord,
		PlatformID:     "discord-123",
		TargetPlatform: domain.PlatformTwitch,
		Confirm:        true,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/link/unlink", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	handler.HandleUnlink()(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, true, resp["success"])

	svc.AssertExpectations(t)
}

func TestHandleStatus_Success(t *testing.T) {
	svc := new(MockLinkingService)
	handler := NewLinkingHandlers(svc)

	status := &linking.LinkStatus{
		LinkedPlatforms: []string{domain.PlatformDiscord, domain.PlatformTwitch, domain.PlatformYoutube},
	}

	svc.On("GetStatus", mock.Anything, domain.PlatformDiscord, "discord-123").Return(status, nil)

	req := httptest.NewRequest(http.MethodGet, "/link/status?platform=discord&platform_id=discord-123", nil)
	w := httptest.NewRecorder()

	handler.HandleStatus()(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp linking.LinkStatus
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 3, len(resp.LinkedPlatforms))

	svc.AssertExpectations(t)
}

// ============================================================================
// ERROR RESPONSE TESTS
// ============================================================================

func TestHandleClaim_ExpiredToken(t *testing.T) {
	svc := new(MockLinkingService)
	handler := NewLinkingHandlers(svc)

	svc.On("ClaimLink", mock.Anything, "ABC123", domain.PlatformTwitch, "twitch-456").Return(nil, fmt.Errorf("token expired"))

	body := ClaimRequest{
		Token:      "ABC123",
		Platform:   domain.PlatformTwitch,
		PlatformID: "twitch-456",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/link/claim", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	handler.HandleClaim()(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "expired")

	svc.AssertExpectations(t)
}

func TestHandleConfirm_NoToken(t *testing.T) {
	svc := new(MockLinkingService)
	handler := NewLinkingHandlers(svc)

	svc.On("ConfirmLink", mock.Anything, domain.PlatformDiscord, "discord-123").Return(nil, fmt.Errorf("no pending link to confirm"))

	body := ConfirmRequest{
		Platform:   domain.PlatformDiscord,
		PlatformID: "discord-123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/link/confirm", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	handler.HandleConfirm()(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	svc.AssertExpectations(t)
}

func TestHandleStatus_MissingParams(t *testing.T) {
	svc := new(MockLinkingService)
	handler := NewLinkingHandlers(svc)

	// Missing platform_id
	req := httptest.NewRequest(http.MethodGet, "/link/status?platform=discord", nil)
	w := httptest.NewRecorder()

	handler.HandleStatus()(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Missing platform")
}

func TestHandleInitiate_ServiceError(t *testing.T) {
	svc := new(MockLinkingService)
	handler := NewLinkingHandlers(svc)

	svc.On("InitiateLink", mock.Anything, domain.PlatformDiscord, "discord-123").Return(nil, fmt.Errorf("internal error"))

	body := InitiateRequest{
		Platform:   domain.PlatformDiscord,
		PlatformID: "discord-123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/link/initiate", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	handler.HandleInitiate()(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	svc.AssertExpectations(t)
}
