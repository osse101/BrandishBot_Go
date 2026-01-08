package gamble_bench

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// --- Stubs (Zero-overhead mocks for benchmarking) ---

type StubRepository struct{}

func (s *StubRepository) CreateGamble(ctx context.Context, gamble *domain.Gamble) error { return nil }
func (s *StubRepository) GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error) {
	// Return a fresh object to simulate db fetch and allow state mutations safely
	// Initialize with bets to exercise the loop logic
	participants := make([]domain.Participant, 100)
	for i := 0; i < 100; i++ {
		participants[i] = domain.Participant{
			UserID:   uuid.NewString(),
			GambleID: id,
			LootboxBets: []domain.LootboxBet{
				{ItemID: 1, Quantity: 1},
			},
		}
	}

	return &domain.Gamble{
		ID:           id,
		State:        domain.GambleStateJoining,
		Participants: participants,
	}, nil
}
func (s *StubRepository) JoinGamble(ctx context.Context, participant *domain.Participant) error {
	return nil
}
func (s *StubRepository) UpdateGambleState(ctx context.Context, id uuid.UUID, state domain.GambleState) error {
	return nil
}
func (s *StubRepository) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error {
	return nil
}
func (s *StubRepository) CompleteGamble(ctx context.Context, result *domain.GambleResult) error {
	return nil
}
func (s *StubRepository) GetActiveGamble(ctx context.Context) (*domain.Gamble, error) {
	return nil, nil // No active gamble by default
}
func (s *StubRepository) BeginTx(ctx context.Context) (repository.Tx, error) {
	return &StubTx{}, nil
}
func (s *StubRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return &domain.Inventory{
		Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 1000}},
	}, nil
}
func (s *StubRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return nil
}
func (s *StubRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return &domain.User{ID: "stub-user"}, nil
}
func (s *StubRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	return &domain.Item{InternalName: "lootbox_common", ID: id}, nil
}

type StubTx struct{}

func (s *StubTx) Commit(ctx context.Context) error   { return nil }
func (s *StubTx) Rollback(ctx context.Context) error { return nil }
func (s *StubTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return &domain.Inventory{
		Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 1000}},
	}, nil
}
func (s *StubTx) UpdateInventory(ctx context.Context, userID string, inv domain.Inventory) error {
	return nil
}

type StubLootboxService struct{}

func (s *StubLootboxService) OpenLootbox(ctx context.Context, lootboxName string, quantity int) ([]lootbox.DroppedItem, error) {
	return []lootbox.DroppedItem{
		{ItemID: 101, Value: 10, Quantity: 1, ShineLevel: "COMMON"},
		{ItemID: 102, Value: 50, Quantity: 1, ShineLevel: "RARE"},
		{ItemID: 103, Value: 5, Quantity: 1, ShineLevel: "COMMON"},
	}, nil
}

type StubJobService struct{}

func (s *StubJobService) AwardXP(ctx context.Context, userID, jobKey string, amount int, source string, meta map[string]interface{}) (*domain.XPAwardResult, error) {
	return &domain.XPAwardResult{}, nil
}

type StubStatsService struct{}

func (s *StubStatsService) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error {
	return nil
}
func (s *StubStatsService) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) {
	return nil, nil
}
func (s *StubStatsService) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	return nil, nil
}
func (s *StubStatsService) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}

// StubBus implements event.Bus
type StubBus struct{}

func (b *StubBus) Publish(ctx context.Context, e event.Event) error { return nil }
func (b *StubBus) Subscribe(eventType event.Type, handler event.Handler) {}

// --- Benchmark Functions ---

// BenchmarkExecuteGamble_HighVolumeParticipants simulates a gamble execution with many participants.
func BenchmarkExecuteGamble_HighVolumeParticipants(b *testing.B) {
	repo := &StubRepository{}
	lbSvc := &StubLootboxService{}
	statsSvc := &StubStatsService{}
	jobSvc := &StubJobService{}
	bus := &StubBus{}

	// Create service with all dependencies stubbed (no nils)
	svc := gamble.NewService(repo, bus, lbSvc, statsSvc, time.Minute, jobSvc)

	gambleID := uuid.New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// The StubRepository.GetGamble returns a fresh "Joining" gamble every time,
		// allowing ExecuteGamble to proceed without state conflicts from previous iterations.
		_, err := svc.ExecuteGamble(ctx, gambleID)
		if err != nil {
			b.Fatalf("ExecuteGamble failed: %v", err)
		}
	}
}

// BenchmarkStartGamble simulates the overhead of starting a gamble.
func BenchmarkStartGamble(b *testing.B) {
	repo := &StubRepository{}
	lbSvc := &StubLootboxService{}
	statsSvc := &StubStatsService{}
	jobSvc := &StubJobService{}
	bus := &StubBus{}

	// Create service with all dependencies stubbed
	svc := gamble.NewService(repo, bus, lbSvc, statsSvc, time.Minute, jobSvc)

	ctx := context.Background()
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.StartGamble(ctx, "discord", "123456789", "User", bets)
		if err != nil {
			b.Fatalf("StartGamble failed: %v", err)
		}
	}
}
