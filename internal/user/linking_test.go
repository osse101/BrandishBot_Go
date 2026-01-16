package user

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestMergeUsers(t *testing.T) {
	tests := []struct {
		name               string
		primaryInventory   []domain.InventorySlot
		secondaryInventory []domain.InventorySlot
		expectedSlots      int
		expectedQuantities map[int]int // itemID -> expected quantity
	}{
		{
			name: "merge non-overlapping items",
			primaryInventory: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10},
			},
			secondaryInventory: []domain.InventorySlot{
				{ItemID: 2, Quantity: 5},
			},
			expectedSlots: 2,
			expectedQuantities: map[int]int{
				1: 10,
				2: 5,
			},
		},
		{
			name: "merge overlapping items",
			primaryInventory: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10},
			},
			secondaryInventory: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5},
			},
			expectedSlots: 1,
			expectedQuantities: map[int]int{
				1: 15,
			},
		},
		{
			name: "merge with max stack size cap",
			primaryInventory: []domain.InventorySlot{
				{ItemID: 1, Quantity: 999990},
			},
			secondaryInventory: []domain.InventorySlot{
				{ItemID: 1, Quantity: 100},
			},
			expectedSlots: 1,
			expectedQuantities: map[int]int{
				1: MaxStackSize, // Should cap at 999999
			},
		},
		{
			name:               "merge empty secondary inventory",
			primaryInventory:   []domain.InventorySlot{{ItemID: 1, Quantity: 10}},
			secondaryInventory: []domain.InventorySlot{},
			expectedSlots:      1,
			expectedQuantities: map[int]int{
				1: 10,
			},
		},
		{
			name:             "merge into empty primary inventory",
			primaryInventory: []domain.InventorySlot{},
			secondaryInventory: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10},
			},
			expectedSlots: 1,
			expectedQuantities: map[int]int{
				1: 10,
			},
		},
		{
			name: "merge multiple items with partial overlap",
			primaryInventory: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10},
				{ItemID: 2, Quantity: 20},
			},
			secondaryInventory: []domain.InventorySlot{
				{ItemID: 2, Quantity: 5},
				{ItemID: 3, Quantity: 15},
			},
			expectedSlots: 3,
			expectedQuantities: map[int]int{
				1: 10,
				2: 25,
				3: 15,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepository()
			svc := &service{
				repo:            repo,
				userCache:       newUserCache(CacheConfig{Size: 100, TTL: 0}),
				itemCacheByName: make(map[string]domain.Item),
				itemIDToName:    make(map[int]string),
			}

			// Create test users
			primary := &domain.User{
				ID:       "primary-user",
				Username: "primary",
				TwitchID: "twitch-primary",
			}
			secondary := &domain.User{
				ID:       "secondary-user",
				Username: "secondary",
				TwitchID: "twitch-secondary",
			}

			repo.users["primary"] = primary
			repo.users["secondary"] = secondary

			// Set up inventories
			repo.inventories[primary.ID] = &domain.Inventory{Slots: tt.primaryInventory}
			repo.inventories[secondary.ID] = &domain.Inventory{Slots: tt.secondaryInventory}

			// Merge users
			err := svc.MergeUsers(context.Background(), primary.ID, secondary.ID)
			if err != nil {
				t.Fatalf("MergeUsers failed: %v", err)
			}

			// Verify merged inventory
			mergedInv := repo.inventories[primary.ID]
			if len(mergedInv.Slots) != tt.expectedSlots {
				t.Errorf("expected %d slots, got %d", tt.expectedSlots, len(mergedInv.Slots))
			}

			// Verify quantities
			for itemID, expectedQty := range tt.expectedQuantities {
				found := false
				for _, slot := range mergedInv.Slots {
					if slot.ItemID == itemID {
						if slot.Quantity != expectedQty {
							t.Errorf("item %d: expected quantity %d, got %d", itemID, expectedQty, slot.Quantity)
						}
						found = true
						break
					}
				}
				if !found {
					t.Errorf("item %d not found in merged inventory", itemID)
				}
			}
		})
	}
}

func TestMergeUsers_PlatformMerge(t *testing.T) {
	tests := []struct {
		name              string
		primaryUser       domain.User
		secondaryUser     domain.User
		expectedDiscordID string
		expectedTwitchID  string
		expectedYoutubeID string
	}{
		{
			name: "merge non-overlapping platforms",
			primaryUser: domain.User{
				ID:       "primary",
				Username: "primary",
				TwitchID: "twitch-primary",
			},
			secondaryUser: domain.User{
				ID:        "secondary",
				Username:  "secondary",
				DiscordID: "discord-secondary",
			},
			expectedTwitchID:  "twitch-primary",
			expectedDiscordID: "discord-secondary",
		},
		{
			name: "primary platform wins on overlap",
			primaryUser: domain.User{
				ID:        "primary",
				Username:  "primary",
				DiscordID: "discord-primary",
			},
			secondaryUser: domain.User{
				ID:        "secondary",
				Username:  "secondary",
				DiscordID: "discord-secondary",
			},
			expectedDiscordID: "discord-primary",
		},
		{
			name: "merge all three platforms",
			primaryUser: domain.User{
				ID:        "primary",
				Username:  "primary",
				DiscordID: "discord-primary",
			},
			secondaryUser: domain.User{
				ID:        "secondary",
				Username:  "secondary",
				TwitchID:  "twitch-secondary",
				YoutubeID: "youtube-secondary",
			},
			expectedDiscordID: "discord-primary",
			expectedTwitchID:  "twitch-secondary",
			expectedYoutubeID: "youtube-secondary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepository()
			svc := &service{
				repo:            repo,
				userCache:       newUserCache(CacheConfig{Size: 100, TTL: 0}),
				itemCacheByName: make(map[string]domain.Item),
				itemIDToName:    make(map[int]string),
			}

			repo.users["primary"] = &tt.primaryUser
			repo.users["secondary"] = &tt.secondaryUser
			repo.inventories[tt.primaryUser.ID] = &domain.Inventory{Slots: []domain.InventorySlot{}}
			repo.inventories[tt.secondaryUser.ID] = &domain.Inventory{Slots: []domain.InventorySlot{}}

			err := svc.MergeUsers(context.Background(), tt.primaryUser.ID, tt.secondaryUser.ID)
			if err != nil {
				t.Fatalf("MergeUsers failed: %v", err)
			}

			// Verify merged user
			mergedUser, _ := repo.GetUserByID(context.Background(), tt.primaryUser.ID)
			if mergedUser.DiscordID != tt.expectedDiscordID {
				t.Errorf("expected DiscordID %s, got %s", tt.expectedDiscordID, mergedUser.DiscordID)
			}
			if mergedUser.TwitchID != tt.expectedTwitchID {
				t.Errorf("expected TwitchID %s, got %s", tt.expectedTwitchID, mergedUser.TwitchID)
			}
			if mergedUser.YoutubeID != tt.expectedYoutubeID {
				t.Errorf("expected YoutubeID %s, got %s", tt.expectedYoutubeID, mergedUser.YoutubeID)
			}
		})
	}
}

func TestUnlinkPlatform(t *testing.T) {
	tests := []struct {
		name          string
		platform      string
		initialUser   domain.User
		expectedField string // which field should be empty after unlink
		expectError   bool
	}{
		{
			name:     "unlink discord",
			platform: domain.PlatformDiscord,
			initialUser: domain.User{
				ID:        "user-1",
				Username:  "testuser",
				DiscordID: "discord-123",
				TwitchID:  "twitch-456",
			},
			expectedField: "DiscordID",
			expectError:   false,
		},
		{
			name:     "unlink twitch",
			platform: domain.PlatformTwitch,
			initialUser: domain.User{
				ID:        "user-1",
				Username:  "testuser",
				DiscordID: "discord-123",
				TwitchID:  "twitch-456",
			},
			expectedField: "TwitchID",
			expectError:   false,
		},
		{
			name:     "unlink youtube",
			platform: domain.PlatformYoutube,
			initialUser: domain.User{
				ID:        "user-1",
				Username:  "testuser",
				YoutubeID: "youtube-789",
			},
			expectedField: "YoutubeID",
			expectError:   false,
		},
		{
			name:     "unlink unknown platform",
			platform: "unknown",
			initialUser: domain.User{
				ID:        "user-1",
				Username:  "testuser",
				DiscordID: "discord-123",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepository()
			svc := &service{
				repo:            repo,
				userCache:       newUserCache(CacheConfig{Size: 100, TTL: 0}),
				itemCacheByName: make(map[string]domain.Item),
				itemIDToName:    make(map[int]string),
			}

			repo.users[tt.initialUser.Username] = &tt.initialUser

			err := svc.UnlinkPlatform(context.Background(), tt.initialUser.ID, tt.platform)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("UnlinkPlatform failed: %v", err)
			}

			// Verify platform was unlinked
			user, _ := repo.GetUserByID(context.Background(), tt.initialUser.ID)
			switch tt.expectedField {
			case "DiscordID":
				if user.DiscordID != "" {
					t.Errorf("DiscordID should be empty, got %s", user.DiscordID)
				}
			case "TwitchID":
				if user.TwitchID != "" {
					t.Errorf("TwitchID should be empty, got %s", user.TwitchID)
				}
			case "YoutubeID":
				if user.YoutubeID != "" {
					t.Errorf("YoutubeID should be empty, got %s", user.YoutubeID)
				}
			}
		})
	}
}

func TestGetLinkedPlatforms(t *testing.T) {
	tests := []struct {
		name              string
		user              domain.User
		queryPlatform     string
		queryPlatformID   string
		expectedPlatforms []string
	}{
		{
			name: "all platforms",
			user: domain.User{
				ID:        "user-1",
				Username:  "testuser",
				DiscordID: "discord-1",
				TwitchID:  "twitch-2",
				YoutubeID: "youtube-3",
			},
			queryPlatform:     domain.PlatformTwitch,
			queryPlatformID:   "twitch-2",
			expectedPlatforms: []string{domain.PlatformDiscord, domain.PlatformTwitch, domain.PlatformYoutube},
		},
		{
			name: "discord only",
			user: domain.User{
				ID:        "user-1",
				Username:  "testuser",
				DiscordID: "discord-1",
			},
			queryPlatform:     domain.PlatformDiscord,
			queryPlatformID:   "discord-1",
			expectedPlatforms: []string{domain.PlatformDiscord},
		},
		{
			name: "twitch and youtube",
			user: domain.User{
				ID:        "user-1",
				Username:  "testuser",
				TwitchID:  "twitch-2",
				YoutubeID: "youtube-3",
			},
			queryPlatform:     domain.PlatformTwitch,
			queryPlatformID:   "twitch-2",
			expectedPlatforms: []string{domain.PlatformTwitch, domain.PlatformYoutube},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepository()
			svc := &service{
				repo:            repo,
				userCache:       newUserCache(CacheConfig{Size: 100, TTL: 0}),
				itemCacheByName: make(map[string]domain.Item),
				itemIDToName:    make(map[int]string),
			}

			repo.users[tt.user.Username] = &tt.user

			platforms, err := svc.GetLinkedPlatforms(context.Background(), tt.queryPlatform, tt.queryPlatformID)

			if err != nil {
				t.Fatalf("GetLinkedPlatforms failed: %v", err)
			}

			// Verify platform count
			if len(platforms) != len(tt.expectedPlatforms) {
				t.Errorf("expected %d platforms, got %d", len(tt.expectedPlatforms), len(platforms))
			}

			// Verify all expected platforms are present
			platformMap := make(map[string]bool)
			for _, p := range platforms {
				platformMap[p] = true
			}
			for _, expected := range tt.expectedPlatforms {
				if !platformMap[expected] {
					t.Errorf("expected platform %s not found", expected)
				}
			}
		})
	}
}
