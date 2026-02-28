package compost_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/compost"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestDeposit(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockRepo := mocks.NewMockRepositoryCompostRepository(t)
		mockUserRepo := mocks.NewMockRepositoryUser(t)
		mockProgressionSvc := mocks.NewMockProgressionService(t)
		mockJobSvc := mocks.NewMockJobService(t)
		mockTx := mocks.NewMockRepositoryCompostTx(t)
		service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

		userID := "user-123"
		platform := domain.PlatformTwitch
		platformID := "12345"
		itemID := 1
		itemName := "apple"

		user := &domain.User{ID: userID}
		bin := &domain.CompostBin{
			UserID:    userID,
			Status:    domain.CompostBinStatusIdle,
			ItemCount: 0,
			Capacity:  5,
			Items:     []domain.CompostBinItem{},
		}
		item := domain.Item{
			ID:           itemID,
			InternalName: itemName,
			PublicName:   "Apple",
			Types:        []string{domain.CompostableTag},
			BaseValue:    10,
			ContentType:  []string{"organic"},
		}

		mockRepo.On("GetUserByPlatformID", ctx, platform, platformID).Return(user, nil).Once()
		mockRepo.On("GetBin", ctx, userID).Return(bin, nil).Once()

		mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Once()
		mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(true, nil).Once()

		mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureCompostCapacity, 3.0).Return(5.0, nil).Once()

		mockUserRepo.On("GetAllItems", ctx).Return([]domain.Item{item}, nil).Once()

		mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
		mockTx.On("GetBinForUpdate", ctx, userID).Return(bin, nil).Once()

		inv := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: itemID, Quantity: 5},
			},
		}
		mockTx.On("GetInventory", ctx, userID).Return(inv, nil).Once()
		mockTx.On("UpdateInventory", ctx, userID, mock.MatchedBy(func(i domain.Inventory) bool {
			return i.Slots[0].Quantity == 2 // 5 - 3
		})).Return(nil).Once()

		mockTx.On("UpdateBin", ctx, mock.MatchedBy(func(b *domain.CompostBin) bool {
			return b.Status == domain.CompostBinStatusComposting && b.ItemCount == 3 && b.Capacity == 5
		})).Return(nil).Once()

		mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureCompostSpeed, 0.0).Return(1.0, nil).Once()
		mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureSludgeExtension, 0.0).Return(0.0, nil).Once()

		mockTx.On("Commit", ctx).Return(nil).Once()
		mockTx.On("Rollback", ctx).Return(nil).Maybe()

		depositItems := []compost.DepositItem{{ItemName: itemName, Quantity: 3}}
		updatedBin, err := service.Deposit(ctx, platform, platformID, depositItems)

		require.NoError(t, err)
		assert.Equal(t, domain.CompostBinStatusComposting, updatedBin.Status)
		assert.Equal(t, 3, updatedBin.ItemCount)
	})

	t.Run("FeatureLocked_Progression", func(t *testing.T) {
		mockRepo := mocks.NewMockRepositoryCompostRepository(t)
		mockUserRepo := mocks.NewMockRepositoryUser(t)
		mockProgressionSvc := mocks.NewMockProgressionService(t)
		mockJobSvc := mocks.NewMockJobService(t)
		service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

		userID := "user-123"
		user := &domain.User{ID: userID}
		mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Once()
		mockRepo.On("GetBin", ctx, userID).Return(&domain.CompostBin{}, nil).Once()
		mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(false, nil).Once()

		_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{})
		assert.ErrorIs(t, err, domain.ErrFeatureLocked)
	})

	t.Run("FeatureLocked_Job", func(t *testing.T) {
		mockRepo := mocks.NewMockRepositoryCompostRepository(t)
		mockUserRepo := mocks.NewMockRepositoryUser(t)
		mockProgressionSvc := mocks.NewMockProgressionService(t)
		mockJobSvc := mocks.NewMockJobService(t)
		service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

		userID := "user-123"
		user := &domain.User{ID: userID}
		mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Once()
		mockRepo.On("GetBin", ctx, userID).Return(&domain.CompostBin{}, nil).Once()
		mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Once()
		mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(false, nil).Once()

		_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{})
		assert.ErrorIs(t, err, domain.ErrFeatureLocked)
	})

	t.Run("BinFull", func(t *testing.T) {
		mockRepo := mocks.NewMockRepositoryCompostRepository(t)
		mockUserRepo := mocks.NewMockRepositoryUser(t)
		mockProgressionSvc := mocks.NewMockProgressionService(t)
		mockJobSvc := mocks.NewMockJobService(t)
		service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

		userID := "user-123"
		user := &domain.User{ID: userID}
		bin := &domain.CompostBin{UserID: userID, Capacity: 2, ItemCount: 2}
		item := domain.Item{InternalName: "apple", Types: []string{domain.CompostableTag}}

		mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Once()
		mockRepo.On("GetBin", ctx, userID).Return(bin, nil).Once()
		mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Once()
		mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(true, nil).Once()
		mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureCompostCapacity, 3.0).Return(2.0, nil).Once()
		mockUserRepo.On("GetAllItems", ctx).Return([]domain.Item{item}, nil).Once()

		_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{{ItemName: "apple", Quantity: 1}})
		assert.ErrorIs(t, err, domain.ErrCompostBinFull)
	})

	t.Run("ItemNotCompostable", func(t *testing.T) {
		mockRepo := mocks.NewMockRepositoryCompostRepository(t)
		mockUserRepo := mocks.NewMockRepositoryUser(t)
		mockProgressionSvc := mocks.NewMockProgressionService(t)
		mockJobSvc := mocks.NewMockJobService(t)
		service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

		userID := "user-123"
		user := &domain.User{ID: userID}
		bin := &domain.CompostBin{UserID: userID}
		item := domain.Item{InternalName: "stone", Types: []string{}}

		mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Once()
		mockRepo.On("GetBin", ctx, userID).Return(bin, nil).Once()
		mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Once()
		mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(true, nil).Once()
		mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureCompostCapacity, 3.0).Return(5.0, nil).Once()
		mockUserRepo.On("GetAllItems", ctx).Return([]domain.Item{item}, nil).Once()

		_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{{ItemName: "stone", Quantity: 1}})
		assert.ErrorIs(t, err, domain.ErrCompostNotCompostable)
	})

	t.Run("BinNotReady_ReadyStatus", func(t *testing.T) {
		mockRepo := mocks.NewMockRepositoryCompostRepository(t)
		mockUserRepo := mocks.NewMockRepositoryUser(t)
		mockProgressionSvc := mocks.NewMockProgressionService(t)
		mockJobSvc := mocks.NewMockJobService(t)
		service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

		userID := "user-123"
		user := &domain.User{ID: userID}
		bin := &domain.CompostBin{UserID: userID, Status: domain.CompostBinStatusReady}

		mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Once()
		mockRepo.On("GetBin", ctx, userID).Return(bin, nil).Once()
		mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Once()
		mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(true, nil).Once()

		_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{})
		assert.ErrorIs(t, err, domain.ErrCompostMustHarvest)
	})

	t.Run("InsufficientQuantity", func(t *testing.T) {
		mockRepo := mocks.NewMockRepositoryCompostRepository(t)
		mockUserRepo := mocks.NewMockRepositoryUser(t)
		mockProgressionSvc := mocks.NewMockProgressionService(t)
		mockJobSvc := mocks.NewMockJobService(t)
		mockTx := mocks.NewMockRepositoryCompostTx(t)
		service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

		userID := "user-123"
		user := &domain.User{ID: userID}
		bin := &domain.CompostBin{UserID: userID, Capacity: 10}
		item := domain.Item{ID: 1, InternalName: "apple", Types: []string{domain.CompostableTag}}

		mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Once()
		mockRepo.On("GetBin", ctx, userID).Return(bin, nil).Once()
		mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Once()
		mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(true, nil).Once()
		mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureCompostCapacity, 3.0).Return(10.0, nil).Once()
		mockUserRepo.On("GetAllItems", ctx).Return([]domain.Item{item}, nil).Once()

		mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
		mockTx.On("GetBinForUpdate", ctx, userID).Return(bin, nil).Once()
		mockTx.On("GetInventory", ctx, userID).Return(&domain.Inventory{Slots: []domain.InventorySlot{}}, nil).Once() // Empty inventory
		mockTx.On("Rollback", ctx).Return(nil).Once()

		_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{{ItemName: "apple", Quantity: 1}})
		assert.ErrorIs(t, err, domain.ErrInsufficientQuantity)
	})

	t.Run("ResolveDepositItems_Errors", func(t *testing.T) {
		mockRepo := mocks.NewMockRepositoryCompostRepository(t)
		mockUserRepo := mocks.NewMockRepositoryUser(t)
		mockProgressionSvc := mocks.NewMockProgressionService(t)
		mockJobSvc := mocks.NewMockJobService(t)
		service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

		userID := "user-123"
		user := &domain.User{ID: userID}
		bin := &domain.CompostBin{UserID: userID, Status: domain.CompostBinStatusIdle, Capacity: 10}

		mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Maybe()
		mockRepo.On("GetBin", ctx, userID).Return(bin, nil).Maybe()
		mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Maybe()
		mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(true, nil).Maybe()
		mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureCompostCapacity, 3.0).Return(10.0, nil).Maybe()

		// Test: Item Not Found
		mockUserRepo.On("GetAllItems", ctx).Return([]domain.Item{}, nil).Once()
		_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{{ItemName: "ghost_item", Quantity: 1}})
		assert.ErrorIs(t, err, domain.ErrItemNotFound)

		// Test: Invalid Quantity
		item := domain.Item{ID: 1, InternalName: "apple", Types: []string{domain.CompostableTag}}
		mockUserRepo.On("GetAllItems", ctx).Return([]domain.Item{item}, nil).Once()
		_, err = service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{{ItemName: "apple", Quantity: 0}})
		assert.ErrorIs(t, err, domain.ErrInvalidQuantity)
	})

	t.Run("ResolveDepositItems_PublicNameMatch", func(t *testing.T) {
		mockRepo := mocks.NewMockRepositoryCompostRepository(t)
		mockUserRepo := mocks.NewMockRepositoryUser(t)
		mockProgressionSvc := mocks.NewMockProgressionService(t)
		mockJobSvc := mocks.NewMockJobService(t)
		mockTx := mocks.NewMockRepositoryCompostTx(t)
		service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

		userID := "user-123"
		user := &domain.User{ID: userID}
		bin := &domain.CompostBin{UserID: userID, Status: domain.CompostBinStatusIdle, Capacity: 10}
		// Notice the internal name is different from public name.
		item := domain.Item{ID: 1, InternalName: "apple_internal", PublicName: "Apple", Types: []string{domain.CompostableTag}}

		mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Once()
		mockRepo.On("GetBin", ctx, userID).Return(bin, nil).Once()
		mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Once()
		mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(true, nil).Once()
		mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureCompostCapacity, 3.0).Return(10.0, nil).Once()
		mockUserRepo.On("GetAllItems", ctx).Return([]domain.Item{item}, nil).Once()

		mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
		mockTx.On("GetBinForUpdate", ctx, userID).Return(bin, nil).Once()
		mockTx.On("GetInventory", ctx, userID).Return(&domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 1}}}, nil).Once()
		mockTx.On("UpdateInventory", ctx, userID, mock.Anything).Return(nil).Once()
		mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureCompostSpeed, 0.0).Return(1.0, nil).Once()
		mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureSludgeExtension, 0.0).Return(0.0, nil).Once()
		mockTx.On("UpdateBin", ctx, mock.Anything).Return(nil).Once()
		mockTx.On("Commit", ctx).Return(nil).Once()
		mockTx.On("Rollback", ctx).Return(nil).Maybe()

		_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{{ItemName: "Apple", Quantity: 1}})
		require.NoError(t, err)
	})

	t.Run("ExecuteDeposit_DBErrors", func(t *testing.T) {
		userID := "user-123"
		user := &domain.User{ID: userID}
		item := domain.Item{ID: 1, InternalName: "apple", Types: []string{domain.CompostableTag}}

		setupMocks := func(t *testing.T) (*mocks.MockRepositoryCompostRepository, *mocks.MockRepositoryCompostTx, *mocks.MockProgressionService, compost.Service, *domain.CompostBin) {
			mockRepo := mocks.NewMockRepositoryCompostRepository(t)
			mockUserRepo := mocks.NewMockRepositoryUser(t)
			mockProgressionSvc := mocks.NewMockProgressionService(t)
			mockJobSvc := mocks.NewMockJobService(t)
			mockTx := mocks.NewMockRepositoryCompostTx(t)
			service := compost.NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

			bin := &domain.CompostBin{UserID: userID, Status: domain.CompostBinStatusIdle, Capacity: 10}

			mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil).Once()
			mockRepo.On("GetBin", ctx, userID).Return(bin, nil).Once()
			mockProgressionSvc.On("IsFeatureUnlocked", ctx, progression.FeatureCompost).Return(true, nil).Once()
			mockJobSvc.On("IsJobFeatureUnlocked", ctx, userID, progression.FeatureCompost).Return(true, nil).Once()
			mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureCompostCapacity, 3.0).Return(10.0, nil).Once()
			mockUserRepo.On("GetAllItems", ctx).Return([]domain.Item{item}, nil).Once()
			return mockRepo, mockTx, mockProgressionSvc, service, bin
		}

		t.Run("BeginTx_Error", func(t *testing.T) {
			mockRepo, _, _, service, _ := setupMocks(t)
			mockRepo.On("BeginTx", ctx).Return(nil, assert.AnError).Once()
			_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{{ItemName: "apple", Quantity: 1}})
			assert.ErrorContains(t, err, "failed to begin transaction")
		})

		t.Run("GetBinForUpdate_Error", func(t *testing.T) {
			mockRepo, mockTx, _, service, _ := setupMocks(t)
			mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
			mockTx.On("GetBinForUpdate", ctx, userID).Return(nil, assert.AnError).Once()
			mockTx.On("Rollback", ctx).Return(nil).Maybe()
			_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{{ItemName: "apple", Quantity: 1}})
			assert.ErrorContains(t, err, "failed to lock bin")
		})

		t.Run("GetInventory_Error", func(t *testing.T) {
			mockRepo, mockTx, _, service, bin := setupMocks(t)
			mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
			mockTx.On("GetBinForUpdate", ctx, userID).Return(bin, nil).Once()
			mockTx.On("GetInventory", ctx, userID).Return(nil, assert.AnError).Once()
			mockTx.On("Rollback", ctx).Return(nil).Maybe()
			_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{{ItemName: "apple", Quantity: 1}})
			assert.ErrorContains(t, err, "failed to get inventory")
		})

		t.Run("UpdateInventory_Error", func(t *testing.T) {
			mockRepo, mockTx, _, service, bin := setupMocks(t)
			mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
			mockTx.On("GetBinForUpdate", ctx, userID).Return(bin, nil).Once()
			mockTx.On("GetInventory", ctx, userID).Return(&domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 1}}}, nil).Once()
			mockTx.On("UpdateInventory", ctx, userID, mock.Anything).Return(assert.AnError).Once()
			mockTx.On("Rollback", ctx).Return(nil).Maybe()
			_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{{ItemName: "apple", Quantity: 1}})
			assert.ErrorContains(t, err, "failed to update inventory")
		})

		t.Run("UpdateBin_Error", func(t *testing.T) {
			mockRepo, mockTx, mockProgressionSvc, service, bin := setupMocks(t)
			mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureCompostSpeed, 0.0).Return(1.0, nil).Once()
			mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureSludgeExtension, 0.0).Return(0.0, nil).Once()

			mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
			mockTx.On("GetBinForUpdate", ctx, userID).Return(bin, nil).Once()
			mockTx.On("GetInventory", ctx, userID).Return(&domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 1}}}, nil).Once()
			mockTx.On("UpdateInventory", ctx, userID, mock.Anything).Return(nil).Once()
			mockTx.On("UpdateBin", ctx, mock.Anything).Return(assert.AnError).Once()
			mockTx.On("Rollback", ctx).Return(nil).Maybe()
			_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{{ItemName: "apple", Quantity: 1}})
			assert.ErrorContains(t, err, "failed to update bin")
		})

		t.Run("Commit_Error", func(t *testing.T) {
			mockRepo, mockTx, mockProgressionSvc, service, bin := setupMocks(t)
			mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureCompostSpeed, 0.0).Return(1.0, nil).Once()
			mockProgressionSvc.On("GetModifiedValue", ctx, userID, compost.FeatureSludgeExtension, 0.0).Return(0.0, nil).Once()

			mockRepo.On("BeginTx", ctx).Return(mockTx, nil).Once()
			mockTx.On("GetBinForUpdate", ctx, userID).Return(bin, nil).Once()
			mockTx.On("GetInventory", ctx, userID).Return(&domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 1}}}, nil).Once()
			mockTx.On("UpdateInventory", ctx, userID, mock.Anything).Return(nil).Once()
			mockTx.On("UpdateBin", ctx, mock.Anything).Return(nil).Once()
			mockTx.On("Commit", ctx).Return(assert.AnError).Once()
			mockTx.On("Rollback", ctx).Return(nil).Maybe()
			_, err := service.Deposit(ctx, domain.PlatformTwitch, "123", []compost.DepositItem{{ItemName: "apple", Quantity: 1}})
			assert.ErrorContains(t, err, "failed to commit")
		})
	})
}
