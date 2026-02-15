package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// Mock services for dependencies we don't need to test in this integration test
type MockJobService struct{}

func (m *MockJobService) GetUserJobs(ctx context.Context, userID string) ([]domain.UserJobInfo, error) {
	return []domain.UserJobInfo{}, nil
}

func (m *MockJobService) GetUserJobsByPlatform(ctx context.Context, platform, platformID string) ([]domain.UserJobInfo, error) {
	return []domain.UserJobInfo{}, nil
}

func (m *MockJobService) GetPrimaryJob(ctx context.Context, platform, platformID string) (*domain.UserJobInfo, error) {
	return &domain.UserJobInfo{JobKey: "explorer"}, nil
}

func (m *MockJobService) GetJobBonus(ctx context.Context, userID, jobKey string, bonusType string) (float64, error) {
	return 0, nil
}

func (m *MockJobService) AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata domain.JobXPMetadata) (*domain.XPAwardResult, error) {
	return &domain.XPAwardResult{}, nil
}

func (m *MockJobService) AwardXPByPlatform(ctx context.Context, platform, platformID, jobKey string, baseAmount int, source string, metadata domain.JobXPMetadata) (*domain.XPAwardResult, error) {
	return &domain.XPAwardResult{}, nil
}

func (m *MockJobService) GetJobLevel(ctx context.Context, userID, jobKey string) (int, error) {
	return 1, nil
}

func (m *MockJobService) ResetDailyJobXP(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *MockJobService) GetDailyResetStatus(ctx context.Context) (*domain.DailyResetStatus, error) {
	return &domain.DailyResetStatus{}, nil
}

func (m *MockJobService) GetAllJobs(ctx context.Context) ([]domain.Job, error) {
	return []domain.Job{}, nil
}

func (m *MockJobService) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return &domain.User{}, nil
}

func (m *MockJobService) CalculateLevel(totalXP int64) int {
	return 1
}

func (m *MockJobService) GetXPForLevel(level int) int64 {
	return 1000
}

func (m *MockJobService) GetXPProgress(currentXP int64) (currentLevel int, xpToNext int64) {
	return 1, 1000
}

func (m *MockJobService) Shutdown(ctx context.Context) error {
	return nil
}

type MockLootboxService struct{}

func (m *MockLootboxService) OpenLootbox(ctx context.Context, lootboxName string, quantity int, boxQuality domain.QualityLevel) ([]lootbox.DroppedItem, error) {
	return []lootbox.DroppedItem{}, nil
}

type MockStatsService struct{}

func (m *MockStatsService) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata interface{}) error {
	return nil
}

func (m *MockStatsService) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) {
	return &domain.StatsSummary{EventCounts: make(map[domain.EventType]int)}, nil
}

func (m *MockStatsService) GetUserCurrentStreak(ctx context.Context, userID string) (int, error) {
	return 0, nil
}

func (m *MockStatsService) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	return &domain.StatsSummary{EventCounts: make(map[domain.EventType]int)}, nil
}

func (m *MockStatsService) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	return []domain.LeaderboardEntry{}, nil
}

func (m *MockStatsService) GetUserSlotsStats(ctx context.Context, userID, period string) (*domain.SlotsStats, error) {
	return nil, nil
}

func (m *MockStatsService) GetSlotsLeaderboardByProfit(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error) {
	return nil, nil
}

func (m *MockStatsService) GetSlotsLeaderboardByWinRate(ctx context.Context, period string, minSpins, limit int) ([]domain.SlotsStats, error) {
	return nil, nil
}

func (m *MockStatsService) GetSlotsLeaderboardByMegaJackpots(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error) {
	return nil, nil
}

type MockNamingResolver struct{}

func (m *MockNamingResolver) GetDisplayName(internalName string, qualityLevel domain.QualityLevel) string {
	return internalName
}

func (m *MockNamingResolver) GetDescription(internalName string) string {
	return "description"
}

func (m *MockNamingResolver) ResolvePublicName(publicName string) (internalName string, ok bool) {
	return publicName, true
}

func (m *MockNamingResolver) GetActiveTheme() string {
	return ""
}

func (m *MockNamingResolver) Reload() error {
	return nil
}

func (m *MockNamingResolver) RegisterItem(internalName, publicName string) {
	// No-op
}

type MockBus struct {
	delay time.Duration
}

func (m *MockBus) Publish(ctx context.Context, evt event.Event) error {
	time.Sleep(m.delay)
	return nil
}

func (m *MockBus) Subscribe(eventType event.Type, handler event.Handler) {}

func setupIntegrationTest(t *testing.T) (*pgxpool.Pool, *UserRepository, user.Service) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if testDBConnString == "" {
		t.Skip("Skipping integration test: database not available")
	}

	// Use shared pool and migrations
	ensureMigrations(t)

	repo := NewUserRepository(testPool)
	trapRepo := NewTrapRepository(testPool)
	cooldownConfig := cooldown.Config{DevMode: true}
	cooldownSvc := cooldown.NewPostgresService(testPool, cooldownConfig, nil)

	svc := user.NewService(
		repo,
		trapRepo,
		&MockStatsService{},
		nil, // publisher
		&MockLootboxService{},
		&MockNamingResolver{},
		cooldownSvc,
		nil,  // jobService
		nil,  // No event bus for tests
		true, // Dev mode to bypass cooldowns
	)

	return testPool, repo, svc
}

func TestUserService_InventoryOperations_Integration(t *testing.T) {
	pool, repo, svc := setupIntegrationTest(t)
	if svc == nil {
		return // Skipped
	}
	ctx := context.Background()
	_ = pool // suppress unused warning

	t.Run("Concurrent GiveItem Between Users", func(t *testing.T) {
		// Setup users
		userA := &domain.User{Username: "inventoryUserA", TwitchID: "inventorytwitchA"}
		userB := &domain.User{Username: "inventoryUserB", TwitchID: "inventorytwitchB"}

		// Register users and seed inventory
		if err := repo.UpsertUser(ctx, userA); err != nil {
			t.Fatalf("failed to setup userA: %v", err)
		}
		if err := repo.UpsertUser(ctx, userB); err != nil {
			t.Fatalf("failed to setup userB: %v", err)
		}

		// Refresh IDs
		userA, _ = repo.GetUserByPlatformUsername(ctx, domain.PlatformTwitch, userA.Username)
		userB, _ = repo.GetUserByPlatformUsername(ctx, domain.PlatformTwitch, userB.Username)

		// Give userA lots of money
		moneyItem, err := repo.GetItemByName(ctx, domain.ItemMoney)
		if err != nil || moneyItem == nil {
			t.Fatalf("money item not found")
		}

		initialAmount := 1000
		invA := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: moneyItem.ID, Quantity: initialAmount}}}
		if err := repo.UpdateInventory(ctx, userA.ID, *invA); err != nil {
			t.Fatalf("failed to seed inventory: %v", err)
		}

		// Execute concurrent transfers
		concurrency := 10
		transferAmount := 10

		var wg sync.WaitGroup
		errChan := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Using GiveItem service method
				err := svc.GiveItem(
					ctx,
					domain.PlatformTwitch, userA.TwitchID, userA.Username,
					domain.PlatformTwitch, userB.Username,
					domain.ItemMoney, transferAmount,
				)
				if err != nil {
					errChan <- err
				}
			}()
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			t.Errorf("concurrent transfer failed: %v", err)
		}

		// Verify final state
		finalInvA, _ := repo.GetInventory(ctx, userA.ID)
		finalInvB, _ := repo.GetInventory(ctx, userB.ID)

		expectedA := initialAmount - (concurrency * transferAmount)
		expectedB := concurrency * transferAmount

		var actualA, actualB int
		for _, s := range finalInvA.Slots {
			if s.ItemID == moneyItem.ID {
				actualA = s.Quantity
			}
		}
		for _, s := range finalInvB.Slots {
			if s.ItemID == moneyItem.ID {
				actualB = s.Quantity
			}
		}

		if actualA != expectedA {
			t.Errorf("UserA balance incorrect. Want %d, Got %d", expectedA, actualA)
		}
		if actualB != expectedB {
			t.Errorf("UserB balance incorrect. Want %d, Got %d", expectedB, actualB)
		}
	})

	t.Run("AddItem to Full Inventory", func(t *testing.T) {
		userC := &domain.User{Username: "userC", TwitchID: "twitchC"}
		if err := repo.UpsertUser(ctx, userC); err != nil {
			t.Fatalf("failed to setup userC: %v", err)
		}

		// Add item via service
		err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, userC.Username, domain.ItemMoney, 50)
		if err != nil {
			t.Errorf("AddItem failed: %v", err)
		}

		// Verify
		userC, _ = repo.GetUserByPlatformUsername(ctx, domain.PlatformTwitch, userC.Username)
		inv, _ := repo.GetInventory(ctx, userC.ID)

		moneyItem, _ := repo.GetItemByName(ctx, domain.ItemMoney)
		found := false
		for _, s := range inv.Slots {
			if s.ItemID == moneyItem.ID {
				if s.Quantity != 50 {
					t.Errorf("Expected 50 money, got %d", s.Quantity)
				}
				found = true
			}
		}
		if !found {
			t.Error("Money not found in inventory")
		}
	})
}

func TestUserService_AsyncXPAward_Integration(t *testing.T) {
	pool, repo, _ := setupIntegrationTest(t)
	if pool == nil {
		return // Skipped
	}

	cooldownConfig := cooldown.Config{DevMode: true}
	cooldownSvc := cooldown.NewPostgresService(pool, cooldownConfig, nil)
	trapRepo := NewTrapRepository(pool)

	// Create a real publisher with a mock bus for testing async wait
	mockBus := &MockBus{delay: 200 * time.Millisecond}
	publisher, _ := event.NewResilientPublisher(mockBus, 3, 10*time.Millisecond, "test_deadletter.jsonl")
	defer os.Remove("test_deadletter.jsonl")

	svc := user.NewService(
		repo,
		trapRepo,
		&MockStatsService{},
		publisher,
		&MockLootboxService{},
		&MockNamingResolver{},
		cooldownSvc,
		&MockJobService{},
		nil, // No event bus for tests
		true,
	)

	ctx := context.Background()
	userD := &domain.User{Username: "userD", TwitchID: "twitchD"}
	repo.UpsertUser(ctx, userD)

	start := time.Now()

	triggered := false
	for i := 0; i < 5; i++ {
		msg, err := svc.HandleSearch(ctx, domain.PlatformTwitch, userD.TwitchID, userD.Username)
		if err == nil && (len(msg) > 0 && msg != domain.MsgSearchNearMiss && msg != domain.MsgSearchCriticalFail) {
			triggered = true
		}
	}

	if !triggered {
		t.Log("Could not trigger search success to test async wait, skipping strict check")
	}

	// Immediate Shutdown
	err := svc.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	elapsed := time.Since(start)
	if triggered && elapsed < 200*time.Millisecond {
		t.Error("Shutdown did not wait for async XP award (took less than 200ms)")
	}
}

type SlowJobService struct {
	delay time.Duration
}

func (m *SlowJobService) AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error) {
	time.Sleep(m.delay)
	return &domain.XPAwardResult{LeveledUp: false, NewLevel: 1, NewXP: 100}, nil
}

// Integration tests for username lookup functionality
func TestGetUserByPlatformUsername_Integration(t *testing.T) {
	_, repo, _ := setupIntegrationTest(t)
	if repo == nil {
		return // Skipped
	}
	ctx := context.Background()

	// Setup users
	alice := &domain.User{
		Username:  "Alice",
		TwitchID:  "twitch_alice",
		DiscordID: "discord_alice",
	}
	bob := &domain.User{
		Username: "Bob",
		TwitchID: "twitch_bob",
	}

	if err := repo.UpsertUser(ctx, alice); err != nil {
		t.Fatalf("failed to create alice: %v", err)
	}
	if err := repo.UpsertUser(ctx, bob); err != nil {
		t.Fatalf("failed to create bob: %v", err)
	}

	t.Run("successful lookup by username", func(t *testing.T) {
		user, err := repo.GetUserByPlatformUsername(ctx, domain.PlatformTwitch, alice.Username)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if user.Username != "Alice" {
			t.Errorf("Expected username Alice, got %s", user.Username)
		}
		if user.TwitchID != "twitch_alice" {
			t.Errorf("Expected twitch ID, got %s", user.TwitchID)
		}
	})

	t.Run("case insensitive lookup", func(t *testing.T) {
		user, err := repo.GetUserByPlatformUsername(ctx, domain.PlatformTwitch, strings.ToUpper(alice.Username))
		if err != nil {
			t.Fatalf("Expected no error with uppercase, got %v", err)
		}
		if user.Username != "Alice" {
			t.Errorf("Expected username Alice, got %s", user.Username)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := repo.GetUserByPlatformUsername(ctx, domain.PlatformTwitch, "nonexistent")
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Errorf("Expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("user without platform link", func(t *testing.T) {
		// Bob doesn't have Discord
		_, err := repo.GetUserByPlatformUsername(ctx, domain.PlatformDiscord, bob.Username)
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Errorf("Expected ErrUserNotFound for missing platform, got %v", err)
		}
	})
}

func TestUsernameBasedMethods_Integration(t *testing.T) {
	_, repo, svc := setupIntegrationTest(t)
	if svc == nil {
		return // Skipped
	}
	ctx := context.Background()

	// Setup test users
	charlie := &domain.User{Username: "Charlie", TwitchID: "twitch_charlie"}
	diana := &domain.User{Username: "Diana", DiscordID: "discord_diana"}

	if err := repo.UpsertUser(ctx, charlie); err != nil {
		t.Fatalf("failed to setup charlie: %v", err)
	}
	if err := repo.UpsertUser(ctx, diana); err != nil {
		t.Fatalf("failed to setup diana: %v", err)
	}

	// Refresh IDs
	charlie, _ = repo.GetUserByPlatformUsername(ctx, domain.PlatformTwitch, "Charlie")
	diana, _ = repo.GetUserByPlatformUsername(ctx, domain.PlatformDiscord, "Diana")

	t.Run("AddItemByUsername", func(t *testing.T) {
		err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, charlie.Username, domain.ItemMoney, 100)
		if err != nil {
			t.Fatalf("AddItemByUsername failed: %v", err)
		}

		// Verify in database
		inv, _ := repo.GetInventory(ctx, charlie.ID)
		moneyItem, _ := repo.GetItemByName(ctx, domain.ItemMoney)
		found := false
		for _, slot := range inv.Slots {
			if slot.ItemID == moneyItem.ID && slot.Quantity == 100 {
				found = true
			}
		}
		if !found {
			t.Error("Money not added via AddItemByUsername")
		}
	})

	t.Run("GetInventoryByUsername", func(t *testing.T) {
		items, err := svc.GetInventoryByUsername(ctx, domain.PlatformTwitch, charlie.Username, "")
		if err != nil {
			t.Fatalf("GetInventoryByUsername failed: %v", err)
		}
		if len(items) != 1 {
			t.Errorf("Expected 1 item in inventory, got %d", len(items))
		}
	})

	t.Run("RemoveItemByUsername", func(t *testing.T) {
		removed, err := svc.RemoveItemByUsername(ctx, domain.PlatformTwitch, charlie.Username, domain.ItemMoney, 30)
		if err != nil {
			t.Fatalf("RemoveItemByUsername failed: %v", err)
		}
		if removed != 30 {
			t.Errorf("Expected 30 removed, got %d", removed)
		}

		// Verify
		inv, _ := repo.GetInventory(ctx, charlie.ID)
		moneyItem, _ := repo.GetItemByName(ctx, domain.ItemMoney)
		for _, slot := range inv.Slots {
			if slot.ItemID == moneyItem.ID && slot.Quantity != 70 {
				t.Errorf("Expected 70 remaining, got %d", slot.Quantity)
			}
		}
	})

	t.Run("GiveItemByUsername cross-platform", func(t *testing.T) {
		// Give from Charlie (twitch) to Diana (discord) using usernames
		err := svc.GiveItem(ctx, domain.PlatformTwitch, charlie.TwitchID, charlie.Username, domain.PlatformDiscord, diana.Username, domain.ItemMoney, 20)
		if err != nil {
			t.Fatalf("GiveItemByUsername failed: %v", err)
		}

		// Verify Charlie has 50 left (70 - 20)
		invCharlie, _ := repo.GetInventory(ctx, charlie.ID)
		moneyItem, _ := repo.GetItemByName(ctx, domain.ItemMoney)
		for _, slot := range invCharlie.Slots {
			if slot.ItemID == moneyItem.ID && slot.Quantity != 50 {
				t.Errorf("Charlie should have 50, got %d", slot.Quantity)
			}
		}

		// Verify Diana has 20
		invDiana, _ := repo.GetInventory(ctx, diana.ID)
		found := false
		for _, slot := range invDiana.Slots {
			if slot.ItemID == moneyItem.ID {
				found = true
				if slot.Quantity != 20 {
					t.Errorf("Diana should have 20, got %d", slot.Quantity)
				}
			}
		}
		if !found {
			t.Error("Diana should have received money")
		}
	})

	t.Run("Case insensitive service operations", func(t *testing.T) {
		// All caps username should still work
		err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, strings.ToUpper(charlie.Username), domain.ItemMoney, 10)
		if err != nil {
			t.Fatalf("Case insensitive AddItemByUsername failed: %v", err)
		}

		inv, _ := repo.GetInventory(ctx, charlie.ID)
		moneyItem, _ := repo.GetItemByName(ctx, domain.ItemMoney)
		for _, slot := range inv.Slots {
			if slot.ItemID == moneyItem.ID && slot.Quantity != 60 {
				t.Errorf("Expected 60 (50 + 10), got %d", slot.Quantity)
			}
		}
	})
}
