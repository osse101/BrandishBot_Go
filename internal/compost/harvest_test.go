package compost_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/compost"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHarvest(t *testing.T) {
	ctx := context.Background()

	t.Run("IdleBin", func(t *testing.T) {
		mockRepo := mocks.NewMockRepositoryCompostRepository(t)
		mockUserRepo := mocks.NewMockRepositoryUser(t)
		mockProgressionSvc := mocks.NewMockProgressionService(t)
		mockJobSvc := mocks.NewMockJobService(t)
		service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

		userID := "user-123"
		user := &domain.User{ID: userID}
		bin := &domain.CompostBin{UserID: userID, Status: domain.CompostBinStatusIdle}

		mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Once()
		mockRepo.On("GetBin", ctx, userID).Return(bin, nil).Once()
		mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Once()
		mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(true, nil).Once()

		result, err := service.Harvest(ctx, domain.PlatformTwitch, "123", "user")
		require.NoError(t, err)
		assert.False(t, result.Harvested)
		assert.Equal(t, domain.CompostBinStatusIdle, result.Status.Status)
	})

	t.Run("CompostingBin_NotReady", func(t *testing.T) {
		mockRepo := mocks.NewMockRepositoryCompostRepository(t)
		mockUserRepo := mocks.NewMockRepositoryUser(t)
		mockProgressionSvc := mocks.NewMockProgressionService(t)
		mockJobSvc := mocks.NewMockJobService(t)
		service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

		userID := "user-123"
		user := &domain.User{ID: userID}
		readyAt := time.Now().Add(1 * time.Hour)
		bin := &domain.CompostBin{
			UserID:  userID,
			Status:  domain.CompostBinStatusComposting,
			ReadyAt: &readyAt,
		}

		mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Once()
		mockRepo.On("GetBin", ctx, userID).Return(bin, nil).Once()
		mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Once()
		mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(true, nil).Once()

		result, err := service.Harvest(ctx, domain.PlatformTwitch, "123", "user")
		require.NoError(t, err)
		assert.False(t, result.Harvested)
		assert.Equal(t, domain.CompostBinStatusComposting, result.Status.Status)
	})

	t.Run("Success_Ready", func(t *testing.T) {
		mockRepo := mocks.NewMockRepositoryCompostRepository(t)
		mockUserRepo := mocks.NewMockRepositoryUser(t)
		mockProgressionSvc := mocks.NewMockProgressionService(t)
		mockJobSvc := mocks.NewMockJobService(t)
		mockTx := mocks.NewMockRepositoryCompostTx(t)
		service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

		userID := "user-123"
		user := &domain.User{ID: userID}
		readyAt := time.Now().Add(-1 * time.Hour)
		bin := &domain.CompostBin{
			UserID:       userID,
			Status:       domain.CompostBinStatusComposting, // Lazy update will change this to Ready
			ReadyAt:      &readyAt,
			InputValue:   100,
			DominantType: "organic",
			ItemCount:    10,
		}
		item := domain.Item{InternalName: "apple", ID: 1, BaseValue: 10}

		mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Once()
		mockRepo.On("GetBin", ctx, userID).Return(bin, nil).Once()
		mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Once()
		mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(true, nil).Once()

		mockUserRepo.On("GetAllItems", ctx).Return([]domain.Item{item}, nil).Once()
		mockProgressionSvc.On("GetModifiedValue", ctx, "", progression.FeatureCompost, 1.0).Return(1.0, nil).Once()

		mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()

		// Tx calls
		mockTx.On("GetInventory", ctx, userID).Return(&domain.Inventory{Slots: []domain.InventorySlot{}}, nil).Once()
		mockUserRepo.On("GetItemsByNames", ctx, mock.Anything).Return([]domain.Item{item}, nil).Once()
		mockTx.On("UpdateInventory", ctx, userID, mock.Anything).Return(nil).Once()
		mockTx.On("ResetBin", ctx, userID).Return(nil).Once()
		mockTx.On("Commit", ctx).Return(nil).Once()
		mockTx.On("Rollback", ctx).Return(nil).Maybe()

		result, err := service.Harvest(ctx, domain.PlatformTwitch, "123", "user")
		require.NoError(t, err)
		assert.True(t, result.Harvested)
		assert.False(t, result.Output.IsSludge)
	})

	t.Run("Success_Sludge", func(t *testing.T) {
		mockRepo := mocks.NewMockRepositoryCompostRepository(t)
		mockUserRepo := mocks.NewMockRepositoryUser(t)
		mockProgressionSvc := mocks.NewMockProgressionService(t)
		mockJobSvc := mocks.NewMockJobService(t)
		mockTx := mocks.NewMockRepositoryCompostTx(t)
		service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

		userID := "user-123"
		user := &domain.User{ID: userID}
		readyAt := time.Now().Add(-2 * time.Hour)
		sludgeAt := time.Now().Add(-1 * time.Hour)
		bin := &domain.CompostBin{
			UserID:       userID,
			Status:       domain.CompostBinStatusComposting, // Lazy update will change this to Sludge
			ReadyAt:      &readyAt,
			SludgeAt:     &sludgeAt,
			InputValue:   100,
			DominantType: "organic",
		}
		item := domain.Item{InternalName: "sludge", ID: 99, BaseValue: 1}

		mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Once()
		mockRepo.On("GetBin", ctx, userID).Return(bin, nil).Once()
		mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Once()
		mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(true, nil).Once()

		mockUserRepo.On("GetAllItems", ctx).Return([]domain.Item{item}, nil).Once()
		mockProgressionSvc.On("GetModifiedValue", ctx, "", progression.FeatureCompost, 1.0).Return(1.0, nil).Once()

		mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()

		mockTx.On("GetInventory", ctx, userID).Return(&domain.Inventory{Slots: []domain.InventorySlot{}}, nil).Once()
		mockUserRepo.On("GetItemsByNames", ctx, mock.Anything).Return([]domain.Item{item}, nil).Once()
		mockTx.On("UpdateInventory", ctx, userID, mock.Anything).Return(nil).Once()
		mockTx.On("ResetBin", ctx, userID).Return(nil).Once()
		mockTx.On("Commit", ctx).Return(nil).Once()
		mockTx.On("Rollback", ctx).Return(nil).Maybe()

		result, err := service.Harvest(ctx, domain.PlatformTwitch, "123", "user")
		require.NoError(t, err)
		assert.True(t, result.Harvested)
		assert.True(t, result.Output.IsSludge)
	})

	t.Run("ExecuteHarvest_DBErrors", func(t *testing.T) {
		userID := "user-123"
		user := &domain.User{ID: userID}
		readyAt := time.Now().Add(-1 * time.Hour)
		bin := &domain.CompostBin{
			UserID:       userID,
			Status:       domain.CompostBinStatusComposting, // Lazy update will change this to Ready
			ReadyAt:      &readyAt,
			InputValue:   100,
			DominantType: "organic",
			ItemCount:    10,
		}
		item := domain.Item{InternalName: "apple", ID: 1, BaseValue: 10}

		setupMocks := func(t *testing.T) (*mocks.MockRepositoryCompostRepository, *mocks.MockRepositoryCompostTx, *mocks.MockRepositoryUser, *mocks.MockJobService, compost.Service) {
			mockRepo := mocks.NewMockRepositoryCompostRepository(t)
			mockUserRepo := mocks.NewMockRepositoryUser(t)
			mockProgressionSvc := mocks.NewMockProgressionService(t)
			mockJobSvc := mocks.NewMockJobService(t)
			mockTx := mocks.NewMockRepositoryCompostTx(t)
			service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

			mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Once()
			mockRepo.On("GetBin", ctx, userID).Return(bin, nil).Once()
			mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Once()
			mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(true, nil).Once()

			mockUserRepo.On("GetAllItems", ctx).Return([]domain.Item{item}, nil).Once()
			mockProgressionSvc.On("GetModifiedValue", ctx, "", progression.FeatureCompost, 1.0).Return(1.0, nil).Once()
			return mockRepo, mockTx, mockUserRepo, mockJobSvc, service
		}

		t.Run("BeginTx_Error", func(t *testing.T) {
			mockRepo, _, _, _, service := setupMocks(t)
			mockRepo.On("BeginTx", ctx).Return(nil, assert.AnError).Once()
			_, err := service.Harvest(ctx, domain.PlatformTwitch, "123", "user")
			assert.ErrorContains(t, err, "failed to begin harvest transaction")
		})

		t.Run("GetInventory_Error", func(t *testing.T) {
			mockRepo, mockTx, _, _, service := setupMocks(t)
			mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
			mockTx.On("GetInventory", ctx, userID).Return(nil, assert.AnError).Once()
			mockTx.On("Rollback", ctx).Return(nil).Maybe()
			_, err := service.Harvest(ctx, domain.PlatformTwitch, "123", "user")
			assert.ErrorContains(t, err, "failed to get inventory")
		})

		t.Run("UpdateInventory_Error", func(t *testing.T) {
			mockRepo, mockTx, mockUserRepo, mockJobSvc, service := setupMocks(t)
			mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
			mockTx.On("GetInventory", ctx, userID).Return(&domain.Inventory{Slots: []domain.InventorySlot{}}, nil).Once()
			mockUserRepo.On("GetItemsByNames", ctx, mock.Anything).Return([]domain.Item{item}, nil).Once()

			mockJobSvc.On("HandleEvent", ctx, mock.Anything).Return(nil).Maybe()

			mockTx.On("UpdateInventory", ctx, userID, mock.Anything).Return(assert.AnError).Once()
			mockTx.On("Rollback", ctx).Return(nil).Maybe()
			_, err := service.Harvest(ctx, domain.PlatformTwitch, "123", "user")
			assert.ErrorContains(t, err, "failed to update inventory")
		})

		t.Run("ResetBin_Error", func(t *testing.T) {
			mockRepo, mockTx, mockUserRepo, mockJobSvc, service := setupMocks(t)
			mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
			mockTx.On("GetInventory", ctx, userID).Return(&domain.Inventory{Slots: []domain.InventorySlot{}}, nil).Once()
			mockUserRepo.On("GetItemsByNames", ctx, mock.Anything).Return([]domain.Item{item}, nil).Once()
			mockJobSvc.On("HandleEvent", ctx, mock.Anything).Return(nil).Maybe()

			mockTx.On("UpdateInventory", ctx, userID, mock.Anything).Return(nil).Once()
			mockTx.On("ResetBin", ctx, userID).Return(assert.AnError).Once()
			mockTx.On("Rollback", ctx).Return(nil).Maybe()
			_, err := service.Harvest(ctx, domain.PlatformTwitch, "123", "user")
			assert.ErrorContains(t, err, "failed to reset bin")
		})

		t.Run("Commit_Error", func(t *testing.T) {
			mockRepo, mockTx, mockUserRepo, mockJobSvc, service := setupMocks(t)
			mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
			mockTx.On("GetInventory", ctx, userID).Return(&domain.Inventory{Slots: []domain.InventorySlot{}}, nil).Once()
			mockUserRepo.On("GetItemsByNames", ctx, mock.Anything).Return([]domain.Item{item}, nil).Once()
			mockJobSvc.On("HandleEvent", ctx, mock.Anything).Return(nil).Maybe()

			mockTx.On("UpdateInventory", ctx, userID, mock.Anything).Return(nil).Once()
			mockTx.On("ResetBin", ctx, userID).Return(nil).Once()
			mockTx.On("Commit", ctx).Return(assert.AnError).Once()
			mockTx.On("Rollback", ctx).Return(nil).Maybe()
			_, err := service.Harvest(ctx, domain.PlatformTwitch, "123", "user")
			assert.ErrorContains(t, err, "failed to commit harvest")
		})
	})
}
