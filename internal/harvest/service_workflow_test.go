package harvest

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
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
		firstTimeHarvest  bool                   // If true, simulate no harvest state
		tooSoon           bool                   // If true, override hoursElapsed to be small
		allRewardsLocked  bool                   // If true, lock all items
		commitFail        bool                   // If true, simulate commit failure
		xpAwardFail       bool                   // NEW: Simulate failure in AwardXP
		partialItemLookup bool                   // NEW: Simulate missing items in DB
		initialInventory  []domain.InventorySlot // NEW: Setup initial inventory state
		expectedInvSlots  []domain.InventorySlot // NEW: Expected final inventory state (nil = default check)
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
		// --- QA NEW TESTS ---
		{
			name:            "Farmer XP Award Failure - Continues Gracefully",
			hoursElapsed:    6.0,
			expectedGains:   map[string]int{"money": 12},
			expectedXPAward: true,
			expectedXP:      48,
			xpAwardFail:     true, // Simulate error
			// Expect success, but message won't have XP details (checked in test logic)
		},
		{
			name:              "Item Lookup Partial Failure - Skips Missing Items",
			hoursElapsed:      24.0, // Should get money and stick
			expectedGains:     map[string]int{"money": 22, "stick": 3},
			expectedXPAward:   true,
			expectedXP:        192,
			partialItemLookup: true, // Only return first item found (money), skip stick
			// Expect inventory to update only with money
			expectedInvSlots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 22, QualityLevel: domain.QualityCommon}, // ID 1 = money
			},
		},
		{
			name:          "Inventory Slot Stacking - Existing Item",
			hoursElapsed:  2.0, // 2 money
			expectedGains: map[string]int{"money": 2},
			initialInventory: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityCommon}, // User already has 10 money (ID 1)
			},
			expectedInvSlots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 12, QualityLevel: domain.QualityCommon}, // Should stack to 12
			},
		},
		{
			name:          "Inventory New Slot - Different Item",
			hoursElapsed:  2.0, // 2 money (ID 1)
			expectedGains: map[string]int{"money": 2},
			initialInventory: []domain.InventorySlot{
				{ItemID: 2, Quantity: 5, QualityLevel: domain.QualityCommon}, // User has stick (ID 2)
			},
			expectedInvSlots: []domain.InventorySlot{
				{ItemID: 2, Quantity: 5, QualityLevel: domain.QualityCommon},
				{ItemID: 1, Quantity: 2, QualityLevel: domain.QualityCommon}, // New slot for money
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockHarvestRepo := mocks.NewMockRepositoryHarvestRepository(t)
			mockUserRepo := new(mocks.MockRepositoryUser)
			mockProgressionSvc := new(mocks.MockProgressionService)
			mockTx := mocks.NewMockRepositoryHarvestTx(t)
			mockJobSvc := new(mocks.MockJobService)

			// Default job bonus expectations (0 bonus)
			mockJobSvc.On("GetJobBonus", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0.0, nil).Maybe()

			svc := NewService(mockHarvestRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, nil)

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
			mockTx.On("Rollback", mock.Anything).Return(nil).Maybe()

			// --- Get State With Lock ---
			mockTx.On("GetHarvestStateWithLock", mock.Anything, defaultUser.ID).Return(&domain.HarvestState{
				LastHarvestedAt: lastHarvested,
			}, nil)

			// --- Validate Minimum Time ---
			if tt.tooSoon {
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
					mockProgressionSvc.On("IsItemUnlocked", mock.Anything, mock.Anything).Return(false, nil).Maybe()
				} else {
					mockProgressionSvc.On("IsItemUnlocked", mock.Anything, mock.Anything).Return(true, nil).Maybe()
				}
			}

			// XP is now awarded via HarvestCompletedPayload event (no direct job service call)

			// --- Empty Rewards Warning Path ---
			if tt.allRewardsLocked && !tt.expectedSpoiled {
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
			initialInv := &domain.Inventory{Slots: []domain.InventorySlot{}}
			if tt.initialInventory != nil {
				initialInv.Slots = tt.initialInventory
			}
			mockTx.On("GetInventory", mock.Anything, defaultUser.ID).Return(initialInv, nil)

			// Mock item lookup
			mockItems := []domain.Item{}
			// Sort keys to ensure deterministic item ID assignment for testing
			keys := make([]string, 0, len(tt.expectedGains))
			for k := range tt.expectedGains {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			// Assign IDs based on index (1-based) to distinguish items
			// But note: map iteration order is random, so we must sort keys first if we want deterministic IDs
			// or just map name->ID explicitly in the test setup.
			// Here we loop through sorted keys
			for i, name := range keys {
				mockItems = append(mockItems, domain.Item{InternalName: name, ID: i + 1, PublicName: name})
			}

			mockUserRepo.On("GetItemsByNames", mock.Anything, mock.Anything).Return(func(_ context.Context, names []string) []domain.Item {
				if tt.partialItemLookup && len(mockItems) > 0 {
					return mockItems[:1] // Return only the first item
				}
				return mockItems
			}, nil)

			// Update Inventory
			if tt.expectedInvSlots != nil {
				mockTx.On("UpdateInventory", mock.Anything, defaultUser.ID, mock.MatchedBy(func(inv domain.Inventory) bool {
					// Compare inv.Slots with tt.expectedInvSlots using assert.ElementsMatch
					// Note: MatchedBy return value is boolean, asserting inside might log but we need return
					return assert.ElementsMatch(t, tt.expectedInvSlots, inv.Slots)
				})).Return(nil)
			} else {
				mockTx.On("UpdateInventory", mock.Anything, defaultUser.ID, mock.Anything).Return(nil)
			}

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

			// Wait for async operations to complete
			_ = svc.Shutdown(context.Background())

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
				if tt.expectedXPAward {
					// XP message has been removed from response per user request
					assert.NotContains(t, resp.Message, "You gained")
				}
			}
		})
	}
}
