package harvest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHarvest_Workflow(t *testing.T) {
	defaultUser := &domain.User{
		ID:       "user-123",
		Username: "TestUser",
	}

	tests := []struct {
		name              string
		hoursElapsed      float64
		featureLocked     bool
		userNotFound      bool // If true, simulate user not found initially
		upsertFail        bool
		firstTimeHarvest  bool // If true, simulate no harvest state
		tooSoon           bool // If true, override hoursElapsed to be small
		allRewardsLocked  bool // If true, lock all items
		commitFail        bool // If true, simulate commit failure
		expectedGains     map[string]int
		expectedXPAward   bool
		expectedXP        int
		expectedSpoiled   bool
		expectedError     bool
		expectedErrorText string
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
			name:              "Harvest Locked - Feature Farming Locked",
			hoursElapsed:      10.0,
			featureLocked:     true,
			expectedError:     true,
			expectedErrorText: "harvest requires farming feature to be unlocked",
		},
		{
			name:         "Farmer XP Harvest - 6 hours",
			hoursElapsed: 6.0, // Tier 2 (12 money)
			expectedGains: map[string]int{
				"money": 12,
			},
			expectedXPAward: true,
			expectedXP:      48,
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
			expectedXP:      2720,
			expectedSpoiled: true,
		},
		{
			name:          "User Registration",
			hoursElapsed:  2.0,
			userNotFound:  true,
			expectedGains: map[string]int{"money": 2},
		},
		{
			name:             "First Time Harvest",
			hoursElapsed:     0.0,
			firstTimeHarvest: true,
			expectedGains:    map[string]int{}, // Empty for first time
		},
		{
			name:              "Harvest Too Soon",
			hoursElapsed:      0.5,
			tooSoon:           true,
			expectedError:     true,
			expectedErrorText: "next harvest available at",
		},
		{
			name:             "Empty Rewards (Tier not reached)",
			hoursElapsed:     1.5, // > 1.0 (min harvest) but < 2.0 (Tier 1)
			allRewardsLocked: true,
			expectedGains:    map[string]int{}, // Empty
		},
		{
			name:              "Transaction Commit Error",
			hoursElapsed:      2.0,
			commitFail:        true,
			expectedGains:     map[string]int{"money": 2}, // Needs gains to reach commit
			expectedError:     true,
			expectedErrorText: "failed to commit transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockHarvestRepo := mocks.NewMockRepositoryHarvestRepository(t)
			mockUserRepo := new(mocks.MockRepositoryUser)
			mockProgressionSvc := new(mocks.MockProgressionService)
			mockJobSvc := mocks.NewMockJobService(t)
			mockTx := mocks.NewMockRepositoryHarvestTx(t)

			svc := NewService(mockHarvestRepo, mockUserRepo, mockProgressionSvc, mockJobSvc)

			// --- User Registration Workflow ---
			if tt.userNotFound {
				mockUserRepo.On("GetUserByPlatformID", mock.Anything, "discord", "123456").Return(nil, domain.ErrUserNotFound).Once()
				mockUserRepo.On("UpsertUser", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
					return u.DiscordID == "123456" && u.Username == "TestUser"
				})).Return(nil).Once()
				mockUserRepo.On("GetUserByPlatformID", mock.Anything, "discord", "123456").Return(defaultUser, nil).Once()
			} else {
				mockUserRepo.On("GetUserByPlatformID", mock.Anything, "discord", "123456").Return(defaultUser, nil)
			}

			// --- Feature Unlock ---
			// Don't setup IsFeatureUnlocked if user retrieval fails (which isn't tested here, but good practice)
			// But here we assume user retrieval succeeds eventually
			mockProgressionSvc.On("IsFeatureUnlocked", mock.Anything, "feature_farming").Return(!tt.featureLocked, nil)

			if tt.featureLocked {
				_, err := svc.Harvest(context.Background(), "discord", "123456", "TestUser")
				assert.Error(t, err)
				if tt.expectedErrorText != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorText)
				}
				return
			}

			// --- Harvest State Retrieval ---
			if tt.firstTimeHarvest {
				mockHarvestRepo.On("GetHarvestState", mock.Anything, defaultUser.ID).Return(nil, domain.ErrHarvestStateNotFound)
				mockHarvestRepo.On("CreateHarvestState", mock.Anything, defaultUser.ID).Return(&domain.HarvestState{
					LastHarvestedAt: time.Now(),
				}, nil)

				// Execute and Assert for First Time
				resp, err := svc.Harvest(context.Background(), "discord", "123456", "TestUser")
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, 0.0, resp.HoursSinceHarvest)
				assert.Empty(t, resp.ItemsGained)
				return
			}

			// Normal flow: GetHarvestState returns state
			lastHarvested := time.Now().Add(-time.Duration(tt.hoursElapsed * float64(time.Hour)))
			mockHarvestRepo.On("GetHarvestState", mock.Anything, defaultUser.ID).Return(&domain.HarvestState{
				LastHarvestedAt: lastHarvested,
			}, nil)

			// --- Transaction Start ---
			mockHarvestRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
			// Defer rollback is called (using Maybe because safe rollback checks error)
			mockTx.On("Rollback", mock.Anything).Return(nil).Maybe()

			// --- Get State With Lock ---
			mockTx.On("GetHarvestStateWithLock", mock.Anything, defaultUser.ID).Return(&domain.HarvestState{
				LastHarvestedAt: lastHarvested,
			}, nil)

			// --- Validate Minimum Time ---
			// Logic handles this after lock
			if tt.tooSoon {
				// Execute
				_, err := svc.Harvest(context.Background(), "discord", "123456", "TestUser")
				assert.Error(t, err)
				if tt.expectedErrorText != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorText)
				}
				return
			}

			// --- Calculate Rewards ---
			if !tt.expectedSpoiled {
				if tt.allRewardsLocked {
					// Setup IsItemUnlocked to return false
					mockProgressionSvc.On("IsItemUnlocked", mock.Anything, mock.Anything).Return(false, nil).Maybe()
				} else {
					// Basic unlocks
					mockProgressionSvc.On("IsItemUnlocked", mock.Anything, mock.Anything).Return(true, nil).Maybe()
				}
			}

			// --- Award XP ---
			if tt.expectedXPAward {
				mockJobSvc.On("AwardXP", mock.Anything, defaultUser.ID, job.JobKeyFarmer, tt.expectedXP, job.SourceHarvest, mock.Anything).Return(&domain.XPAwardResult{XPGained: tt.expectedXP}, nil)
			}

			// --- Empty Rewards Warning Path ---
			if tt.allRewardsLocked && !tt.expectedSpoiled {
				// Should update timestamp and commit, but NOT update inventory
				mockTx.On("UpdateHarvestState", mock.Anything, defaultUser.ID, mock.Anything).Return(nil)
				mockTx.On("Commit", mock.Anything).Return(nil)

				resp, err := svc.Harvest(context.Background(), "discord", "123456", "TestUser")
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Empty(t, resp.ItemsGained)
				assert.Contains(t, resp.Message, "No rewards available")
				return
			}

			// --- Inventory & Item Handling ---
			mockTx.On("GetInventory", mock.Anything, defaultUser.ID).Return(&domain.Inventory{Slots: []domain.InventorySlot{}}, nil)

			// Mock item lookup
			mockItems := []domain.Item{}
			for name := range tt.expectedGains {
				mockItems = append(mockItems, domain.Item{InternalName: name, ID: 1, PublicName: name})
				mockUserRepo.On("GetItemsByNames", mock.Anything, mock.Anything).Return(func(_ context.Context, names []string) []domain.Item {
					return mockItems
				}, nil)
			}

			// Update Inventory
			mockTx.On("UpdateInventory", mock.Anything, defaultUser.ID, mock.Anything).Return(nil)

			// Update Timestamp
			mockTx.On("UpdateHarvestState", mock.Anything, defaultUser.ID, mock.Anything).Return(nil)

			// Commit
			if tt.commitFail {
				mockTx.On("Commit", mock.Anything).Return(errors.New("db error"))
			} else {
				mockTx.On("Commit", mock.Anything).Return(nil)
			}

			// Execute
			resp, err := svc.Harvest(context.Background(), "discord", "123456", "TestUser")

			// Verify
			if tt.expectedError {
				assert.Error(t, err)
				if tt.expectedErrorText != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorText)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.expectedGains, resp.ItemsGained)

				if tt.expectedSpoiled {
					assert.Contains(t, resp.Message, "spoiled")
				}
			}
		})
	}
}
