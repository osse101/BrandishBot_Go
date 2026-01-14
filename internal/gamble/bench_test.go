package gamble

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// BenchRepository is a high-performance manual mock optimized for benchmarking
type BenchRepository struct {
	gamble *domain.Gamble
	users  map[string]*domain.User
	items  map[int]*domain.Item
}

func NewBenchRepository() *BenchRepository {
	repo := &BenchRepository{
		users: make(map[string]*domain.User),
		items: make(map[int]*domain.Item),
	}
	// Setup standard items
	repo.items[1] = &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	repo.items[10] = &domain.Item{ID: 10, InternalName: domain.ItemMoney}
	return repo
}

func (r *BenchRepository) CreateGamble(ctx context.Context, gamble *domain.Gamble) error {
	r.gamble = gamble
	return nil
}

func (r *BenchRepository) GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error) {
	return r.gamble, nil
}

func (r *BenchRepository) GetActiveGamble(ctx context.Context) (*domain.Gamble, error) {
	if r.gamble != nil && r.gamble.State == domain.GambleStateJoining {
		return r.gamble, nil
	}
	return nil, nil
}

func (r *BenchRepository) JoinGamble(ctx context.Context, participant *domain.Participant) error {
	// In a real scenario this appends to DB. Here we assume it's already in the gamble struct passed to CreateGamble
	// or updated in memory.
	return nil
}

func (r *BenchRepository) BeginTx(ctx context.Context) (repository.Tx, error) {
	return &BenchTx{repo: r}, nil
}

func (r *BenchRepository) BeginGambleTx(ctx context.Context) (repository.GambleTx, error) {
	return &BenchTx{repo: r}, nil
}

func (r *BenchRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
}

func (r *BenchRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return &domain.User{ID: "user-" + platformID}, nil
}

func (r *BenchRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	return r.items[id], nil
}

// Stubs for interface compliance
func (r *BenchRepository) UpdateGambleState(ctx context.Context, id uuid.UUID, state domain.GambleState) error {
	if r.gamble != nil {
		r.gamble.State = state
	}
	return nil
}
func (r *BenchRepository) UpdateGambleStateIfMatches(ctx context.Context, id uuid.UUID, expected, new domain.GambleState) (int64, error) {
	if r.gamble != nil && r.gamble.State == expected {
		r.gamble.State = new
		return 1, nil
	}
	return 0, nil
}
func (r *BenchRepository) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error { return nil }
func (r *BenchRepository) CompleteGamble(ctx context.Context, result *domain.GambleResult) error {
	if r.gamble != nil {
		r.gamble.State = domain.GambleStateCompleted
	}
	return nil
}
func (r *BenchRepository) UpdateInventory(ctx context.Context, userID string, inv domain.Inventory) error { return nil }

// BenchTx implements both Tx and GambleTx
type BenchTx struct {
	repo *BenchRepository
}

func (t *BenchTx) Commit(ctx context.Context) error   { return nil }
func (t *BenchTx) Rollback(ctx context.Context) error { return nil }
func (t *BenchTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return t.repo.GetInventory(ctx, userID)
}
func (t *BenchTx) UpdateInventory(ctx context.Context, userID string, inv domain.Inventory) error {
	return nil
}
func (t *BenchTx) UpdateGambleStateIfMatches(ctx context.Context, id uuid.UUID, expected, new domain.GambleState) (int64, error) {
	return t.repo.UpdateGambleStateIfMatches(ctx, id, expected, new)
}
func (t *BenchTx) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error { return nil }
func (t *BenchTx) CompleteGamble(ctx context.Context, result *domain.GambleResult) error {
	return t.repo.CompleteGamble(ctx, result)
}

// Interface compliance stubs for Tx

// BenchLootboxService
type BenchLootboxService struct{}

func (s *BenchLootboxService) OpenLootbox(ctx context.Context, name string, quantity int) ([]lootbox.DroppedItem, error) {
	// Deterministic drops: always 10 money per lootbox
	drops := make([]lootbox.DroppedItem, quantity)
	for i := 0; i < quantity; i++ {
		drops[i] = lootbox.DroppedItem{
			ItemID:   10,
			ItemName: domain.ItemMoney,
			Quantity: 10,
			Value:    100,
		}
	}
	return drops, nil
}

func (s *BenchLootboxService) Shutdown(ctx context.Context) error { return nil }

func setupBenchmarkGamble(repo *BenchRepository, participants int) uuid.UUID {
	id := uuid.New()
	gamble := &domain.Gamble{
		ID:           id,
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(-time.Hour), // Deadline passed, ready to execute
		Participants: make([]domain.Participant, participants),
	}

	for i := 0; i < participants; i++ {
		userID := "user-" + strconv.Itoa(i)
		gamble.Participants[i] = domain.Participant{
			UserID:   userID,
			Username: "User" + strconv.Itoa(i),
			LootboxBets: []domain.LootboxBet{
				{ItemID: 1, Quantity: 1},
			},
		}
	}
	repo.gamble = gamble
	return id
}

func BenchmarkExecuteGamble(b *testing.B) {
	participantCounts := []int{10, 100, 1000}

	for _, count := range participantCounts {
		b.Run(fmt.Sprintf("Participants_%d", count), func(b *testing.B) {
			repo := NewBenchRepository()
			// Need a fresh service for each run? No, service is stateless except for wg
			svc := NewService(repo, nil, &BenchLootboxService{}, nil, time.Minute, nil, nil)

			// Helper to reset state
			resetState := func() uuid.UUID {
				return setupBenchmarkGamble(repo, count)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Stop timer to setup/reset state
				b.StopTimer()
				id := resetState()
				b.StartTimer()

				_, err := svc.ExecuteGamble(context.Background(), id)
				if err != nil {
					b.Fatalf("ExecuteGamble failed: %v", err)
				}

				// Wait for async goroutines to prevent leak accumulation
				// In a benchmark this might be slow, but essential for correctness if we launch goroutines
				// The service uses svc.wg.Add(1) for XP awards.
				// Since we passed nil JobService, awardGamblerXP returns early but AFTER wg.Done().
				// Wait, awardGamblerXP calls wg.Done() via defer.
				// But we need to make sure we don't spawn millions of goroutines.
				// Actually, passing nil JobService makes awardGamblerXP return early, but it still does defer wg.Done().
				// So we should be fine.
			}

			// Ensure cleanup
			b.StopTimer()
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = svc.Shutdown(context.Background())
			}()
			wg.Wait()
		})
	}
}
