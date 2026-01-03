package user

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

func init() {
	// Set log level to WARN for benchmarks (reduces noise)
	opts := &slog.HandlerOptions{Level: slog.LevelWarn}
	handler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
}

// Mock repository for benchmarking
type mockBenchRepository struct{}

func (m *mockBenchRepository) UpsertUser(ctx context.Context, user *domain.User) error {
	return nil
}

func (m *mockBenchRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	// Simulate cache miss for new user benchmark
	return nil, domain.ErrUserNotFound
}

func (m *mockBenchRepository) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}


func (m *mockBenchRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	return &domain.User{ID: userID, Username: "benchuser"}, nil
}

func (m *mockBenchRepository) UpdateUser(ctx context.Context, user domain.User) error {
	return nil
}

func (m *mockBenchRepository) DeleteUser(ctx context.Context, userID string) error {
	return nil
}

func (m *mockBenchRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 100}, // money
		},
	}, nil
}

func (m *mockBenchRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return nil
}

func (m *mockBenchRepository) DeleteInventory(ctx context.Context, userID string) error {
	return nil
}

func (m *mockBenchRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	return &domain.Item{
		ID:           42,
		InternalName: itemName,
		Description:  "Benchmark item",
		BaseValue:    10,
	}, nil
}

func (m *mockBenchRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	return &domain.Item{
		ID:           id,
		InternalName: "bench_item",
		Description:  "Benchmark item",
		BaseValue:    10,
	}, nil
}

func (m *mockBenchRepository) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	items := make([]domain.Item, len(itemIDs))
	for i, id := range itemIDs {
		items[i] = domain.Item{
			ID:           id,
			InternalName: "item_" + string(rune(id)),
			Description:  "Item description",
			BaseValue:    10,
		}
	}
	return items, nil
}

func (m *mockBenchRepository) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	return nil, nil
}

func (m *mockBenchRepository) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	return false, nil
}

func (m *mockBenchRepository) BeginTx(ctx context.Context) (repository.Tx, error) {
	return &mockBenchTx{repo: m}, nil
}

func (m *mockBenchRepository) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	return nil, nil
}

func (m *mockBenchRepository) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	return false, nil
}

func (m *mockBenchRepository) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	return nil
}

func (m *mockBenchRepository) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]crafting.UnlockedRecipeInfo, error) {
	return nil, nil
}

func (m *mockBenchRepository) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	return nil, nil
}

func (m *mockBenchRepository) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	return nil
}

func (m *mockBenchRepository) MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, mergedUser domain.User, mergedInventory domain.Inventory) error {
	return nil
}

// Mock transaction
type mockBenchTx struct {
	repo *mockBenchRepository
}

func (m *mockBenchTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return m.repo.GetInventory(ctx, userID)
}

func (m *mockBenchTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return m.repo.UpdateInventory(ctx, userID, inventory)
}

func (m *mockBenchTx) Commit(ctx context.Context) error {
	return nil
}

func (m *mockBenchTx) Rollback(ctx context.Context) error {
	return nil
}

// Mock stats service
type mockBenchStatsService struct{}

func (m *mockBenchStatsService) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error {
	return nil
}

func (m *mockBenchStatsService) GetUserStats(ctx context.Context, userID, period string) (*domain.StatsSummary, error) {
	return nil, nil
}

func (m *mockBenchStatsService) GetUserCurrentStreak(ctx context.Context, userID string) (int, error) {
	return 0, nil
}

func (m *mockBenchStatsService) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	return nil, nil
}

func (m *mockBenchStatsService) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}

// Mock job service
type mockBenchJobService struct{}

func (m *mockBenchJobService) AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error) {
	return &domain.XPAwardResult{
		JobKey:    jobKey,
		XPGained:  baseAmount,
		NewXP:     int64(baseAmount),
		NewLevel:  1,
		LeveledUp: false,
	}, nil
}

// Mock lootbox service
type mockBenchLootboxService struct{}

func (m *mockBenchLootboxService) OpenLootbox(ctx context.Context, lootboxName string, quantity int) ([]lootbox.DroppedItem, error) {
	return []lootbox.DroppedItem{
		{ItemID: 1, ItemName: "money", Quantity: 10, Value: 10, ShineLevel: "COMMON"},
	}, nil
}

// Mock naming resolver
type mockBenchNamingResolver struct{}

func (m *mockBenchNamingResolver) ResolvePublicName(publicName string) (string, bool) {
	return publicName, true
}

func (m *mockBenchNamingResolver) GetDisplayName(internalName string, shineLevel string) string {
	return internalName
}

func (m *mockBenchNamingResolver) GetActiveTheme() string {
	return ""
}

func (m *mockBenchNamingResolver) Reload() error {
	return nil
}

func (m *mockBenchNamingResolver) RegisterItem(internalName, publicName string) {
	// no-op
}

// Mock cooldown service
type mockBenchCooldownService struct{}

func (m *mockBenchCooldownService) CheckCooldown(ctx context.Context, userID, action string) (bool, time.Duration, error) {
	return false, 0, nil
}

func (m *mockBenchCooldownService) EnforceCooldown(ctx context.Context, userID, action string, fn func() error) error {
	return fn()
}

func (m *mockBenchCooldownService) ResetCooldown(ctx context.Context, userID, action string) error {
	return nil
}

func (m *mockBenchCooldownService) GetLastUsed(ctx context.Context, userID, action string) (*time.Time, error) {
	return nil, nil
}

// BenchmarkService_HandleIncomingMessage benchmarks user lookup/creation
func BenchmarkService_HandleIncomingMessage(b *testing.B) {
	repo := &mockBenchRepository{}
	statsService := &mockBenchStatsService{}
	jobService := &mockBenchJobService{}
	lootboxService := &mockBenchLootboxService{}
	namingResolver := &mockBenchNamingResolver{}
	cooldownService := &mockBenchCooldownService{}

	service := NewService(repo, statsService, jobService, lootboxService, namingResolver, cooldownService, false)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.HandleIncomingMessage(ctx, "twitch", "bench-123", "benchuser", "hello")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkService_HandleIncomingMessage_WithMatches benchmarks with message parsing
func BenchmarkService_HandleIncomingMessage_WithMatches(b *testing.B) {
	repo := &mockBenchRepository{}
	statsService := &mockBenchStatsService{}
	jobService := &mockBenchJobService{}
	lootboxService := &mockBenchLootboxService{}
	namingResolver := &mockBenchNamingResolver{}
	cooldownService := &mockBenchCooldownService{}

	service := NewService(repo, statsService, jobService, lootboxService, namingResolver, cooldownService, false)

	ctx := context.Background()
	message := "this is a longer message with multiple words to test string matching performance"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.HandleIncomingMessage(ctx, "discord", "bench-456", "matchuser", message)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkService_AddItem benchmarks inventory transaction performance
func BenchmarkService_AddItem(b *testing.B) {
	repo := &mockBenchRepository{}
	statsService := &mockBenchStatsService{}
	jobService := &mockBenchJobService{}
	lootboxService := &mockBenchLootboxService{}
	namingResolver := &mockBenchNamingResolver{}
	cooldownService := &mockBenchCooldownService{}

	service := NewService(repo, statsService, jobService, lootboxService, namingResolver, cooldownService, false)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := service.AddItem(ctx, "twitch", "bench-789", "itemuser", "money", 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkService_AddItem_NewItem benchmarks adding a completely new item
func BenchmarkService_AddItem_NewItem(b *testing.B) {
	repo := &mockBenchRepository{}
	statsService := &mockBenchStatsService{}
	jobService := &mockBenchJobService{}
	lootboxService := &mockBenchLootboxService{}
	namingResolver := &mockBenchNamingResolver{}
	cooldownService := &mockBenchCooldownService{}

	service := NewService(repo, statsService, jobService, lootboxService, namingResolver, cooldownService, false)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := service.AddItem(ctx, "twitch", "bench-new", "newitemuser", "sword", 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}
// Benchmark batch operations

func BenchmarkService_AddItems_Batch10(b *testing.B) {
	repo := &mockBenchRepository{}
	statsService := &mockBenchStatsService{}
	jobService := &mockBenchJobService{}
	lootboxService := &mockBenchLootboxService{}
	namingResolver := &mockBenchNamingResolver{}
	cooldownService := &mockBenchCooldownService{}

	service := NewService(repo, statsService, jobService, lootboxService, namingResolver, cooldownService, false)

	ctx := context.Background()

	// Simulate 10 items from lootbox opening
	items := map[string]int{
		"money":          5,
		"sword":          2,
		"lootbox_tier0":  1,
		"shield":         1,
		"potion":         1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := service.AddItems(ctx, "twitch", "bench-batch", "batchuser", items)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkService_AddItems_Batch25(b *testing.B) {
	repo := &mockBenchRepository{}
	statsService := &mockBenchStatsService{}
	jobService := &mockBenchJobService{}
	lootboxService := &mockBenchLootboxService{}
	namingResolver := &mockBenchNamingResolver{}
	cooldownService := &mockBenchCooldownService{}

	service := NewService(repo, statsService, jobService, lootboxService, namingResolver, cooldownService, false)

	ctx := context.Background()

	// Simulate 25 items from gamble (5 users × 5 lootboxes)
	items := map[string]int{
		"money":          15,
		"sword":          5,
		"lootbox_tier0":  3,
		"shield":         1,
		"potion":         1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := service.AddItems(ctx, "twitch", "bench-batch25", "gambleuser", items)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark comparison: Individual vs Batch
func BenchmarkService_AddItem_Individual10(b *testing.B) {
	repo := &mockBenchRepository{}
	statsService := &mockBenchStatsService{}
	jobService := &mockBenchJobService{}
	lootboxService := &mockBenchLootboxService{}
	namingResolver := &mockBenchNamingResolver{}
	cooldownService := &mockBenchCooldownService{}

	service := NewService(repo, statsService, jobService, lootboxService, namingResolver, cooldownService, false)

	ctx := context.Background()

	// Same 10 items as batch test, but added individually
	itemsList := []struct{
		name string
		qty int
	}{
		{"money", 5},
		{"sword", 2},
		{"lootbox_tier0", 1},
		{"shield", 1},
		{"potion", 1},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, item := range itemsList {
			err := service.AddItem(ctx, "twitch", "bench-indiv", "indivuser", item.name, item.qty)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
