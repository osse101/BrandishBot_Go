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

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

func init() {
	// Set log level to WARN for benchmarks (reduces noise)
	opts := &slog.HandlerOptions{Level: slog.LevelWarn}
	handler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
}

// Mock services for benchmarking
type mockUserService struct{}

func (m *mockUserService) HandleIncomingMessage(ctx context.Context, platform, platformID, username, message string) (*domain.MessageResult, error) {
	return &domain.MessageResult{
		User: domain.User{
			ID:       "bench-user-123",
			Username: username,
		},
		Matches: []domain.FoundString{},
	}, nil
}

func (m *mockUserService) RegisterUser(ctx context.Context, user domain.User) (domain.User, error) {
	return user, nil
}
func (m *mockUserService) FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (m *mockUserService) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (m *mockUserService) AddItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) error {
	return nil
}
func (m *mockUserService) AddItems(ctx context.Context, platform, platformID, username string, items map[string]int) error {
	return nil
}
func (m *mockUserService) AddItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) error {
	return nil
}
func (m *mockUserService) RemoveItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, error) {
	return 0, nil
}
func (m *mockUserService) RemoveItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) (int, error) {
	return 0, nil
}
func (m *mockUserService) GiveItem(ctx context.Context, ownerPlatform, ownerPlatformID, ownerUsername, receiverPlatform, receiverPlatformID, receiverUsername, itemName string, quantity int) error {
	return nil
}
func (m *mockUserService) GiveItemByUsername(ctx context.Context, ownerPlatform, ownerUsername, receiverPlatform, receiverUsername, itemName string, quantity int) (string, error) {
	return "", nil
}
func (m *mockUserService) UseItem(ctx context.Context, platform, platformID, username, itemName string, quantity int, targetUsername string) (string, error) {
	return "", nil
}
func (m *mockUserService) UseItemByUsername(ctx context.Context, platform, username, itemName string, quantity int, targetUsername string) (string, error) {
	return "", nil
}
func (m *mockUserService) GetInventory(ctx context.Context, platform, platformID, username, filter string) ([]user.UserInventoryItem, error) {
	return nil, nil
}
func (m *mockUserService) GetInventoryByUsername(ctx context.Context, platform, username, filter string) ([]user.UserInventoryItem, error) {
	return nil, nil
}
func (m *mockUserService) TimeoutUser(ctx context.Context, username string, duration time.Duration, reason string) error {
	return nil
}
func (m *mockUserService) HandleSearch(ctx context.Context, platform, platformID, username string) (string, error) {
	return "", nil
}
func (m *mockUserService) MergeUsers(ctx context.Context, primaryUserID, secondaryUserID string) error {
	return nil
}
func (m *mockUserService) UnlinkPlatform(ctx context.Context, userID, platform string) error {
	return nil
}
func (m *mockUserService) GetLinkedPlatforms(ctx context.Context, platform, platformID string) ([]string, error) {
	return nil, nil
}
func (m *mockUserService) GetTimeout(ctx context.Context, username string) (time.Duration, error) {
	return 0, nil
}
func (m *mockUserService) Shutdown(ctx context.Context) error {
	return nil
}

type mockProgressionService struct{}

func (m *mockProgressionService) GetProgressionTree(ctx context.Context) ([]*domain.ProgressionTreeNode, error) {
	return nil, nil
}
func (m *mockProgressionService) GetAvailableUnlocks(ctx context.Context) ([]*domain.ProgressionNode, error) {
	return nil, nil
}
func (m *mockProgressionService) GetNode(ctx context.Context, id int) (*domain.ProgressionNode, error) {
	return nil, nil
}
func (m *mockProgressionService) IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error) {
	return true, nil
}
func (m *mockProgressionService) IsItemUnlocked(ctx context.Context, itemName string) (bool, error) {
	return true, nil
}
func (m *mockProgressionService) VoteForUnlock(ctx context.Context, userID string, nodeKey string) error {
	return nil
}
func (m *mockProgressionService) GetActiveVotingSession(ctx context.Context) (*domain.ProgressionVotingSession, error) {
	return nil, nil
}
func (m *mockProgressionService) StartVotingSession(ctx context.Context, unlockedNodeID *int) error {
	return nil
}
func (m *mockProgressionService) EndVoting(ctx context.Context) (*domain.ProgressionVotingOption, error) {
	return nil, nil
}
func (m *mockProgressionService) CheckAndUnlockCriteria(ctx context.Context) (*domain.ProgressionUnlock, error) {
	return nil, nil
}
func (m *mockProgressionService) CheckAndUnlockNode(ctx context.Context) (*domain.ProgressionUnlock, error) {
	return nil, nil
}
func (m *mockProgressionService) ForceInstantUnlock(ctx context.Context) (*domain.ProgressionUnlock, error) {
	return nil, nil
}
func (m *mockProgressionService) GetUnlockProgress(ctx context.Context) (*domain.UnlockProgress, error) {
	return nil, nil
}
func (m *mockProgressionService) EstimateUnlockTime(ctx context.Context, nodeKey string) (*domain.UnlockEstimate, error) {
	return nil, nil
}
func (m *mockProgressionService) GetEngagementVelocity(ctx context.Context, days int) (*domain.VelocityMetrics, error) {
	return nil, nil
}
func (m *mockProgressionService) AddContribution(ctx context.Context, amount int) error {
	return nil
}
func (m *mockProgressionService) RecordEngagement(ctx context.Context, userID string, metricType string, value int) error {
	return nil
}
func (m *mockProgressionService) GetEngagementScore(ctx context.Context) (int, error) {
	return 0, nil
}
func (m *mockProgressionService) GetUserEngagement(ctx context.Context, userID string) (*domain.ContributionBreakdown, error) {
	return nil, nil
}
func (m *mockProgressionService) GetContributionLeaderboard(ctx context.Context, limit int) ([]domain.ContributionLeaderboardEntry, error) {
	return nil, nil
}
func (m *mockProgressionService) GetProgressionStatus(ctx context.Context) (*domain.ProgressionStatus, error) {
	return nil, nil
}
func (m *mockProgressionService) GetRequiredNodes(ctx context.Context, nodeKey string) ([]*domain.ProgressionNode, error) {
	return nil, nil
}
func (m *mockProgressionService) AdminUnlock(ctx context.Context, nodeKey string, level int) error {
	return nil
}
func (m *mockProgressionService) AdminRelock(ctx context.Context, nodeKey string, level int) error {
	return nil
}
func (m *mockProgressionService) ResetProgressionTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error {
	return nil
}
func (m *mockProgressionService) InvalidateWeightCache() {}
func (m *mockProgressionService) GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
	return 0, nil
}
func (m *mockProgressionService) GetModifierForFeature(ctx context.Context, featureKey string) (*progression.ValueModifier, error) {
	return nil, nil
}
func (m *mockProgressionService) Shutdown(ctx context.Context) error {
	return nil
}

type mockEventBus struct{}

func (m *mockEventBus) Publish(ctx context.Context, event event.Event) error {
	return nil
}

func (m *mockEventBus) Subscribe(eventType event.Type, handler event.Handler) {
	// No-op for benchmarking
}

// BenchmarkHandler_HandleMessage benchmarks the full HTTP handler
func BenchmarkHandler_HandleMessage(b *testing.B) {
	mockUserService := &mockUserService{}
	mockProgressionService := &mockProgressionService{}
	mockEventBus := &mockEventBus{}

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
	mockUserService := &mockUserService{}
	mockProgressionService := &mockProgressionService{}
	mockEventBus := &mockEventBus{}

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
	mockUserService := &mockUserService{}
	mockProgressionService := &mockProgressionService{}
	mockEventBus := &mockEventBus{}

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

func (m *mockUserService) GetCacheStats() user.CacheStats {
return user.CacheStats{}
}

func (m *mockUserService) UpdateUser(ctx context.Context, user domain.User) error {
return nil
}
