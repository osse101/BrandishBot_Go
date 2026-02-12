package user

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

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
type fakeBenchRepository struct{}

func (f *fakeBenchRepository) UpsertUser(ctx context.Context, user *domain.User) error {
	return nil
}

func (f *fakeBenchRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	// Simulate cache miss for new user benchmark
	return nil, domain.ErrUserNotFound
}

func (f *fakeBenchRepository) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (f *fakeBenchRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	return &domain.User{ID: userID, Username: "benchuser"}, nil
}

func (f *fakeBenchRepository) UpdateUser(ctx context.Context, user domain.User) error {
	return nil
}

func (f *fakeBenchRepository) DeleteUser(ctx context.Context, userID string) error {
	return nil
}

func (f *fakeBenchRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 100}, // money
		},
	}, nil
}

func (f *fakeBenchRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return nil
}

func (f *fakeBenchRepository) DeleteInventory(ctx context.Context, userID string) error {
	return nil
}

func (f *fakeBenchRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	return &domain.Item{
		ID:           42,
		InternalName: itemName,
		Description:  "Benchmark item",
		BaseValue:    10,
	}, nil
}

func (f *fakeBenchRepository) GetItemsByNames(ctx context.Context, names []string) ([]domain.Item, error) {
	items := make([]domain.Item, len(names))
	for i, name := range names {
		items[i] = domain.Item{
			ID:           42 + i,
			InternalName: name,
			Description:  "Benchmark item",
			BaseValue:    10,
		}
	}
	return items, nil
}

func (f *fakeBenchRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	return &domain.Item{
		ID:           id,
		InternalName: "bench_item",
		Description:  "Benchmark item",
		BaseValue:    10,
	}, nil
}

func (f *fakeBenchRepository) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
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

func (f *fakeBenchRepository) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	return nil, nil
}

func (f *fakeBenchRepository) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	return false, nil
}

func (f *fakeBenchRepository) GetActiveTrap(ctx context.Context, targetID uuid.UUID) (*domain.Trap, error) {
	return nil, nil
}

func (f *fakeBenchRepository) GetActiveTrapForUpdate(ctx context.Context, targetID uuid.UUID) (*domain.Trap, error) {
	return nil, nil
}

func (f *fakeBenchRepository) TriggerTrap(ctx context.Context, trapID uuid.UUID) error {
	return nil
}

func (f *fakeBenchRepository) CreateTrap(ctx context.Context, trap *domain.Trap) error {
	return nil
}

func (f *fakeBenchRepository) GetTrapsByUser(ctx context.Context, setterID uuid.UUID, limit int) ([]*domain.Trap, error) {
	return nil, nil
}

func (f *fakeBenchRepository) GetTriggeredTrapsForTarget(ctx context.Context, targetID uuid.UUID, limit int) ([]*domain.Trap, error) {
	return nil, nil
}

func (f *fakeBenchRepository) CleanupStaleTraps(ctx context.Context, daysOld int) (int, error) {
	return 0, nil
}

func (f *fakeBenchRepository) BeginTx(ctx context.Context) (repository.UserTx, error) {
	return &fakeBenchTx{repo: f}, nil
}

func (f *fakeBenchRepository) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	return nil, nil
}

func (f *fakeBenchRepository) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	return false, nil
}

func (f *fakeBenchRepository) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	return nil
}

func (f *fakeBenchRepository) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]repository.UnlockedRecipeInfo, error) {
	return nil, nil
}

func (f *fakeBenchRepository) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	return nil, nil
}

func (f *fakeBenchRepository) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	return nil
}

func (f *fakeBenchRepository) MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, mergedUser domain.User, mergedInventory domain.Inventory) error {
	return nil // No-op
}

func (f *fakeBenchRepository) GetAllItems(ctx context.Context) ([]domain.Item, error) {
	return []domain.Item{}, nil // Bench mock doesn't store items
}

func (f *fakeBenchRepository) GetRecentlyActiveUsers(ctx context.Context, limit int) ([]domain.User, error) {
	return []domain.User{}, nil
}

// Mock transaction
type fakeBenchTx struct {
	repo *fakeBenchRepository
}

func (f *fakeBenchTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return f.repo.GetInventory(ctx, userID)
}

func (f *fakeBenchTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return f.repo.UpdateInventory(ctx, userID, inventory)
}

func (f *fakeBenchTx) Commit(ctx context.Context) error {
	return nil
}

func (f *fakeBenchTx) Rollback(ctx context.Context) error {
	return nil
}

// Mock stats service
type fakeBenchStatsService struct{}

func (f *fakeBenchStatsService) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error {
	return nil
}

func (f *fakeBenchStatsService) GetUserStats(ctx context.Context, userID, period string) (*domain.StatsSummary, error) {
	return nil, nil
}

func (f *fakeBenchStatsService) GetUserCurrentStreak(ctx context.Context, userID string) (int, error) {
	return 0, nil
}

func (f *fakeBenchStatsService) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	return nil, nil
}

func (f *fakeBenchStatsService) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}

func (f *fakeBenchStatsService) GetUserSlotsStats(ctx context.Context, userID, period string) (*domain.SlotsStats, error) {
	return nil, nil
}

func (f *fakeBenchStatsService) GetSlotsLeaderboardByProfit(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error) {
	return nil, nil
}

func (f *fakeBenchStatsService) GetSlotsLeaderboardByWinRate(ctx context.Context, period string, minSpins, limit int) ([]domain.SlotsStats, error) {
	return nil, nil
}

func (f *fakeBenchStatsService) GetSlotsLeaderboardByMegaJackpots(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error) {
	return nil, nil
}

// Mock lootbox service
type fakeBenchLootboxService struct{}

func (f *fakeBenchLootboxService) OpenLootbox(ctx context.Context, lootboxName string, quantity int, boxQuality domain.QualityLevel) ([]lootbox.DroppedItem, error) {
	return []lootbox.DroppedItem{
		{ItemID: 1, ItemName: "money", Quantity: 10, Value: 10, QualityLevel: domain.QualityCommon},
	}, nil
}

// Mock naming resolver
type fakeBenchNamingResolver struct{}

func (f *fakeBenchNamingResolver) ResolvePublicName(publicName string) (string, bool) {
	return publicName, true
}

func (f *fakeBenchNamingResolver) GetDisplayName(internalName string, qualityLevel domain.QualityLevel) string {
	return internalName
}

func (f *fakeBenchNamingResolver) GetActiveTheme() string {
	return ""
}

func (f *fakeBenchNamingResolver) Reload() error {
	return nil
}

func (f *fakeBenchNamingResolver) RegisterItem(internalName, publicName string) {
	// no-op
}

// Mock cooldown service
type fakeBenchCooldownService struct{}

func (f *fakeBenchCooldownService) CheckCooldown(ctx context.Context, userID, action string) (bool, time.Duration, error) {
	return false, 0, nil
}

func (f *fakeBenchCooldownService) EnforceCooldown(ctx context.Context, userID, action string, fn func() error) error {
	return fn()
}

func (f *fakeBenchCooldownService) ResetCooldown(ctx context.Context, userID, action string) error {
	return nil
}

func (f *fakeBenchCooldownService) GetLastUsed(ctx context.Context, userID, action string) (*time.Time, error) {
	return nil, nil
}

// BenchmarkService_HandleIncomingMessage benchmarks user lookup/creation
func BenchmarkService_HandleIncomingMessage(b *testing.B) {
	repo := &fakeBenchRepository{}
	statsService := &fakeBenchStatsService{}
	lootboxService := &fakeBenchLootboxService{}
	namingResolver := &fakeBenchNamingResolver{}
	cooldownService := &fakeBenchCooldownService{}

	service := NewService(repo, repo, statsService, nil, lootboxService, namingResolver, cooldownService, nil, false)

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
	repo := &fakeBenchRepository{}
	statsService := &fakeBenchStatsService{}
	lootboxService := &fakeBenchLootboxService{}
	namingResolver := &fakeBenchNamingResolver{}
	cooldownService := &fakeBenchCooldownService{}

	service := NewService(repo, repo, statsService, nil, lootboxService, namingResolver, cooldownService, nil, false)

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
	repo := &fakeBenchRepository{}
	statsService := &fakeBenchStatsService{}
	lootboxService := &fakeBenchLootboxService{}
	namingResolver := &fakeBenchNamingResolver{}
	cooldownService := &fakeBenchCooldownService{}

	service := NewService(repo, repo, statsService, nil, lootboxService, namingResolver, cooldownService, nil, false)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := service.AddItemByUsername(ctx, "twitch", "itemuser", "money", 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkService_AddItem_NewItem benchmarks adding a completely new item
func BenchmarkService_AddItem_NewItem(b *testing.B) {
	repo := &fakeBenchRepository{}
	statsService := &fakeBenchStatsService{}
	lootboxService := &fakeBenchLootboxService{}
	namingResolver := &fakeBenchNamingResolver{}
	cooldownService := &fakeBenchCooldownService{}

	service := NewService(repo, repo, statsService, nil, lootboxService, namingResolver, cooldownService, nil, false)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := service.AddItemByUsername(ctx, "twitch", "bench-new", "sword", 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark comparison: Individual vs Batch
func BenchmarkService_AddItem_Individual10(b *testing.B) {
	repo := &fakeBenchRepository{}
	statsService := &fakeBenchStatsService{}
	lootboxService := &fakeBenchLootboxService{}
	namingResolver := &fakeBenchNamingResolver{}
	cooldownService := &fakeBenchCooldownService{}

	service := NewService(repo, repo, statsService, nil, lootboxService, namingResolver, cooldownService, nil, false)

	ctx := context.Background()

	// Same 10 items as batch test, but added individually
	itemsList := []struct {
		name string
		qty  int
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
			err := service.AddItemByUsername(ctx, domain.PlatformTwitch, "indivuser", item.name, item.qty)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
