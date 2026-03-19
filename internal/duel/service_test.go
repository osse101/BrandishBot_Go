package duel

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/mocks"
)

// Simplified test struct for quick mocking
type mockDuelRepo struct {
	mock.Mock
}

func (m *mockDuelRepo) CreateDuel(ctx context.Context, duel *domain.Duel) error { return nil }
func (m *mockDuelRepo) GetDuel(ctx context.Context, id uuid.UUID) (*domain.Duel, error) {
	return nil, nil
}
func (m *mockDuelRepo) UpdateDuelState(ctx context.Context, id uuid.UUID, state domain.DuelState) error {
	return nil
}
func (m *mockDuelRepo) GetPendingDuelsForUser(ctx context.Context, userID uuid.UUID) ([]domain.Duel, error) {
	return nil, nil
}
func (m *mockDuelRepo) AcceptDuel(ctx context.Context, id uuid.UUID, result *domain.DuelResult) error {
	return nil
}
func (m *mockDuelRepo) DeclineDuel(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockDuelRepo) ExpireDuels(ctx context.Context) error               { return nil }
func (m *mockDuelRepo) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return nil, nil
}
func (m *mockDuelRepo) BeginTx(ctx context.Context) (repository.Tx, error) { return nil, nil }

func (m *mockDuelRepo) BeginDuelTx(ctx context.Context) (repository.DuelTx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(repository.DuelTx), args.Error(1)
}

type mockDuelTx struct {
	mock.Mock
}

func (m *mockDuelTx) Commit(ctx context.Context) error   { return m.Called(ctx).Error(0) }
func (m *mockDuelTx) Rollback(ctx context.Context) error { return m.Called(ctx).Error(0) }
func (m *mockDuelTx) GetDuel(ctx context.Context, id uuid.UUID) (*domain.Duel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Duel), args.Error(1)
}
func (m *mockDuelTx) UpdateDuelState(ctx context.Context, id uuid.UUID, state domain.DuelState) error {
	return m.Called(ctx, id, state).Error(0)
}
func (m *mockDuelTx) AcceptDuel(ctx context.Context, id uuid.UUID, result *domain.DuelResult) error {
	return m.Called(ctx, id, result).Error(0)
}

type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) RemoveItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) (int, error) {
	args := m.Called(ctx, platform, username, itemName, quantity)
	return args.Int(0), args.Error(1)
}
func (m *mockUserService) AddItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) error {
	return m.Called(ctx, platform, username, itemName, quantity).Error(0)
}
func (m *mockUserService) AddTimeout(ctx context.Context, platform, username string, duration time.Duration, reason string) error {
	return m.Called(ctx, platform, username, duration, reason).Error(0)
}
func (m *mockUserService) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	args := m.Called(ctx, platform, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserService) FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) UpsertUser(ctx context.Context, user *domain.User) error { return nil }
func (m *mockUserRepo) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) UpdateUser(ctx context.Context, user domain.User) error { return nil }
func (m *mockUserRepo) DeleteUser(ctx context.Context, userID string) error    { return nil }
func (m *mockUserRepo) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return nil, nil
}
func (m *mockUserRepo) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return nil
}
func (m *mockUserRepo) DeleteInventory(ctx context.Context, userID string) error { return nil }
func (m *mockUserRepo) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	return nil, nil
}
func (m *mockUserRepo) GetItemsByNames(ctx context.Context, names []string) ([]domain.Item, error) {
	return nil, nil
}
func (m *mockUserRepo) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	return nil, nil
}
func (m *mockUserRepo) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	return nil, nil
}
func (m *mockUserRepo) GetAllItems(ctx context.Context) ([]domain.Item, error) { return nil, nil }
func (m *mockUserRepo) GetRecentlyActiveUsers(ctx context.Context, limit int) ([]domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) BeginTx(ctx context.Context) (repository.UserTx, error) { return nil, nil }
func (m *mockUserRepo) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	return nil, nil
}
func (m *mockUserRepo) UpdateCooldown(ctx context.Context, userID, action string, cooldownTime time.Time) error {
	return nil
}
func (m *mockUserRepo) GetFeatureState(ctx context.Context, featureKey string) (bool, error) {
	return false, nil
}
func (m *mockUserRepo) MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, user domain.User, inventory domain.Inventory) error {
	return nil
}

func TestService_Accept(t *testing.T) {
	t.Parallel()
	// A bit tricky because of rand.Seed being global and we can't easily mock rand natively without refactoring code.
	// But it handles a random 50/50 so tests might have to mock BOTH AddItem or AddTimeout depending on luck, or we just mock 'Anything'
	ctx := context.Background()

	challengerID := uuid.New()
	opponentID := uuid.New()
	duelID := uuid.New()

	mockOpponent := &domain.User{
		ID:       opponentID.String(),
		Username: "opponent",
	}

	mockChallenger := &domain.User{
		ID:       challengerID.String(),
		Username: "challenger",
	}

	validDuel := &domain.Duel{
		ID:           duelID,
		ChallengerID: challengerID,
		OpponentID:   &opponentID,
		State:        domain.DuelStatePending,
		ExpiresAt:    time.Now().Add(10 * time.Minute),
		Stakes: domain.DuelStakes{
			WagerItemKey:    "money",
			WagerAmount:     100,
			TimeoutDuration: 60,
		},
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		repo := new(mockDuelRepo)
		tx := new(mockDuelTx)
		userRepo := new(mockUserRepo)
		userSvc := new(mockUserService)
		progSvc := mocks.NewMockProgressionService(t)
		eventBus := mocks.NewMockEventBus(t)

		repo.On("BeginDuelTx", ctx).Return(tx, nil)
		tx.On("Rollback", ctx).Return(nil)
		tx.On("GetDuel", ctx, duelID).Return(validDuel, nil)

		userSvc.On("FindUserByPlatformID", ctx, "twitch", "123").Return(mockOpponent, nil)
		userRepo.On("GetUserByID", ctx, challengerID.String()).Return(mockChallenger, nil)

		userSvc.On("RemoveItemByUsername", ctx, "twitch", "opponent", "money", 100).Return(100, nil)
		userSvc.On("AddItemByUsername", ctx, "twitch", mock.Anything, "money", 200).Return(nil)
		userSvc.On("AddTimeout", ctx, "twitch", mock.Anything, 60*time.Second, mock.Anything).Return(nil)

		tx.On("AcceptDuel", ctx, duelID, mock.Anything).Return(nil)
		tx.On("Commit", ctx).Return(nil)

		svc := NewService(repo, userRepo, eventBus, progSvc, userSvc, 5*time.Minute)
		result, err := svc.Accept(ctx, "twitch", "123", duelID)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("expired duel", func(t *testing.T) {
		t.Parallel()
		repo := new(mockDuelRepo)
		tx := new(mockDuelTx)

		repo.On("BeginDuelTx", ctx).Return(tx, nil)
		tx.On("Rollback", ctx).Return(nil)

		expiredDuel := &domain.Duel{
			ID:           duelID,
			ChallengerID: challengerID,
			OpponentID:   &opponentID,
			State:        domain.DuelStatePending,
			ExpiresAt:    time.Now().Add(-1 * time.Minute), // EXPIRED
		}
		tx.On("GetDuel", ctx, duelID).Return(expiredDuel, nil)
		tx.On("UpdateDuelState", ctx, duelID, domain.DuelStateExpired).Return(nil)
		tx.On("Commit", ctx).Return(nil)

		svc := NewService(repo, nil, nil, nil, nil, 5*time.Minute)
		result, err := svc.Accept(ctx, "twitch", "123", duelID)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, domain.ErrDuelExpired)
	})

	t.Run("unauthorized user", func(t *testing.T) {
		t.Parallel()
		repo := new(mockDuelRepo)
		tx := new(mockDuelTx)
		userSvc := new(mockUserService)

		repo.On("BeginDuelTx", ctx).Return(tx, nil)
		tx.On("Rollback", ctx).Return(nil)
		tx.On("GetDuel", ctx, duelID).Return(validDuel, nil)

		// Returns a user with a DIFFERENT ID than opponentID
		wrongUser := &domain.User{
			ID:       uuid.New().String(),
			Username: "wrong",
		}
		userSvc.On("FindUserByPlatformID", ctx, "twitch", "wrong_id").Return(wrongUser, nil)

		svc := NewService(repo, nil, nil, nil, userSvc, 5*time.Minute)
		result, err := svc.Accept(ctx, "twitch", "wrong_id", duelID)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, domain.ErrDuelUnauthorized)
	})

	t.Run("not pending", func(t *testing.T) {
		t.Parallel()
		repo := new(mockDuelRepo)
		tx := new(mockDuelTx)

		repo.On("BeginDuelTx", ctx).Return(tx, nil)
		tx.On("Rollback", ctx).Return(nil)

		notPendingDuel := &domain.Duel{
			ID:           duelID,
			ChallengerID: challengerID,
			OpponentID:   &opponentID,
			State:        domain.DuelStateCompleted, // NOT PENDING
		}

		tx.On("GetDuel", ctx, duelID).Return(notPendingDuel, nil)

		svc := NewService(repo, nil, nil, nil, nil, 5*time.Minute)
		result, err := svc.Accept(ctx, "twitch", "123", duelID)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, domain.ErrDuelNotPending)
	})
}
