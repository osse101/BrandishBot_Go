package harvest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/mocks"
)

// MockHarvestRepository is a manual mock for HarvestRepository
type MockHarvestRepository struct {
	mock.Mock
}

func (m *MockHarvestRepository) GetHarvestState(ctx context.Context, userID string) (*domain.HarvestState, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.HarvestState), args.Error(1)
}

func (m *MockHarvestRepository) CreateHarvestState(ctx context.Context, userID string) (*domain.HarvestState, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.HarvestState), args.Error(1)
}

func (m *MockHarvestRepository) BeginTx(ctx context.Context) (repository.HarvestTx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(repository.HarvestTx), args.Error(1)
}

// MockHarvestTx is a manual mock for HarvestTx
type MockHarvestTx struct {
	mock.Mock
}

func (m *MockHarvestTx) Commit(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockHarvestTx) Rollback(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockHarvestTx) GetHarvestStateWithLock(ctx context.Context, userID string) (*domain.HarvestState, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.HarvestState), args.Error(1)
}

func (m *MockHarvestTx) UpdateHarvestState(ctx context.Context, userID string, lastHarvestedAt time.Time) error {
	return m.Called(ctx, userID, lastHarvestedAt).Error(0)
}

func (m *MockHarvestTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Inventory), args.Error(1)
}

func (m *MockHarvestTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return m.Called(ctx, userID, inventory).Error(0)
}

func TestHarvest_Integration(t *testing.T) {
	user := &domain.User{
		ID:       "user-123",
		Username: "TestUser",
	}

	tests := []struct {
		name            string
		hoursElapsed    float64
		expectedGains   map[string]int
		expectedXPAward bool
		expectedXP      int
		expectedSpoiled bool
	}{
		{
			name:         "Normal Harvest - No XP (less than 5h)",
			hoursElapsed: 2.0, // Tier 1 (2 money)
			expectedGains: map[string]int{
				"money": 2,
			},
			expectedXPAward: false,
			expectedSpoiled: false,
		},
		{
			name:         "Farmer XP Harvest - 6 hours",
			hoursElapsed: 6.0, // Tier 2 (12 money)
			expectedGains: map[string]int{
				"money": 12,
			},
			expectedXPAward: true,
			expectedXP:      60,
			expectedSpoiled: false,
		},
		{
			name:         "Spoiled Harvest - > 336 hours",
			hoursElapsed: 340.0,
			expectedGains: map[string]int{
				"lootbox1": 1,
				"stick":    3,
			},
			expectedXPAward: true,
			expectedXP:      3400,
			expectedSpoiled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks per test
			mockHarvestRepo := new(MockHarvestRepository)
			mockUserRepo := new(mocks.MockRepositoryUser)
			mockProgressionSvc := new(mocks.MockProgressionService)
			mockJobSvc := mocks.NewMockJobService(t)

			svc := NewService(mockHarvestRepo, mockUserRepo, mockProgressionSvc, mockJobSvc)

			// --- Mock Setup for User & State Retrieval ---
			mockUserRepo.On("GetUserByPlatformID", mock.Anything, "discord", "123456").Return(user, nil)

			// Helper to calculate last harvested based on elapsed
			lastHarvested := time.Now().Add(-time.Duration(tt.hoursElapsed * float64(time.Hour)))

			mockHarvestRepo.On("GetHarvestState", mock.Anything, user.ID).Return(&domain.HarvestState{
				LastHarvestedAt: lastHarvested,
			}, nil)

			// Transaction Mocks
			mockTx := new(MockHarvestTx)
			mockHarvestRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
			mockTx.On("GetHarvestStateWithLock", mock.Anything, user.ID).Return(&domain.HarvestState{
				LastHarvestedAt: lastHarvested,
			}, nil)

			// Unlock checks (assume basic unlocks for rewards)
			if !tt.expectedSpoiled {
				// Only check progression if NOT spoiled
				// Mock progression checks for calculating rewards
				// For simplicity, assume items are unlocked if checked
				mockProgressionSvc.On("IsItemUnlocked", mock.Anything, mock.Anything).Return(true, nil).Maybe()
			}

			// Inventory & Item Handling
			mockTx.On("GetInventory", mock.Anything, user.ID).Return(&domain.Inventory{Slots: []domain.InventorySlot{}}, nil)

			// Item Details Mock
			mockItems := []domain.Item{}
			for name := range tt.expectedGains {
				mockItems = append(mockItems, domain.Item{InternalName: name, ID: 1, PublicName: name})
				mockUserRepo.On("GetItemsByNames", mock.Anything, mock.Anything).Return(func(_ context.Context, names []string) []domain.Item {
					// Minimal check, just satisfy return
					return mockItems
				}, nil)
			}

			// Expect UpdateInventory
			mockTx.On("UpdateInventory", mock.Anything, user.ID, mock.Anything).Return(nil)
			mockTx.On("UpdateHarvestState", mock.Anything, user.ID, mock.Anything).Return(nil)
			mockTx.On("Commit", mock.Anything).Return(nil)
			mockTx.On("Rollback", mock.Anything).Return(nil).Maybe()

			// Specific API/Test Logic Setup
			// We need to pass the mocks to the setup function now
			// But setupMocks signature is func()
			// We can redefine setupMocks to accept mocks or just access them via closure if we move this definition inside (which we can't easily)
			// OR we can assign the closure in the test struct to a helper that takes the mocks.

			// Simplified: We define expectations inline here based on test case fields
			if tt.expectedXPAward {
				mockJobSvc.On("AwardXP", mock.Anything, user.ID, job.JobKeyFarmer, tt.expectedXP, job.SourceHarvest, mock.Anything).Return(&domain.XPAwardResult{XPGained: tt.expectedXP}, nil)
			}

			// Execute
			resp, err := svc.Harvest(context.Background(), "discord", "123456", "TestUser")

			// Verify
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.expectedGains, resp.ItemsGained)

			if tt.expectedSpoiled {
				assert.Contains(t, resp.Message, "spoiled")
			}
		})
	}
}
