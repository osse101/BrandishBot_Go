package gamble

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// BenchMockRepository is a lightweight manual mock for benchmarking
type BenchMockRepository struct {
	gamble *domain.Gamble
}

func (m *BenchMockRepository) GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error) {
	return m.gamble, nil
}

func (m *BenchMockRepository) BeginGambleTx(ctx context.Context) (repository.GambleTx, error) {
	return &BenchMockTx{}, nil
}

func (m *BenchMockRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	return &domain.Item{ID: id, InternalName: fmt.Sprintf("item_%d", id)}, nil
}

// Stubs for unused methods in this benchmark
func (m *BenchMockRepository) CreateGamble(ctx context.Context, gamble *domain.Gamble) error { return nil }
func (m *BenchMockRepository) JoinGamble(ctx context.Context, participant *domain.Participant) error { return nil }
func (m *BenchMockRepository) UpdateGambleState(ctx context.Context, id uuid.UUID, state domain.GambleState) error { return nil }
func (m *BenchMockRepository) UpdateGambleStateIfMatches(ctx context.Context, id uuid.UUID, expectedState, newState domain.GambleState) (int64, error) { return 1, nil }
func (m *BenchMockRepository) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error { return nil }
func (m *BenchMockRepository) CompleteGamble(ctx context.Context, result *domain.GambleResult) error { return nil }
func (m *BenchMockRepository) GetActiveGamble(ctx context.Context) (*domain.Gamble, error) { return nil, nil }
func (m *BenchMockRepository) BeginTx(ctx context.Context) (repository.Tx, error) { return nil, nil }
func (m *BenchMockRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) { return nil, nil }
func (m *BenchMockRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error { return nil }
func (m *BenchMockRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) { return nil, nil }

// BenchMockTx is a lightweight transaction mock
type BenchMockTx struct {}

func (m *BenchMockTx) UpdateGambleStateIfMatches(ctx context.Context, id uuid.UUID, expectedState, newState domain.GambleState) (int64, error) {
	return 1, nil
}
func (m *BenchMockTx) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error { return nil }
func (m *BenchMockTx) CompleteGamble(ctx context.Context, result *domain.GambleResult) error { return nil }
func (m *BenchMockTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return &domain.Inventory{}, nil
}
func (m *BenchMockTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error { return nil }
func (m *BenchMockTx) Commit(ctx context.Context) error { return nil }
func (m *BenchMockTx) Rollback(ctx context.Context) error { return nil }


// BenchMockLootboxService
type BenchMockLootboxService struct {}

func (m *BenchMockLootboxService) OpenLootbox(ctx context.Context, lootboxName string, quantity int) ([]lootbox.DroppedItem, error) {
	// Return fixed value to avoid RNG overhead in this benchmark if we only want to measure service logic
	return []lootbox.DroppedItem{
		{ItemID: 1, ItemName: "Gold", Quantity: 100, Value: 100},
	}, nil
}

// BenchMockStatsService
type BenchMockStatsService struct {}

func (m *BenchMockStatsService) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error {
	return nil
}
func (m *BenchMockStatsService) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) { return nil, nil }
func (m *BenchMockStatsService) GetUserCurrentStreak(ctx context.Context, userID string) (int, error) { return 0, nil }
func (m *BenchMockStatsService) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) { return nil, nil }
func (m *BenchMockStatsService) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) { return nil, nil }


func BenchmarkExecuteGamble(b *testing.B) {
	participantCounts := []int{5, 50, 100}

	for _, count := range participantCounts {
		b.Run(fmt.Sprintf("Participants-%d", count), func(b *testing.B) {
			participants := make([]domain.Participant, count)
			for i := 0; i < count; i++ {
				participants[i] = domain.Participant{
					UserID: fmt.Sprintf("user-%d", i),
					LootboxBets: []domain.LootboxBet{
						{ItemID: 1, Quantity: 1},
					},
				}
			}

			gamble := &domain.Gamble{
				ID:           uuid.New(),
				State:        domain.GambleStateJoining,
				Participants: participants,
			}

			repo := &BenchMockRepository{gamble: gamble}
			svc := NewService(repo, nil, &BenchMockLootboxService{}, &BenchMockStatsService{}, time.Minute, nil, nil)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// We need to reset the state potentially if the service modifies it in memory
				// But here we are passing a fresh context and the repo returns the same gamble object.
				// The service logic might fail if state is already Opening/Completed.
				// However, `ExecuteGamble` transitions state from Joining -> Opening.
				// If we reuse the same gamble object in memory, the second iteration will fail.
				// So we need to reset the gamble state in the loop.
				gamble.State = domain.GambleStateJoining

				_, err := svc.ExecuteGamble(ctx, gamble.ID)
				if err != nil {
					b.Fatalf("ExecuteGamble failed: %v", err)
				}
			}
		})
	}
}
