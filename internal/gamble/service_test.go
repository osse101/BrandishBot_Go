package gamble

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// MockRepository
type MockRepository struct {
	mock.Mock
}

// GetItemByName implements [repository.Gamble].
func (m *MockRepository) GetItemByName(ctx context.Context, name string) (*domain.Item, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Item), args.Error(1)
}

func (m *MockRepository) CreateGamble(ctx context.Context, gamble *domain.Gamble) error {
	args := m.Called(ctx, gamble)
	return args.Error(0)
}

func (m *MockRepository) GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Gamble), args.Error(1)
}

func (m *MockRepository) JoinGamble(ctx context.Context, participant *domain.Participant) error {
	args := m.Called(ctx, participant)
	return args.Error(0)
}

func (m *MockRepository) UpdateGambleState(ctx context.Context, id uuid.UUID, state domain.GambleState) error {
	args := m.Called(ctx, id, state)
	return args.Error(0)
}

func (m *MockRepository) UpdateGambleStateIfMatches(ctx context.Context, id uuid.UUID, expectedState, newState domain.GambleState) (int64, error) {
	args := m.Called(ctx, id, expectedState, newState)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockRepository) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

func (m *MockRepository) CompleteGamble(ctx context.Context, result *domain.GambleResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
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

func (m *MockRepository) BeginGambleTx(ctx context.Context) (repository.GambleTx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(repository.GambleTx), args.Error(1)
}

func (m *MockRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Inventory), args.Error(1)
}

func (m *MockRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	args := m.Called(ctx, userID, inventory)
	return args.Error(0)
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

// MockLootboxService
type MockLootboxService struct {
	mock.Mock
}

func (m *MockLootboxService) OpenLootbox(ctx context.Context, lootboxName string, quantity int, boxShine string) ([]lootbox.DroppedItem, error) {
	args := m.Called(ctx, lootboxName, quantity, boxShine)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]lootbox.DroppedItem), args.Error(1)
}

// MockTx
type MockTx struct {
	mock.Mock
}

func (m *MockTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Inventory), args.Error(1)
}

func (m *MockTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	args := m.Called(ctx, userID, inventory)
	return args.Error(0)
}

func (m *MockTx) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTx) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTx) UpdateGambleStateIfMatches(ctx context.Context, id uuid.UUID, expectedState, newState domain.GambleState) (int64, error) {
	args := m.Called(ctx, id, expectedState, newState)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTx) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

func (m *MockTx) CompleteGamble(ctx context.Context, result *domain.GambleResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

// MockStatsService
type MockStatsService struct {
	mock.Mock
}

func (m *MockStatsService) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error {
	args := m.Called(ctx, userID, eventType, metadata)
	return args.Error(0)
}

func (m *MockStatsService) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) {
	args := m.Called(ctx, userID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StatsSummary), args.Error(1)
}

func (m *MockStatsService) GetUserCurrentStreak(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockStatsService) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	args := m.Called(ctx, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StatsSummary), args.Error(1)
}

func (m *MockStatsService) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	args := m.Called(ctx, eventType, period, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.LeaderboardEntry), args.Error(1)
}

// MockJobService
type MockJobService struct {
	mock.Mock
}

func (m *MockJobService) AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error) {
	args := m.Called(ctx, userID, jobKey, baseAmount, source, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.XPAwardResult), args.Error(1)
}

// ========================================
// StartGamble Tests
// ========================================

func TestStartGamble_Success(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	user := &domain.User{ID: "user1"}
	bets := []domain.LootboxBet{{ItemName: "lootbox_tier1", Quantity: 2}}
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 5}}}
	tx := new(MockTx)

	repo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil)
	repo.On("GetInventory", ctx, "user1").Return(inventory, nil)
	repo.On("GetActiveGamble", ctx).Return(nil, nil)

	// Item validation
	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	repo.On("GetItemByName", ctx, "lootbox_tier1").Return(lootboxItem, nil)

	repo.On("BeginTx", ctx).Return(tx, nil)
	tx.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()
	repo.On("CreateGamble", ctx, mock.Anything).Return(nil)
	repo.On("JoinGamble", ctx, mock.Anything).Return(nil)

	gamble, err := s.StartGamble(ctx, domain.PlatformTwitch, "123", "testuser", bets)

	assert.NoError(t, err)
	assert.NotNil(t, gamble)
	assert.Equal(t, domain.GambleStateJoining, gamble.State)
	assert.Equal(t, "user1", gamble.InitiatorID)
	assert.True(t, gamble.JoinDeadline.After(time.Now()))
	repo.AssertExpectations(t)
	tx.AssertExpectations(t)
}

func TestStartGamble_NoBets(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	bets := []domain.LootboxBet{}

	gamble, err := s.StartGamble(ctx, domain.PlatformTwitch, "123", "testuser", bets)

	assert.Error(t, err)
	assert.Nil(t, gamble)
	assert.ErrorIs(t, err, domain.ErrAtLeastOneLootboxRequired)
}

func TestStartGamble_InvalidBetQuantity(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	bets := []domain.LootboxBet{{ItemName: "lootbox_tier1", Quantity: 0}}

	gamble, err := s.StartGamble(ctx, domain.PlatformTwitch, "123", "testuser", bets)

	assert.Error(t, err)
	assert.Nil(t, gamble)
	assert.ErrorIs(t, err, domain.ErrBetQuantityMustBePositive)
}

func TestStartGamble_UserNotFound(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	bets := []domain.LootboxBet{{ItemName: "lootbox_tier1", Quantity: 1}}

	repo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(nil, nil)

	gamble, err := s.StartGamble(ctx, domain.PlatformTwitch, "123", "testuser", bets)

	assert.Error(t, err)
	assert.Nil(t, gamble)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
	repo.AssertExpectations(t)
}

func TestStartGamble_ActiveGambleExists(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	user := &domain.User{ID: "user1"}
	bets := []domain.LootboxBet{{ItemName: "lootbox_tier1", Quantity: 1}}
	activeGamble := &domain.Gamble{ID: uuid.New(), State: domain.GambleStateJoining}

	repo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil)
	repo.On("GetActiveGamble", ctx).Return(activeGamble, nil)

	gamble, err := s.StartGamble(ctx, domain.PlatformTwitch, "123", "testuser", bets)

	assert.Error(t, err)
	assert.Nil(t, gamble)
	assert.ErrorIs(t, err, domain.ErrGambleAlreadyActive)
	repo.AssertExpectations(t)
}

func TestStartGamble_InsufficientLootboxes(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	user := &domain.User{ID: "user1"}
	bets := []domain.LootboxBet{{ItemName: "lootbox_tier1", Quantity: 5}}
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 2}}}
	repo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil)
	repo.On("GetActiveGamble", ctx).Return(nil, nil)

	// Item validation
	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	repo.On("GetItemByName", ctx, "lootbox_tier1").Return(lootboxItem, nil)
	repo.On("GetInventory", ctx, "user1").Return(inventory, nil)

	gamble, err := s.StartGamble(ctx, domain.PlatformTwitch, "123", "testuser", bets)

	assert.Error(t, err)
	assert.Nil(t, gamble)
	assert.ErrorIs(t, err, domain.ErrInsufficientQuantity)
	repo.AssertExpectations(t)
}

func TestStartGamble_LootboxNotInInventory(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	user := &domain.User{ID: "user1"}
	bets := []domain.LootboxBet{{ItemName: "lootbox_tier1", Quantity: 1}}
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 5}}}
	repo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil)
	repo.On("GetActiveGamble", ctx).Return(nil, nil)

	// Item validation - testing with non-existent item ID
	nonExistentItem := &domain.Item{ID: 99, InternalName: domain.ItemLootbox2}
	repo.On("GetItemByName", ctx, "lootbox_tier1").Return(nonExistentItem, nil)
	repo.On("GetInventory", ctx, "user1").Return(inventory, nil)

	gamble, err := s.StartGamble(ctx, domain.PlatformTwitch, "123", "testuser", bets)

	assert.Error(t, err)
	assert.Nil(t, gamble)
	assert.ErrorIs(t, err, domain.ErrItemNotFound)
	repo.AssertExpectations(t)
}

// ========================================
// JoinGamble Tests
// ========================================

func TestJoinGamble_Success(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user2"}
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 3}}}
	gamble := &domain.Gamble{
		ID:           gambleID,
		InitiatorID:  "initiator_user",
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(time.Minute),
		Participants: []domain.Participant{
			{UserID: "initiator_user", GambleID: gambleID, LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		},
	}
	tx := new(MockTx)

	repo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "456").Return(user, nil)
	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	// Item validation
	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	repo.On("GetItemByName", ctx, "lootbox_tier1").Return(lootboxItem, nil)

	repo.On("BeginTx", ctx).Return(tx, nil)
	tx.On("GetInventory", ctx, "user2").Return(inventory, nil)
	tx.On("UpdateInventory", ctx, "user2", mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()
	repo.On("JoinGamble", ctx, mock.Anything).Return(nil)

	err := s.JoinGamble(ctx, gambleID, domain.PlatformTwitch, "456", "joiner")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	tx.AssertExpectations(t)
}

func TestJoinGamble_GambleNotFound(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user2"}
	repo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "456").Return(user, nil)
	repo.On("GetGamble", ctx, gambleID).Return(nil, nil)

	err := s.JoinGamble(ctx, gambleID, domain.PlatformTwitch, "456", "joiner")

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrGambleNotFound)
	repo.AssertExpectations(t)
}

func TestJoinGamble_WrongState(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user2"}
	gamble := &domain.Gamble{
		ID:           gambleID,
		InitiatorID:  "initiator_user",
		State:        domain.GambleStateOpening,
		JoinDeadline: time.Now().Add(time.Minute),
		Participants: []domain.Participant{
			{UserID: "initiator_user", GambleID: gambleID, LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		},
	}

	repo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "456").Return(user, nil)
	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	err := s.JoinGamble(ctx, gambleID, domain.PlatformTwitch, "456", "joiner")

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotInJoiningState)
	repo.AssertExpectations(t)
}

func TestJoinGamble_DeadlinePassed(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user2"}
	gamble := &domain.Gamble{
		ID:           gambleID,
		InitiatorID:  "initiator_user",
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(-time.Minute), // Past deadline
		Participants: []domain.Participant{
			{UserID: "initiator_user", GambleID: gambleID, LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		},
	}

	repo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "456").Return(user, nil)
	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	err := s.JoinGamble(ctx, gambleID, domain.PlatformTwitch, "456", "joiner")

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrJoinDeadlinePassed)
	repo.AssertExpectations(t)
}

func TestJoinGamble_InsufficientLootboxes(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user2"}
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 2}}}
	gamble := &domain.Gamble{
		ID:           gambleID,
		InitiatorID:  "initiator_user",
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(time.Minute),
		Participants: []domain.Participant{
			{UserID: "initiator_user", GambleID: gambleID, LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 10}}},
		},
	}
	tx := new(MockTx)

	repo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "456").Return(user, nil)
	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	// Item validation
	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem, nil)

	repo.On("BeginTx", ctx).Return(tx, nil)
	tx.On("GetInventory", ctx, "user2").Return(inventory, nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()

	err := s.JoinGamble(ctx, gambleID, domain.PlatformTwitch, "456", "joiner")

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInsufficientQuantity)
	repo.AssertExpectations(t)
	tx.AssertExpectations(t)
}

// ========================================
// ExecuteGamble Tests
// ========================================

func TestExecuteGamble_Success(t *testing.T) {
	repo := new(MockRepository)
	lootboxSvc := new(MockLootboxService)
	s := NewService(repo, nil, nil, lootboxSvc, nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		},
	}
	tx := new(MockTx)
	winnerInventory := &domain.Inventory{Slots: []domain.InventorySlot{}}
	lootboxItem := &domain.Item{ID: 1, InternalName: domain.PublicNameLootbox}
	droppedItems := []lootbox.DroppedItem{{ItemID: 10, ItemName: domain.ItemMoney, Quantity: 5, Value: 10}}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	repo.On("BeginGambleTx", ctx).Return(tx, nil)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(1), nil)
	repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem, nil)
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)
	lootboxSvc.On("OpenLootbox", ctx, mock.Anything, mock.Anything, mock.Anything).Return(droppedItems, nil)
	tx.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)
	tx.On("GetInventory", ctx, "user1").Return(winnerInventory, nil)
	tx.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
	tx.On("CompleteGamble", ctx, mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()

	result, err := s.ExecuteGamble(ctx, gambleID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "user1", result.WinnerID)
	assert.True(t, result.TotalValue > 0)
	repo.AssertExpectations(t)
	tx.AssertExpectations(t)
	lootboxSvc.AssertExpectations(t)
}

func TestExecuteGamble_MultipleParticipants(t *testing.T) {
	repo := new(MockRepository)
	lootboxSvc := new(MockLootboxService)
	s := NewService(repo, nil, nil, lootboxSvc, nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 2}}},
			{UserID: "user2", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		},
	}
	tx := new(MockTx)
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{}}
	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	droppedItems := []lootbox.DroppedItem{{ItemID: 10, ItemName: domain.ItemMoney, Quantity: 5, Value: 10}}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	repo.On("BeginGambleTx", ctx).Return(tx, nil)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(1), nil)
	repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem, nil)
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)
	lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox1, mock.Anything, mock.Anything).Return(droppedItems, nil)
	tx.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)
	tx.On("GetInventory", ctx, mock.Anything).Return(inventory, nil)
	tx.On("UpdateInventory", ctx, mock.Anything, mock.Anything).Return(nil)
	tx.On("CompleteGamble", ctx, mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()

	result, err := s.ExecuteGamble(ctx, gambleID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.WinnerID)
	assert.True(t, result.WinnerID == "user1" || result.WinnerID == "user2")
	repo.AssertExpectations(t)
	tx.AssertExpectations(t)
	lootboxSvc.AssertExpectations(t)
}

func TestExecuteGamble_GambleNotFound(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()

	repo.On("GetGamble", ctx, gambleID).Return(nil, nil)

	result, err := s.ExecuteGamble(ctx, gambleID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrGambleNotFound)
	repo.AssertExpectations(t)
}

func TestExecuteGamble_AlreadyCompleted(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateCompleted,
	}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	result, err := s.ExecuteGamble(ctx, gambleID)

	assert.NoError(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestExecuteGamble_WrongState(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateRefunded,
	}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	result, err := s.ExecuteGamble(ctx, gambleID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotInJoiningState)
	repo.AssertExpectations(t)
}

func TestExecuteGamble_StateUpdateFails(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		},
	}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	tx := new(MockTx)
	repo.On("BeginGambleTx", ctx).Return(tx, nil)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(0), errors.New("database error"))
	tx.On("Rollback", ctx).Return(nil).Maybe()

	result, err := s.ExecuteGamble(ctx, gambleID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), domain.ErrMsgFailedToTransitionState)
	repo.AssertExpectations(t)
	tx.AssertExpectations(t)
}

func TestExecuteGamble_SaveOpenedItemsFails(t *testing.T) {
	repo := new(MockRepository)
	lootboxSvc := new(MockLootboxService)
	s := NewService(repo, nil, nil, lootboxSvc, nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		},
	}
	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	droppedItems := []lootbox.DroppedItem{{ItemID: 10, ItemName: domain.ItemMoney, Quantity: 5, Value: 10}}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	tx := new(MockTx)
	repo.On("BeginGambleTx", ctx).Return(tx, nil)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(1), nil)
	repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem, nil)
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)
	lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox1, mock.Anything, mock.Anything).Return(droppedItems, nil)
	tx.On("SaveOpenedItems", ctx, mock.Anything).Return(domain.ErrDatabaseError)
	tx.On("Rollback", ctx).Return(nil).Maybe()

	result, err := s.ExecuteGamble(ctx, gambleID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), domain.ErrMsgFailedToSaveOpenedItems)
	repo.AssertExpectations(t)
	tx.AssertExpectations(t)
	lootboxSvc.AssertExpectations(t)
}

// ========================================
// GetGamble Tests
// ========================================

func TestGetGamble_Success(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	expectedGamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
	}

	repo.On("GetGamble", ctx, gambleID).Return(expectedGamble, nil)

	gamble, err := s.GetGamble(ctx, gambleID)

	assert.NoError(t, err)
	assert.Equal(t, expectedGamble, gamble)
	repo.AssertExpectations(t)
}

// ========================================
// GetActiveGamble Tests
// ========================================

func TestGetActiveGamble_Success(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	expectedGamble := &domain.Gamble{
		ID:    uuid.New(),
		State: domain.GambleStateJoining,
	}

	repo.On("GetActiveGamble", ctx).Return(expectedGamble, nil)

	gamble, err := s.GetActiveGamble(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expectedGamble, gamble)
	repo.AssertExpectations(t)
}

func TestGetActiveGamble_NoActiveGamble(t *testing.T) {
	repo := new(MockRepository)
	s := NewService(repo, nil, nil, new(MockLootboxService), nil, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()

	repo.On("GetActiveGamble", ctx).Return(nil, nil)

	gamble, err := s.GetActiveGamble(ctx)

	assert.NoError(t, err)
	assert.Nil(t, gamble)
	repo.AssertExpectations(t)
}

func TestExecuteGamble_NearMiss(t *testing.T) {
	repo := new(MockRepository)
	lootboxSvc := new(MockLootboxService)
	statsSvc := new(MockStatsService)
	// Passing nil for JobService
	s := NewService(repo, nil, nil, lootboxSvc, statsSvc, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()

	// Setup 2 participants
	// User1: Wins with 100
	// User2: Loses with 95 (Within 5%, should trigger NearMiss)
	// User3: Loses with 50 (Should NOT trigger NearMiss)

	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox0, Quantity: 1}}},
			{UserID: "user2", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
			{UserID: "user3", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox2, Quantity: 1}}},
		},
	}

	lootboxItem1 := &domain.Item{ID: 1, InternalName: domain.ItemLootbox0}
	lootboxItem2 := &domain.Item{ID: 2, InternalName: domain.ItemLootbox1}
	lootboxItem3 := &domain.Item{ID: 3, InternalName: domain.ItemLootbox2}

	// Mocks for lootbox drops
	drops1 := []lootbox.DroppedItem{{ItemID: 10, ItemName: domain.ItemMoney, Quantity: 1, Value: 100}}
	drops2 := []lootbox.DroppedItem{{ItemID: 11, ItemName: domain.ItemMoney, Quantity: 1, Value: 95}}
	drops3 := []lootbox.DroppedItem{{ItemID: 12, ItemName: domain.ItemMoney, Quantity: 1, Value: 50}}

	// Setup Repo expectations
	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	// Transaction for state update and item operations
	tx := new(MockTx)
	repo.On("BeginGambleTx", ctx).Return(tx, nil)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(1), nil)

	repo.On("GetItemByName", ctx, domain.ItemLootbox0).Return(lootboxItem1, nil)
	repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem2, nil)
	repo.On("GetItemByName", ctx, domain.ItemLootbox2).Return(lootboxItem3, nil)
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem1, nil)
	repo.On("GetItemByID", ctx, 2).Return(lootboxItem2, nil)
	repo.On("GetItemByID", ctx, 3).Return(lootboxItem3, nil)

	lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox0, 1, mock.Anything).Return(drops1, nil)
	lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox1, 1, mock.Anything).Return(drops2, nil)
	lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox2, 1, mock.Anything).Return(drops3, nil)

	tx.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)
	tx.On("GetInventory", ctx, "user1").Return(&domain.Inventory{}, nil)
	tx.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
	tx.On("CompleteGamble", ctx, mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()

	// Expect NearMiss event for User2
	statsSvc.On("RecordUserEvent", ctx, "user2", domain.EventGambleNearMiss, mock.MatchedBy(func(m map[string]interface{}) bool {
		return m["winner_score"] == int64(100) && m["score"] == int64(95)
	})).Return(nil)

	// Should NOT expect NearMiss for User3 (50 is < 95)

	result, err := s.ExecuteGamble(ctx, gambleID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "user1", result.WinnerID)

	repo.AssertExpectations(t)
	lootboxSvc.AssertExpectations(t)
	statsSvc.AssertExpectations(t)
}

func TestExecuteGamble_CriticalFailure(t *testing.T) {
	repo := new(MockRepository)
	lootboxSvc := new(MockLootboxService)
	statsSvc := new(MockStatsService)
	s := NewService(repo, nil, nil, lootboxSvc, statsSvc, time.Minute, nil, nil, nil, nil)

	ctx := context.Background()
	gambleID := uuid.New()

	// Setup 3 participants
	// User1: 100
	// User2: 100
	// User3: 10 (Avg = 70. Threshold = 14. 10 <= 14 => Critical Fail)

	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox0, Quantity: 1}}},
			{UserID: "user2", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
			{UserID: "user3", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox2, Quantity: 1}}},
		},
	}

	lootboxItem1 := &domain.Item{ID: 1, InternalName: domain.ItemLootbox0}
	lootboxItem2 := &domain.Item{ID: 2, InternalName: domain.ItemLootbox1}
	lootboxItem3 := &domain.Item{ID: 3, InternalName: domain.ItemLootbox2}

	drops1 := []lootbox.DroppedItem{{ItemID: 10, ItemName: domain.ItemMoney, Quantity: 1, Value: 100}}
	drops2 := []lootbox.DroppedItem{{ItemID: 11, ItemName: domain.ItemMoney, Quantity: 1, Value: 100}}
	drops3 := []lootbox.DroppedItem{{ItemID: 12, ItemName: domain.ItemMoney, Quantity: 1, Value: 10}}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	// Transaction for state update and item operations
	tx := new(MockTx)
	repo.On("BeginGambleTx", ctx).Return(tx, nil)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(1), nil)

	repo.On("GetItemByName", ctx, domain.ItemLootbox0).Return(lootboxItem1, nil)
	repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem2, nil)
	repo.On("GetItemByName", ctx, domain.ItemLootbox2).Return(lootboxItem3, nil)
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem1, nil)
	repo.On("GetItemByID", ctx, 2).Return(lootboxItem2, nil)
	repo.On("GetItemByID", ctx, 3).Return(lootboxItem3, nil)

	lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox0, 1, mock.Anything).Return(drops1, nil)
	lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox1, 1, mock.Anything).Return(drops2, nil)
	lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox2, 1, mock.Anything).Return(drops3, nil)

	tx.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)
	tx.On("GetInventory", ctx, mock.Anything).Return(&domain.Inventory{}, nil)
	tx.On("UpdateInventory", ctx, mock.Anything, mock.Anything).Return(nil)
	tx.On("CompleteGamble", ctx, mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()

	// Expect Critical Fail for User3
	statsSvc.On("RecordUserEvent", ctx, "user3", domain.EventGambleCriticalFail, mock.MatchedBy(func(m map[string]interface{}) bool {
		avg := m["average_score"].(float64)
		threshold := m["threshold"].(int64)
		score := m["score"].(int64)
		return score == 10 && threshold == 14 && avg == 70.0
	})).Return(nil)

	// We might also get TieBreakLost event for the loser of the tie break (User1 or User2).
	// We should allow it.
	statsSvc.On("RecordUserEvent", ctx, mock.Anything, domain.EventGambleTieBreakLost, mock.Anything).Return(nil).Maybe()

	result, err := s.ExecuteGamble(ctx, gambleID)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	repo.AssertExpectations(t)
	lootboxSvc.AssertExpectations(t)
	statsSvc.AssertExpectations(t)
}

func TestExecuteGamble_TieBreak(t *testing.T) {
	repo := new(MockRepository)
	lootboxSvc := new(MockLootboxService)
	statsSvc := new(MockStatsService)

	// Deterministic RNG: always pick index 1
	// With users "userA" and "userB", sorted order is ["userA", "userB"].
	// Index 1 is "userB".
	mockRng := func(n int) int { return 1 }

	s := NewService(repo, nil, nil, lootboxSvc, statsSvc, time.Minute, nil, nil, nil, mockRng)

	ctx := context.Background()
	gambleID := uuid.New()

	// Setup 2 participants with equal outcome
	participants := []domain.Participant{
		{UserID: "userA", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		{UserID: "userB", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
	}

	gamble := &domain.Gamble{
		ID:           gambleID,
		State:        domain.GambleStateJoining,
		Participants: participants,
	}

	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	drops := []lootbox.DroppedItem{{ItemID: 10, ItemName: domain.ItemMoney, Quantity: 1, Value: 100}}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	tx := new(MockTx)
	repo.On("BeginGambleTx", ctx).Return(tx, nil)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(1), nil)

	repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem, nil)
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)
	lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox1, 1, mock.Anything).Return(drops, nil)

	tx.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)
	tx.On("GetInventory", ctx, mock.Anything).Return(&domain.Inventory{}, nil)
	tx.On("UpdateInventory", ctx, mock.Anything, mock.Anything).Return(nil)
	tx.On("CompleteGamble", ctx, mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()

	// Expect TieBreakLost for userA (loser)
	statsSvc.On("RecordUserEvent", ctx, "userA", domain.EventGambleTieBreakLost, mock.MatchedBy(func(m map[string]interface{}) bool {
		return m["score"] == int64(100)
	})).Return(nil)

	result, err := s.ExecuteGamble(ctx, gambleID)

	assert.NoError(t, err)
	assert.Equal(t, "userB", result.WinnerID)

	repo.AssertExpectations(t)
	statsSvc.AssertExpectations(t)
}

func TestShutdown_WaitsForAsync(t *testing.T) {
	repo := new(MockRepository)
	lootboxSvc := new(MockLootboxService)
	jobSvc := new(MockJobService)

	s := NewService(repo, nil, nil, lootboxSvc, nil, time.Minute, jobSvc, nil, nil, nil)

	ctx := context.Background()
	user := &domain.User{ID: "user1"}
	bets := []domain.LootboxBet{{ItemName: "lootbox_tier1", Quantity: 1}}
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 1}}}
	tx := new(MockTx)

	repo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil)
	repo.On("GetInventory", ctx, "user1").Return(inventory, nil)
	repo.On("GetActiveGamble", ctx).Return(nil, nil)

	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	repo.On("GetItemByName", ctx, "lootbox_tier1").Return(lootboxItem, nil)

	repo.On("BeginTx", ctx).Return(tx, nil)
	tx.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()
	repo.On("CreateGamble", ctx, mock.Anything).Return(nil)
	repo.On("JoinGamble", ctx, mock.Anything).Return(nil)

	// Make AwardXP block
	blockCh := make(chan struct{})
	jobSvc.On("AwardXP", mock.Anything, "user1", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			<-blockCh // Block until we signal
		}).
		Return(nil, nil)

	// Start Gamble (triggers async XP)
	_, err := s.StartGamble(ctx, domain.PlatformTwitch, "123", "testuser", bets)
	assert.NoError(t, err)

	// Call Shutdown in a separate goroutine so we can measure it
	shutdownDone := make(chan struct{})
	start := time.Now()
	go func() {
		_ = s.Shutdown(ctx)
		close(shutdownDone)
	}()

	// Ensure Shutdown is blocked (give it a bit of time to start waiting)
	select {
	case <-shutdownDone:
		t.Fatal("Shutdown returned immediately, should be waiting")
	case <-time.After(10 * time.Millisecond):
		// Good, it's blocked
	}

	// Unblock XP
	close(blockCh)

	// Wait for Shutdown to finish
	select {
	case <-shutdownDone:
		// Success
		assert.True(t, time.Since(start) >= 10*time.Millisecond)
	case <-time.After(1 * time.Second):
		t.Fatal("Shutdown timed out")
	}

	repo.AssertExpectations(t)
	jobSvc.AssertExpectations(t)
}
