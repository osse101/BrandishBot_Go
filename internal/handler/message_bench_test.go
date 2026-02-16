package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/user"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func init() {
	// Set log level to WARN for benchmarks (reduces noise)
	opts := &slog.HandlerOptions{Level: slog.LevelWarn}
	handler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
}

// Manual mocks for benchmarking to avoid testify overhead and strictness
type benchMockUserService struct {
	mock.Mock
}

func (m *benchMockUserService) HandleIncomingMessage(ctx context.Context, platform, platformID, username, message string) (*domain.MessageResult, error) {
	if username == "matchuser" {
		return &domain.MessageResult{
			User: domain.User{ID: "bench-user-123", Username: "matchuser"},
			Matches: []domain.FoundString{
				{Code: "test", Value: "test item"},
			},
		}, nil
	}
	return &domain.MessageResult{
		User:    domain.User{ID: "bench-user-123", Username: username},
		Matches: []domain.FoundString{},
	}, nil
}

// Stubs for other interface methods to satisfy dependencies if needed
func (m *benchMockUserService) GetUser(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return nil, nil
}

func (m *benchMockUserService) AddItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) error {
	return nil
}

func (m *benchMockUserService) AddTimeout(ctx context.Context, platform, username string, duration time.Duration, reason string) error {
	return nil
}

func (m *benchMockUserService) ApplyShield(ctx context.Context, user *domain.User, quantity int, isMirror bool) error {
	return nil
}

// Implement remaining Service interface methods with stubs
func (m *benchMockUserService) UseItem(ctx context.Context, platform, platformID, username, itemName string, quantity int, targetUsername string) (string, error) {
	return "", nil
}
func (m *benchMockUserService) GetInventory(ctx context.Context, platform, platformID, username, filter string) ([]user.InventoryItem, error) {
	return nil, nil
}
func (m *benchMockUserService) GiveItem(ctx context.Context, ownerPlatform, ownerPlatformID, ownerUsername, receiverPlatform, receiverUsername, itemName string, quantity int) error {
	return nil
}
func (m *benchMockUserService) RemoveItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) (int, error) {
	return 0, nil
}
func (m *benchMockUserService) GetInventoryByUsername(ctx context.Context, platform, username, filter string) ([]user.InventoryItem, error) {
	return nil, nil
}
func (m *benchMockUserService) RegisterUser(ctx context.Context, u domain.User) (domain.User, error) {
	return u, nil
}
func (m *benchMockUserService) UpdateUser(ctx context.Context, u domain.User) error {
	return nil
}
func (m *benchMockUserService) FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return nil, nil
}
func (m *benchMockUserService) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	return nil, nil
}
func (m *benchMockUserService) MergeUsers(ctx context.Context, primaryUserID, secondaryUserID string) error {
	return nil
}
func (m *benchMockUserService) UnlinkPlatform(ctx context.Context, userID, platform string) error {
	return nil
}
func (m *benchMockUserService) GetLinkedPlatforms(ctx context.Context, platform, platformID string) ([]string, error) {
	return nil, nil
}
func (m *benchMockUserService) HandleSearch(ctx context.Context, platform, platformID, username string) (string, error) {
	return "", nil
}
func (m *benchMockUserService) ClearTimeout(ctx context.Context, platform, username string) error {
	return nil
}
func (m *benchMockUserService) GetTimeoutPlatform(ctx context.Context, platform, username string) (time.Duration, error) {
	return 0, nil
}
func (m *benchMockUserService) ReduceTimeoutPlatform(ctx context.Context, platform, username string, reduction time.Duration) error {
	return nil
}
func (m *benchMockUserService) TimeoutUser(ctx context.Context, username string, duration time.Duration, reason string) error {
	return nil
}
func (m *benchMockUserService) GetTimeout(ctx context.Context, username string) (time.Duration, error) {
	return 0, nil
}
func (m *benchMockUserService) ReduceTimeout(ctx context.Context, username string, reduction time.Duration) error {
	return nil
}
func (m *benchMockUserService) GetCacheStats() user.CacheStats {
	return user.CacheStats{}
}
func (m *benchMockUserService) Shutdown(ctx context.Context) error {
	return nil
}
func (m *benchMockUserService) GetActiveChatters() []user.ActiveChatter {
	return nil
}

type benchMockEventBus struct{}

func (m *benchMockEventBus) Publish(ctx context.Context, evt event.Event) error {
	return nil
}

func (m *benchMockEventBus) Subscribe(topic event.Type, handler event.Handler) {
}

// BenchmarkHandler_HandleMessage benchmarks the full HTTP handler
func BenchmarkHandler_HandleMessage(b *testing.B) {
	mockUserService := &benchMockUserService{}
	mockProgressionService := mocks.NewMockProgressionService(b) // Keep this for now or replace if needed
	// Actually handling progression service might be complex, let's see if we can use the testify one if we configure it correctly,
	// or just stub it. Ideally we stub everything.
	// For now, let's just stub the method we know is called if any.
	// But wait, the handler likely doesn't call progression service directly in the hot path unless there's a match?

	mockEventBus := &benchMockEventBus{}

	handler := HandleMessageHandler(mockUserService, mockProgressionService, mockEventBus)

	reqBody := HandleMessageRequest{
		Platform:   "twitch",
		PlatformID: "12345",
		Username:   "benchuser",
		Message:    "hello world",
	}

	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/message/handle", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkHandler_HandleMessage_ExistingUser benchmarks with cached user lookup
func BenchmarkHandler_HandleMessage_ExistingUser(b *testing.B) {
	mockUserService := &benchMockUserService{}
	mockProgressionService := mocks.NewMockProgressionService(b)
	mockEventBus := &benchMockEventBus{}

	handler := HandleMessageHandler(mockUserService, mockProgressionService, mockEventBus)

	reqBody := HandleMessageRequest{
		Platform:   "twitch",
		PlatformID: "existing-user-123",
		Username:   "existinguser",
		Message:    "test message",
	}

	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/message/handle", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
	}
}

// BenchmarkHandler_HandleMessage_WithMatches benchmarks message with string matches
func BenchmarkHandler_HandleMessage_WithMatches(b *testing.B) {
	mockUserService := &benchMockUserService{}
	mockProgressionService := mocks.NewMockProgressionService(b)
	mockEventBus := &benchMockEventBus{}

	handler := HandleMessageHandler(mockUserService, mockProgressionService, mockEventBus)

	reqBody := HandleMessageRequest{
		Platform:   "discord",
		PlatformID: "discord-789",
		Username:   "matchuser",
		Message:    "this message contains multiple words for matching",
	}

	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/message/handle", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
	}
}
