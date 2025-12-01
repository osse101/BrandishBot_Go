package gamble

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/concurrency"
	"github.com/osse101/BrandishBot_Go/internal/domain"
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

func TestStartGamble_Success(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil)

	ctx := context.Background()
	user := &domain.User{ID: "user1"}
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 1}}}
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
	repo.AssertExpectations(t)
	tx.AssertExpectations(t)
}

func TestJoinGamble_Success(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	user := &domain.User{ID: "user2"}
	bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 1}}}
	gamble := &domain.Gamble{ID: gambleID, State: domain.GambleStateJoining, JoinDeadline: time.Now().Add(time.Minute)}
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

func TestExecuteGamble_Success(t *testing.T) {
	repo := new(MockRepository)
	lockManager := concurrency.NewLockManager()
	s := NewService(repo, lockManager, nil)

	ctx := context.Background()
	gambleID := uuid.New()
	gamble := &domain.Gamble{
		ID: gambleID,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemID: 1, Quantity: 1}}},
		},
	}
	tx := new(MockTx)
	winnerInventory := &domain.Inventory{Slots: []domain.InventorySlot{}}

	repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
	repo.On("UpdateGambleState", ctx, gambleID, domain.GambleStateOpening).Return(nil)
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
	repo.AssertExpectations(t)
	tx.AssertExpectations(t)
}
