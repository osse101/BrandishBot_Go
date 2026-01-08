package gamble_test

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
	"github.com/stretchr/testify/mock"
)

// --- Mocks using testify/mock ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateGamble(ctx context.Context, gamble *domain.Gamble) error {
	return m.Called(ctx, gamble).Error(0)
}

func (m *MockRepository) GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error) {
	args := m.Called(ctx, id)
	// Support functional return for fresh objects in benchmarks
	if fn, ok := args.Get(0).(func(context.Context, uuid.UUID) *domain.Gamble); ok {
		return fn(ctx, id), args.Error(1)
	}
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Gamble), args.Error(1)
}

func (m *MockRepository) JoinGamble(ctx context.Context, participant *domain.Participant) error {
	return m.Called(ctx, participant).Error(0)
}

func (m *MockRepository) UpdateGambleState(ctx context.Context, id uuid.UUID, state domain.GambleState) error {
	return m.Called(ctx, id, state).Error(0)
}

func (m *MockRepository) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error {
	return m.Called(ctx, items).Error(0)
}

func (m *MockRepository) CompleteGamble(ctx context.Context, result *domain.GambleResult) error {
	return m.Called(ctx, result).Error(0)
}

func (m *MockRepository) GetActiveGamble(ctx context.Context) (*domain.Gamble, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Gamble), args.Error(1)
}

func (m *MockRepository) BeginTx(ctx context.Context) (repository.Tx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(repository.Tx), args.Error(1)
}

func (m *MockRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Inventory), args.Error(1)
}

func (m *MockRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return m.Called(ctx, userID, inventory).Error(0)
}

func (m *MockRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Item), args.Error(1)
}

type MockTx struct {
	mock.Mock
}

func (m *MockTx) Commit(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockTx) Rollback(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	args := m.Called(ctx, userID)
	if fn, ok := args.Get(0).(func(context.Context, string) *domain.Inventory); ok {
		return fn(ctx, userID), args.Error(1)
	}
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Inventory), args.Error(1)
}

func (m *MockTx) UpdateInventory(ctx context.Context, userID string, inv domain.Inventory) error {
	return m.Called(ctx, userID, inv).Error(0)
}

type MockLootboxService struct {
	mock.Mock
}

func (m *MockLootboxService) OpenLootbox(ctx context.Context, lootboxName string, quantity int) ([]lootbox.DroppedItem, error) {
	args := m.Called(ctx, lootboxName, quantity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]lootbox.DroppedItem), args.Error(1)
}

type MockJobService struct {
	mock.Mock
}

func (m *MockJobService) AwardXP(ctx context.Context, userID, jobKey string, amount int, source string, meta map[string]interface{}) (*domain.XPAwardResult, error) {
	// Minimal return for bench
	return &domain.XPAwardResult{}, nil
}

type MockStatsService struct {
	mock.Mock
}

func (m *MockStatsService) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error {
	return nil // Low overhead
}
func (m *MockStatsService) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) {
	return nil, nil
}
func (m *MockStatsService) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	return nil, nil
}
func (m *MockStatsService) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}

type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(ctx context.Context, e event.Event) error {
	return nil // Minimal overhead
}
func (m *MockEventBus) Subscribe(eventType event.Type, handler event.Handler) {}

// --- Benchmark Functions ---

func BenchmarkExecuteGamble_HighVolumeParticipants(b *testing.B) {
	repo := new(MockRepository)
	lbSvc := new(MockLootboxService)
	statsSvc := new(MockStatsService)
	jobSvc := new(MockJobService)
	bus := new(MockEventBus)
	tx := new(MockTx)

	svc := gamble.NewService(repo, bus, lbSvc, statsSvc, time.Minute, jobSvc)

	gambleID := uuid.New()
	ctx := context.Background()

	// Data Setup
	participants := make([]domain.Participant, 100)
	for i := 0; i < 100; i++ {
		participants[i] = domain.Participant{
			UserID:   uuid.NewString(),
			GambleID: gambleID,
			LootboxBets: []domain.LootboxBet{
				{ItemID: 1, Quantity: 1},
			},
		}
	}

	createGamble := func() *domain.Gamble {
		return &domain.Gamble{
			ID:           gambleID,
			State:        domain.GambleStateJoining,
			Participants: participants,
		}
	}

	// Expectations
	repo.On("GetGamble", mock.Anything, gambleID).Return(func(ctx context.Context, id uuid.UUID) *domain.Gamble {
		return createGamble()
	}, nil)

	repo.On("UpdateGambleState", mock.Anything, gambleID, domain.GambleStateOpening).Return(nil)
	repo.On("GetItemByID", mock.Anything, 1).Return(&domain.Item{InternalName: "lootbox_common"}, nil)

	// Single static drop to minimize RNG overhead in bench
	drops := []lootbox.DroppedItem{{ItemID: 101, Value: 10, Quantity: 1, ShineLevel: "COMMON"}}
	lbSvc.On("OpenLootbox", mock.Anything, "lootbox_common", 1).Return(drops, nil)

	repo.On("SaveOpenedItems", mock.Anything, mock.Anything).Return(nil)
	repo.On("BeginTx", mock.Anything).Return(tx, nil)

	// Return fresh inventory
	tx.On("GetInventory", mock.Anything, mock.Anything).Return(func(ctx context.Context, uid string) *domain.Inventory {
		return &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 100}}}
	}, nil)

	tx.On("UpdateInventory", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	tx.On("Commit", mock.Anything).Return(nil)
	tx.On("Rollback", mock.Anything).Return(nil)
	repo.On("CompleteGamble", mock.Anything, mock.Anything).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.ExecuteGamble(ctx, gambleID)
		if err != nil {
			b.Fatalf("ExecuteGamble failed: %v", err)
		}
	}
}

func BenchmarkStartGamble(b *testing.B) {
	repo := new(MockRepository)
	lbSvc := new(MockLootboxService)
	statsSvc := new(MockStatsService)
	jobSvc := new(MockJobService)
	bus := new(MockEventBus)
	tx := new(MockTx)

	svc := gamble.NewService(repo, bus, lbSvc, statsSvc, time.Minute, jobSvc)

	ctx := context.Background()
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}

	user := &domain.User{ID: "user-123"}

	repo.On("GetUserByPlatformID", mock.Anything, "discord", "123456789").Return(user, nil)
	repo.On("GetActiveGamble", mock.Anything).Return(nil, nil)
	repo.On("BeginTx", mock.Anything).Return(tx, nil)

	tx.On("GetInventory", mock.Anything, "user-123").Return(func(ctx context.Context, uid string) *domain.Inventory {
		return &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 100}}}
	}, nil)

	tx.On("UpdateInventory", mock.Anything, "user-123", mock.Anything).Return(nil)
	repo.On("CreateGamble", mock.Anything, mock.Anything).Return(nil)
	repo.On("JoinGamble", mock.Anything, mock.Anything).Return(nil)
	tx.On("Commit", mock.Anything).Return(nil)
	tx.On("Rollback", mock.Anything).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.StartGamble(ctx, "discord", "123456789", "User", bets)
		if err != nil {
			b.Fatalf("StartGamble failed: %v", err)
		}
	}
}
