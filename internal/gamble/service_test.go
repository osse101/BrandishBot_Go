package gamble

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/concurrency"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository
type MockRepository struct {
	mock.Mock
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

func (m *MockLootboxService) OpenLootbox(ctx context.Context, lootboxName string, quantity int) ([]lootbox.DroppedItem, error) {
	args := m.Called(ctx, lootboxName, quantity)
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

// ========================================
// StartGamble Tests
// ========================================

func TestStartGamble_Success(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	user := &domain.User{ID: "user1"}
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 2}}
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 5}}}
	tx := new(MockTx)

	repo.On("GetUserByPlatformID", ctx, "twitch", "123").Return(user, nil)
	repo.On("GetActiveGamble", ctx).Return(nil, nil)
	repo.On("BeginTx", ctx).Return(tx, nil)
	tx.On("GetInventory", ctx, "user1").Return(inventory, nil)
	tx.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()
	repo.On("CreateGamble", ctx, mock.Anything).Return(nil)
	repo.On("JoinGamble", ctx, mock.Anything).Return(nil)

	gamble, err := s.StartGamble(ctx, "twitch", "123", "testuser", bets)

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
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	bets := []domain.LootboxBet{}

	gamble, err := s.StartGamble(ctx, "twitch", "123", "testuser", bets)

	assert.Error(t, err)
	assert.Nil(t, gamble)
	assert.Contains(t, err.Error(), "at least one lootbox bet is required")
}

func TestStartGamble_InvalidBetQuantity(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 0}}

	gamble, err := s.StartGamble(ctx, "twitch", "123", "testuser", bets)

	assert.Error(t, err)
	assert.Nil(t, gamble)
	assert.Contains(t, err.Error(), "bet quantity must be positive")
}

func TestStartGamble_UserNotFound(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}

	repo.On("GetUserByPlatformID", ctx, "twitch", "123").Return(nil, nil)

	gamble, err := s.StartGamble(ctx, "twitch", "123", "testuser", bets)

	assert.Error(t, err)
	assert.Nil(t, gamble)
	assert.Contains(t, err.Error(), "user not found")
	repo.AssertExpectations(t)
}

func TestStartGamble_ActiveGambleExists(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	user := &domain.User{ID: "user1"}
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}
	activeGamble := &domain.Gamble{ID: uuid.New(), State: domain.GambleStateJoining}

	repo.On("GetUserByPlatformID", ctx, "twitch", "123").Return(user, nil)
	repo.On("GetActiveGamble", ctx).Return(activeGamble, nil)

	gamble, err := s.StartGamble(ctx, "twitch", "123", "testuser", bets)

	assert.Error(t, err)
	assert.Nil(t, gamble)
	assert.Contains(t, err.Error(), "a gamble is already active")
	repo.AssertExpectations(t)
}

func TestStartGamble_InsufficientLootboxes(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	user := &domain.User{ID: "user1"}
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 5}}
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 2}}}
	tx := new(MockTx)

	repo.On("GetUserByPlatformID", ctx, "twitch", "123").Return(user, nil)
	repo.On("GetActiveGamble", ctx).Return(nil, nil)
	repo.On("BeginTx", ctx).Return(tx, nil)
	tx.On("GetInventory", ctx, "user1").Return(inventory, nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()

	gamble, err := s.StartGamble(ctx, "twitch", "123", "testuser", bets)

	assert.Error(t, err)
	assert.Nil(t, gamble)
	assert.Contains(t, err.Error(), "insufficient quantity")
	repo.AssertExpectations(t)
	tx.AssertExpectations(t)
}

func TestStartGamble_LootboxNotInInventory(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	user := &domain.User{ID: "user1"}
	bets := []domain.LootboxBet{{ItemID: 99, Quantity: 1}}
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 5}}}
	tx := new(MockTx)

	repo.On("GetUserByPlatformID", ctx, "twitch", "123").Return(user, nil)
	repo.On("GetActiveGamble", ctx).Return(nil, nil)
	repo.On("BeginTx", ctx).Return(tx, nil)
	tx.On("GetInventory", ctx, "user1").Return(inventory, nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()

	gamble, err := s.StartGamble(ctx, "twitch", "123", "testuser", bets)

	assert.Error(t, err)
	assert.Nil(t, gamble)
	assert.Contains(t, err.Error(), "item not found")
	repo.AssertExpectations(t)
	tx.AssertExpectations(t)
}

// ========================================
// JoinGamble Tests
// ========================================

func TestJoinGamble_Success(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user2"}
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 3}}}
	gamble := &domain.Gamble{
		ID:           gambleID,
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(time.Minute),
	}
	tx := new(MockTx)

	repo.On("GetUserByPlatformID", ctx, "twitch", "456").Return(user, nil)
	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	repo.On("BeginTx", ctx).Return(tx, nil)
	tx.On("GetInventory", ctx, "user2").Return(inventory, nil)
	tx.On("UpdateInventory", ctx, "user2", mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()
	repo.On("JoinGamble", ctx, mock.Anything).Return(nil)

	err := s.JoinGamble(ctx, gambleID, "twitch", "456", "joiner", bets)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	tx.AssertExpectations(t)
}

func TestJoinGamble_NoBets(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	gambleID := uuid.New()
	bets := []domain.LootboxBet{}

	err := s.JoinGamble(ctx, gambleID, "twitch", "456", "joiner", bets)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one lootbox bet is required")
}

func TestJoinGamble_GambleNotFound(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user2"}
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}

	repo.On("GetUserByPlatformID", ctx, "twitch", "456").Return(user, nil)
	repo.On("GetGamble", ctx, gambleID).Return(nil, nil)

	err := s.JoinGamble(ctx, gambleID, "twitch", "456", "joiner", bets)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gamble not found")
	repo.AssertExpectations(t)
}

func TestJoinGamble_WrongState(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user2"}
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}
	gamble := &domain.Gamble{
		ID:           gambleID,
		State:        domain.GambleStateOpening,
		JoinDeadline: time.Now().Add(time.Minute),
	}

	repo.On("GetUserByPlatformID", ctx, "twitch", "456").Return(user, nil)
	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	err := s.JoinGamble(ctx, gambleID, "twitch", "456", "joiner", bets)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in joining state")
	repo.AssertExpectations(t)
}

func TestJoinGamble_DeadlinePassed(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user2"}
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}
	gamble := &domain.Gamble{
		ID:           gambleID,
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(-time.Minute), // Past deadline
	}

	repo.On("GetUserByPlatformID", ctx, "twitch", "456").Return(user, nil)
	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	err := s.JoinGamble(ctx, gambleID, "twitch", "456", "joiner", bets)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "join deadline has passed")
	repo.AssertExpectations(t)
}

func TestJoinGamble_InsufficientLootboxes(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user2"}
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 10}}
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 2}}}
	gamble := &domain.Gamble{
		ID:           gambleID,
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(time.Minute),
	}
	tx := new(MockTx)

	repo.On("GetUserByPlatformID", ctx, "twitch", "456").Return(user, nil)
	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	repo.On("BeginTx", ctx).Return(tx, nil)
	tx.On("GetInventory", ctx, "user2").Return(inventory, nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()

	err := s.JoinGamble(ctx, gambleID, "twitch", "456", "joiner", bets)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient quantity")
	repo.AssertExpectations(t)
	tx.AssertExpectations(t)
}

// ========================================
// ExecuteGamble Tests
// ========================================

func TestExecuteGamble_Success(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	lootboxSvc := new(MockLootboxService)
	s := NewService(repo, lockManager, nil, lootboxSvc, time.Minute)

	ctx := context.Background()
	gambleID := uuid.New()
	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemID: 1, Quantity: 1}}},
		},
	}
	tx := new(MockTx)
	winnerInventory := &domain.Inventory{Slots: []domain.InventorySlot{}}
	lootboxItem := &domain.Item{ID: 1, Name: "lootbox1"}
	droppedItems := []lootbox.DroppedItem{{ItemID: 10, ItemName: "coin", Quantity: 5, Value: 10}}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	repo.On("UpdateGambleState", ctx, gambleID, domain.GambleStateOpening).Return(nil)
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)
	lootboxSvc.On("OpenLootbox", ctx, "lootbox1", 1).Return(droppedItems, nil)
	repo.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)
	repo.On("BeginTx", ctx).Return(tx, nil)
	tx.On("GetInventory", ctx, "user1").Return(winnerInventory, nil)
	tx.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()
	repo.On("CompleteGamble", ctx, mock.Anything).Return(nil)

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
	lockManager := concurrency.NewLockManager()
	lootboxSvc := new(MockLootboxService)
	s := NewService(repo, lockManager, nil, lootboxSvc, time.Minute)

	ctx := context.Background()
	gambleID := uuid.New()
	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemID: 1, Quantity: 2}}},
			{UserID: "user2", LootboxBets: []domain.LootboxBet{{ItemID: 1, Quantity: 1}}},
		},
	}
	tx := new(MockTx)
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{}}
	lootboxItem := &domain.Item{ID: 1, Name: "lootbox1"}
	droppedItems := []lootbox.DroppedItem{{ItemID: 10, ItemName: "coin", Quantity: 5, Value: 10}}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	repo.On("UpdateGambleState", ctx, gambleID, domain.GambleStateOpening).Return(nil)
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)
	lootboxSvc.On("OpenLootbox", ctx, "lootbox1", mock.Anything).Return(droppedItems, nil)
	repo.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)
	repo.On("BeginTx", ctx).Return(tx, nil)
	tx.On("GetInventory", ctx, mock.Anything).Return(inventory, nil)
	tx.On("UpdateInventory", ctx, mock.Anything, mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()
	repo.On("CompleteGamble", ctx, mock.Anything).Return(nil)

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
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	gambleID := uuid.New()

	repo.On("GetGamble", ctx, gambleID).Return(nil, nil)

	result, err := s.ExecuteGamble(ctx, gambleID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "gamble not found")
	repo.AssertExpectations(t)
}

func TestExecuteGamble_AlreadyCompleted(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

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
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

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
	assert.Contains(t, err.Error(), "not in joining state")
	repo.AssertExpectations(t)
}

func TestExecuteGamble_StateUpdateFails(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()
	gambleID := uuid.New()
	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemID: 1, Quantity: 1}}},
		},
	}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	repo.On("UpdateGambleState", ctx, gambleID, domain.GambleStateOpening).Return(errors.New("database error"))

	result, err := s.ExecuteGamble(ctx, gambleID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to update state")
	repo.AssertExpectations(t)
}

func TestExecuteGamble_SaveOpenedItemsFails(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	lootboxSvc := new(MockLootboxService)
	s := NewService(repo, lockManager, nil, lootboxSvc, time.Minute)

	ctx := context.Background()
	gambleID := uuid.New()
	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemID: 1, Quantity: 1}}},
		},
	}
	lootboxItem := &domain.Item{ID: 1, Name: "lootbox1"}
	droppedItems := []lootbox.DroppedItem{{ItemID: 10, ItemName: "coin", Quantity: 5, Value: 10}}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	repo.On("UpdateGambleState", ctx, gambleID, domain.GambleStateOpening).Return(nil)
	repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)
	lootboxSvc.On("OpenLootbox", ctx, "lootbox1", mock.Anything).Return(droppedItems, nil)
	repo.On("SaveOpenedItems", ctx, mock.Anything).Return(errors.New("database error"))

	result, err := s.ExecuteGamble(ctx, gambleID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to save opened items")
	repo.AssertExpectations(t)
	lootboxSvc.AssertExpectations(t)
}

// ========================================
// GetGamble Tests
// ========================================

func TestGetGamble_Success(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

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
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

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
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil, new(MockLootboxService), time.Minute)

	ctx := context.Background()

	repo.On("GetActiveGamble", ctx).Return(nil, nil)

	gamble, err := s.GetActiveGamble(ctx)

	assert.NoError(t, err)
	assert.Nil(t, gamble)
	repo.AssertExpectations(t)
}
