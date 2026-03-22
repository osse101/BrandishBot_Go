package expedition

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

type MockExpeditionRepo struct {
	mock.Mock
}

func (m *MockExpeditionRepo) CreateExpedition(ctx context.Context, expedition *domain.Expedition) error {
	args := m.Called(ctx, expedition)
	return args.Error(0)
}

func (m *MockExpeditionRepo) GetExpedition(ctx context.Context, id uuid.UUID) (*domain.ExpeditionDetails, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ExpeditionDetails), args.Error(1)
}

func (m *MockExpeditionRepo) AddParticipant(ctx context.Context, participant *domain.ExpeditionParticipant) error {
	args := m.Called(ctx, participant)
	return args.Error(0)
}

func (m *MockExpeditionRepo) UpdateExpeditionState(ctx context.Context, id uuid.UUID, state domain.ExpeditionState) error {
	args := m.Called(ctx, id, state)
	return args.Error(0)
}

func (m *MockExpeditionRepo) UpdateExpeditionStateIfMatches(ctx context.Context, id uuid.UUID, expected, newState domain.ExpeditionState) (int64, error) {
	args := m.Called(ctx, id, expected, newState)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockExpeditionRepo) GetActiveExpedition(ctx context.Context) (*domain.ExpeditionDetails, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ExpeditionDetails), args.Error(1)
}

func (m *MockExpeditionRepo) GetParticipants(ctx context.Context, expeditionID uuid.UUID) ([]domain.ExpeditionParticipant, error) {
	args := m.Called(ctx, expeditionID)
	return args.Get(0).([]domain.ExpeditionParticipant), args.Error(1)
}

func (m *MockExpeditionRepo) SaveParticipantRewards(ctx context.Context, expeditionID uuid.UUID, userID uuid.UUID, rewards *domain.ExpeditionRewards) error {
	args := m.Called(ctx, expeditionID, userID, rewards)
	return args.Error(0)
}

func (m *MockExpeditionRepo) UpdateParticipantResults(ctx context.Context, expeditionID uuid.UUID, userID uuid.UUID, isLeader bool, jobLevels map[string]int, money int, xp int, items []string) error {
	args := m.Called(ctx, expeditionID, userID, isLeader, jobLevels, money, xp, items)
	return args.Error(0)
}

func (m *MockExpeditionRepo) CompleteExpedition(ctx context.Context, expeditionID uuid.UUID) error {
	args := m.Called(ctx, expeditionID)
	return args.Error(0)
}

func (m *MockExpeditionRepo) GetLastCompletedExpedition(ctx context.Context) (*domain.Expedition, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Expedition), args.Error(1)
}

func (m *MockExpeditionRepo) SaveJournalEntry(ctx context.Context, entry *domain.ExpeditionJournalEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockExpeditionRepo) GetJournalEntries(ctx context.Context, expeditionID uuid.UUID) ([]domain.ExpeditionJournalEntry, error) {
	args := m.Called(ctx, expeditionID)
	return args.Get(0).([]domain.ExpeditionJournalEntry), args.Error(1)
}

func (m *MockExpeditionRepo) BeginExpeditionTx(ctx context.Context) (repository.ExpeditionTx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(repository.ExpeditionTx), args.Error(1)
}

func (m *MockExpeditionRepo) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockExpeditionRepo) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Inventory), args.Error(1)
}

func (m *MockExpeditionRepo) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	args := m.Called(ctx, userID, inventory)
	return args.Error(0)
}

type MockJobSvc struct {
	mock.Mock
}

func (m *MockJobSvc) GetUserJobs(ctx context.Context, userID string) ([]domain.UserJobInfo, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.UserJobInfo), args.Error(1)
}

func (m *MockJobSvc) GetJobLevel(ctx context.Context, userID, jobKey string) (int, error) {
	args := m.Called(ctx, userID, jobKey)
	return args.Int(0), args.Error(1)
}

type MockUserSvc struct {
	mock.Mock
}

func (m *MockUserSvc) AddItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) error {
	args := m.Called(ctx, platform, username, itemName, quantity)
	return args.Error(0)
}

func (m *MockUserSvc) RemoveItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) (int, error) {
	args := m.Called(ctx, platform, username, itemName, quantity)
	return args.Int(0), args.Error(1)
}

type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(ctx context.Context, e event.Event) error {
	args := m.Called(ctx, e)
	return args.Error(0)
}

func (m *MockEventBus) Subscribe(eventType event.Type, handler event.Handler) {
	m.Called(eventType, handler)
}

func TestStartExpedition_Constraints(t *testing.T) {
	ctx := context.Background()
	platform := domain.PlatformTwitch
	platformID := "123"
	username := "testuser"
	userID := uuid.New().String()

	setupMocks := func() (*MockExpeditionRepo, *MockJobSvc, *MockUserSvc, *MockEventBus, Service) {
		repo := new(MockExpeditionRepo)
		jobSvc := new(MockJobSvc)
		userSvc := new(MockUserSvc)
		bus := new(MockEventBus)
		svc := NewService(repo, bus, nil, jobSvc, nil, userSvc, nil, nil, 5*time.Minute, 10*time.Minute)
		return repo, jobSvc, userSvc, bus, svc
	}

	t.Run("Invalid expedition type", func(t *testing.T) {
		_, _, _, _, svc := setupMocks()
		exp, err := svc.StartExpedition(ctx, platform, platformID, username, domain.ExpeditionType("invalid"))

		assert.Error(t, err)
		assert.Nil(t, exp)
		assert.ErrorIs(t, err, domain.ErrInvalidExpeditionType)
	})

	t.Run("Level requirement failure", func(t *testing.T) {
		repo, jobSvc, _, _, svc := setupMocks()
		user := &domain.User{ID: userID, Username: username}

		repo.On("GetUserByPlatformID", mock.Anything, platform, platformID).Return(user, nil)
		repo.On("GetLastCompletedExpedition", mock.Anything).Return(nil, nil)
		jobSvc.On("GetJobLevel", mock.Anything, userID, domain.JobKeyExplorer).Return(4, nil)

		exp, err := svc.StartExpedition(ctx, platform, platformID, username, domain.ExpeditionTypeNormal)

		assert.Error(t, err)
		assert.Nil(t, exp)
		assert.ErrorIs(t, err, domain.ErrInsufficientLevel)
	})

	t.Run("Money cost failure", func(t *testing.T) {
		repo, jobSvc, userSvc, _, svc := setupMocks()
		user := &domain.User{ID: userID, Username: username}

		repo.On("GetUserByPlatformID", mock.Anything, platform, platformID).Return(user, nil)
		repo.On("GetLastCompletedExpedition", mock.Anything).Return(nil, nil)
		jobSvc.On("GetJobLevel", mock.Anything, userID, domain.JobKeyExplorer).Return(5, nil)
		userSvc.On("RemoveItemByUsername", mock.Anything, platform, username, domain.ItemMoney, 500).Return(0, nil)

		exp, err := svc.StartExpedition(ctx, platform, platformID, username, domain.ExpeditionTypeNormal)

		assert.Error(t, err)
		assert.Nil(t, exp)
		assert.ErrorIs(t, err, domain.ErrInsufficientFunds)
	})

	t.Run("Global cooldown failure", func(t *testing.T) {
		repo, _, _, _, svc := setupMocks()
		user := &domain.User{ID: userID, Username: username}
		now := time.Now()
		lastExp := &domain.Expedition{
			CompletedAt: &now,
		}

		repo.On("GetUserByPlatformID", mock.Anything, platform, platformID).Return(user, nil)
		repo.On("GetLastCompletedExpedition", mock.Anything).Return(lastExp, nil)

		exp, err := svc.StartExpedition(ctx, platform, platformID, username, domain.ExpeditionTypeNormal)

		assert.Error(t, err)
		assert.Nil(t, exp)
		assert.ErrorIs(t, err, domain.ErrOnCooldown)
	})

	t.Run("Success path", func(t *testing.T) {
		repo, jobSvc, userSvc, bus, svc := setupMocks()
		user := &domain.User{ID: userID, Username: username}

		repo.On("GetUserByPlatformID", mock.Anything, platform, platformID).Return(user, nil)
		repo.On("GetLastCompletedExpedition", mock.Anything).Return(nil, nil)
		jobSvc.On("GetJobLevel", mock.Anything, userID, domain.JobKeyExplorer).Return(5, nil)
		userSvc.On("RemoveItemByUsername", mock.Anything, platform, username, domain.ItemMoney, 500).Return(500, nil)
		repo.On("GetActiveExpedition", mock.Anything).Return(nil, nil)
		repo.On("CreateExpedition", mock.Anything, mock.Anything).Return(nil)
		repo.On("AddParticipant", mock.Anything, mock.Anything).Return(nil)
		bus.On("Publish", mock.Anything, mock.Anything).Return(nil)

		exp, err := svc.StartExpedition(ctx, platform, platformID, username, domain.ExpeditionTypeNormal)

		assert.NoError(t, err)
		assert.NotNil(t, exp)
		repo.AssertExpectations(t)
		jobSvc.AssertExpectations(t)
		userSvc.AssertExpectations(t)
	})
}
