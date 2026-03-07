package user

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

func setupInventoryEventTest(t *testing.T) (*event.MemoryBus, *FakeRepository, *service) {
	bus := event.NewMemoryBus()
	tmpFile, err := os.CreateTemp("", "deadletter")
	require.NoError(t, err)
	t.Cleanup(func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	})

	publisher, err := event.NewResilientPublisher(bus, 1, time.Millisecond, tmpFile.Name())
	require.NoError(t, err)
	t.Cleanup(func() { publisher.Shutdown(context.Background()) })

	repo := NewFakeRepository()
	// Populate items
	stickItem := &domain.Item{
		ID:           1,
		InternalName: domain.ItemStick,
		PublicName:   domain.PublicNameStick,
		BaseValue:    10,
	}
	repo.items[domain.ItemStick] = stickItem
	moneyItem := &domain.Item{
		ID:           2,
		InternalName: domain.ItemMoney,
		PublicName:   domain.PublicNameMoney,
		BaseValue:    1,
	}
	repo.items[domain.ItemMoney] = moneyItem

	svc := NewService(repo, repo, nil, publisher, nil, NewMockNamingResolver(), nil, nil, nil, nil, false).(*service)
	return bus, repo, svc
}

func TestInventoryEvents_ItemAdded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		itemName         string
		quantity         int
		setupUser        func(context.Context, *FakeRepository) *domain.User
		expectEvent      bool
		expectedError    string
		expectedQuantity int
	}{
		{
			name:     "Best Case - Add valid item",
			itemName: domain.ItemStick,
			quantity: 5,
			setupUser: func(ctx context.Context, repo *FakeRepository) *domain.User {
				user := &domain.User{ID: "user_added", Username: "user_added", TwitchID: "twitch_added"}
				repo.UpsertUser(ctx, user)
				return user
			},
			expectEvent: true,
		},
		// AddItemByUsername doesn't explicitly validate quantity at the service level,
		{
			name:     "Boundary Case - Zero quantity",
			itemName: domain.ItemStick,
			quantity: 0,
			setupUser: func(ctx context.Context, repo *FakeRepository) *domain.User {
				user := &domain.User{ID: "user_added_zero", Username: "user_added_zero", TwitchID: "twitch_added_zero"}
				repo.UpsertUser(ctx, user)
				return user
			},
			expectEvent: true,
		},
		{
			name:     "Boundary Case - Negative quantity",
			itemName: domain.ItemStick,
			quantity: -1,
			setupUser: func(ctx context.Context, repo *FakeRepository) *domain.User {
				user := &domain.User{ID: "user_added_neg", Username: "user_added_neg", TwitchID: "twitch_added_neg"}
				repo.UpsertUser(ctx, user)
				return user
			},
			expectEvent: true,
		},
		{
			name:     "Invalid Case - Unknown item",
			itemName: "unknown_item",
			quantity: 1,
			setupUser: func(ctx context.Context, repo *FakeRepository) *domain.User {
				user := &domain.User{ID: "user_added_inv", Username: "user_added_inv", TwitchID: "twitch_added_inv"}
				repo.UpsertUser(ctx, user)
				return user
			},
			expectEvent:   false,
			expectedError: domain.ErrMsgItemNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bus, repo, svc := setupInventoryEventTest(t)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			user := tt.setupUser(ctx, repo)

			received := make(chan event.Event, 1)
			bus.Subscribe(event.Type(domain.EventTypeItemAdded), func(ctx context.Context, e event.Event) error {
				select {
				case received <- e:
				default:
				}
				return nil
			})

			err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, user.Username, tt.itemName, tt.quantity)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, received, "Expected no event to be emitted")
				return
			}

			require.NoError(t, err)

			if tt.expectEvent {
				select {
				case e := <-received:
					assert.Equal(t, event.Type(domain.EventTypeItemAdded), e.Type)
					payload, ok := e.Payload.(domain.ItemAddedPayload)
					require.True(t, ok)
					assert.Equal(t, tt.itemName, payload.ItemName)
					assert.Equal(t, tt.quantity, payload.Quantity)
				case <-ctx.Done():
					t.Fatal("timeout waiting for event")
				}
			}
		})
	}
}

func TestInventoryEvents_ItemRemoved(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		itemName         string
		quantity         int
		setupUser        func(context.Context, *FakeRepository) *domain.User
		expectEvent      bool
		expectedError    string
		expectedQuantity int
	}{
		{
			name:     "Best Case - Remove valid quantity",
			itemName: domain.ItemStick,
			quantity: 3,
			setupUser: func(ctx context.Context, repo *FakeRepository) *domain.User {
				user := &domain.User{ID: "user_removed", Username: "user_removed", TwitchID: "twitch_removed"}
				repo.UpsertUser(ctx, user)
				inv := &domain.Inventory{
					Slots: []domain.InventorySlot{
						{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityCommon},
					},
				}
				repo.UpdateInventory(ctx, user.ID, *inv)
				return user
			},
			expectEvent:      true,
			expectedQuantity: 3,
		},
		{
			name:     "Boundary Case - Remove exact amount",
			itemName: domain.ItemStick,
			quantity: 10,
			setupUser: func(ctx context.Context, repo *FakeRepository) *domain.User {
				user := &domain.User{ID: "user_removed_exact", Username: "user_removed_exact", TwitchID: "twitch_removed_exact"}
				repo.UpsertUser(ctx, user)
				inv := &domain.Inventory{
					Slots: []domain.InventorySlot{
						{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityCommon},
					},
				}
				repo.UpdateInventory(ctx, user.ID, *inv)
				return user
			},
			expectEvent:      true,
			expectedQuantity: 10,
		},
		// Since RemoveItemByUsername removes up to the amount requested and returns the amount removed,
		// it actually succeeds without an error but only emits an event for the amount it actually removed.
		{
			name:     "Boundary Case - Remove more than owned",
			itemName: domain.ItemStick,
			quantity: 11,
			setupUser: func(ctx context.Context, repo *FakeRepository) *domain.User {
				user := &domain.User{ID: "user_removed_more", Username: "user_removed_more", TwitchID: "twitch_removed_more"}
				repo.UpsertUser(ctx, user)
				inv := &domain.Inventory{
					Slots: []domain.InventorySlot{
						{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityCommon},
					},
				}
				repo.UpdateInventory(ctx, user.ID, *inv)
				return user
			},
			expectEvent:      true,
			expectedError:    "", // It doesn't error, just removes what it can (10)
			expectedQuantity: 10,
		},
		{
			name:     "Invalid Case - Remove item not in inventory",
			itemName: domain.ItemMoney,
			quantity: 1,
			setupUser: func(ctx context.Context, repo *FakeRepository) *domain.User {
				user := &domain.User{ID: "user_removed_not_owned", Username: "user_removed_not_owned", TwitchID: "twitch_removed_not_owned"}
				repo.UpsertUser(ctx, user)
				inv := &domain.Inventory{
					Slots: []domain.InventorySlot{
						{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityCommon},
					},
				}
				repo.UpdateInventory(ctx, user.ID, *inv)
				return user
			},
			expectEvent:   false,
			expectedError: domain.ErrMsgNotInInventory,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bus, repo, svc := setupInventoryEventTest(t)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			user := tt.setupUser(ctx, repo)

			received := make(chan event.Event, 1)
			bus.Subscribe(event.Type(domain.EventTypeItemRemoved), func(ctx context.Context, e event.Event) error {
				select {
				case received <- e:
				default:
				}
				return nil
			})

			_, err := svc.RemoveItemByUsername(ctx, domain.PlatformTwitch, user.Username, tt.itemName, tt.quantity)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, received, "Expected no event to be emitted")
				return
			}

			require.NoError(t, err)

			if tt.expectEvent {
				select {
				case e := <-received:
					assert.Equal(t, event.Type(domain.EventTypeItemRemoved), e.Type)
					payload, ok := e.Payload.(domain.ItemRemovedPayload)
					require.True(t, ok)
					assert.Equal(t, tt.itemName, payload.ItemName)

					expectedQty := tt.quantity
					if tt.expectedQuantity > 0 {
						expectedQty = tt.expectedQuantity
					}
					assert.Equal(t, expectedQty, payload.Quantity)
				case <-ctx.Done():
					t.Fatal("timeout waiting for event")
				}
			}
		})
	}
}

func TestInventoryEvents_ItemTransferred(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		itemName      string
		quantity      int
		setupUsers    func(context.Context, *FakeRepository) (*domain.User, *domain.User)
		expectEvent   bool
		expectedError string
	}{
		{
			name:     "Best Case - Valid transfer",
			itemName: domain.ItemStick,
			quantity: 2,
			setupUsers: func(ctx context.Context, repo *FakeRepository) (*domain.User, *domain.User) {
				owner := &domain.User{ID: "owner1", Username: "owner1", TwitchID: "owner_id1"}
				receiver := &domain.User{ID: "receiver1", Username: "receiver1", TwitchID: "receiver_id1"}
				repo.UpsertUser(ctx, owner)
				repo.UpsertUser(ctx, receiver)
				inv := &domain.Inventory{
					Slots: []domain.InventorySlot{
						{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityCommon},
					},
				}
				repo.UpdateInventory(ctx, owner.ID, *inv)
				repo.UpdateInventory(ctx, receiver.ID, domain.Inventory{Slots: []domain.InventorySlot{}})
				return owner, receiver
			},
			expectEvent: true,
		},
		{
			name:     "Boundary Case - Transfer exact inventory amount",
			itemName: domain.ItemStick,
			quantity: 10,
			setupUsers: func(ctx context.Context, repo *FakeRepository) (*domain.User, *domain.User) {
				owner := &domain.User{ID: "owner2", Username: "owner2", TwitchID: "owner_id2"}
				receiver := &domain.User{ID: "receiver2", Username: "receiver2", TwitchID: "receiver_id2"}
				repo.UpsertUser(ctx, owner)
				repo.UpsertUser(ctx, receiver)
				inv := &domain.Inventory{
					Slots: []domain.InventorySlot{
						{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityCommon},
					},
				}
				repo.UpdateInventory(ctx, owner.ID, *inv)
				repo.UpdateInventory(ctx, receiver.ID, domain.Inventory{Slots: []domain.InventorySlot{}})
				return owner, receiver
			},
			expectEvent: true,
		},
		{
			name:     "Boundary Case - Transfer insufficient amount",
			itemName: domain.ItemStick,
			quantity: 11,
			setupUsers: func(ctx context.Context, repo *FakeRepository) (*domain.User, *domain.User) {
				owner := &domain.User{ID: "owner3", Username: "owner3", TwitchID: "owner_id3"}
				receiver := &domain.User{ID: "receiver3", Username: "receiver3", TwitchID: "receiver_id3"}
				repo.UpsertUser(ctx, owner)
				repo.UpsertUser(ctx, receiver)
				inv := &domain.Inventory{
					Slots: []domain.InventorySlot{
						{ItemID: 1, Quantity: 10, QualityLevel: domain.QualityCommon},
					},
				}
				repo.UpdateInventory(ctx, owner.ID, *inv)
				repo.UpdateInventory(ctx, receiver.ID, domain.Inventory{Slots: []domain.InventorySlot{}})
				return owner, receiver
			},
			expectEvent:   false,
			expectedError: domain.ErrMsgInsufficientQuantity,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bus, repo, svc := setupInventoryEventTest(t)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			owner, receiver := tt.setupUsers(ctx, repo)

			received := make(chan event.Event, 1)
			bus.Subscribe(event.Type(domain.EventTypeItemTransferred), func(ctx context.Context, e event.Event) error {
				select {
				case received <- e:
				default:
				}
				return nil
			})

			err := svc.GiveItem(ctx, domain.PlatformTwitch, owner.TwitchID, owner.Username, domain.PlatformTwitch, receiver.Username, tt.itemName, tt.quantity)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, received, "Expected no event to be emitted")
				return
			}

			require.NoError(t, err)

			if tt.expectEvent {
				select {
				case e := <-received:
					assert.Equal(t, event.Type(domain.EventTypeItemTransferred), e.Type)
					payload, ok := e.Payload.(domain.ItemTransferredPayload)
					require.True(t, ok)
					assert.Equal(t, owner.ID, payload.FromUserID)
					assert.Equal(t, receiver.ID, payload.ToUserID)
					assert.Equal(t, tt.itemName, payload.ItemName)
					assert.Equal(t, tt.quantity, payload.Quantity)
				case <-ctx.Done():
					t.Fatal("timeout waiting for event")
				}
			}
		})
	}
}

func TestInventoryEvents_AddItemsBulk(t *testing.T) {
	t.Parallel()

	t.Run("Best Case - Multiple items", func(t *testing.T) {
		t.Parallel()
		bus, repo, svc := setupInventoryEventTest(t)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		user := &domain.User{ID: "bulk_user_id", Username: "bulk_user", TwitchID: "bulk_user_id"}
		repo.UpsertUser(ctx, user)

		items := map[string]int{
			domain.ItemStick: 2,
			domain.ItemMoney: 5,
		}

		received := make(chan event.Event, len(items))
		bus.Subscribe(event.Type(domain.EventTypeItemAdded), func(ctx context.Context, e event.Event) error {
			received <- e
			return nil
		})

		err := svc.AddItems(ctx, domain.PlatformTwitch, user.TwitchID, user.Username, items)
		require.NoError(t, err)

		count := 0
		for count < len(items) {
			select {
			case e := <-received:
				assert.Equal(t, event.Type(domain.EventTypeItemAdded), e.Type)
				count++
			case <-ctx.Done():
				t.Fatalf("timeout waiting for events, got %d expected %d", count, len(items))
			}
		}
	})

	t.Run("Edge Case - Empty map", func(t *testing.T) {
		t.Parallel()
		bus, repo, svc := setupInventoryEventTest(t)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		user := &domain.User{ID: "bulk_empty_user", Username: "bulk_empty", TwitchID: "bulk_empty_id"}
		repo.UpsertUser(ctx, user)

		items := map[string]int{}

		received := make(chan event.Event, 1)
		bus.Subscribe(event.Type(domain.EventTypeItemAdded), func(ctx context.Context, e event.Event) error {
			received <- e
			return nil
		})

		err := svc.AddItems(ctx, domain.PlatformTwitch, user.TwitchID, user.Username, items)
		require.NoError(t, err)

		// Wait briefly to ensure no event is fired
		select {
		case <-received:
			t.Fatal("Expected no event for empty map")
		case <-time.After(50 * time.Millisecond):
			// Success
		}
	})
}

func TestInventoryEvents_ItemUsed(t *testing.T) {
	t.Parallel()

	t.Run("Best Case - Valid item use", func(t *testing.T) {
		t.Parallel()
		bus, repo, svc := setupInventoryEventTest(t)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		user := &domain.User{ID: "user_used", Username: "user_used", TwitchID: "twitch_used"}
		repo.UpsertUser(ctx, user)
		inv := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 5, QualityLevel: domain.QualityCommon}, // stick
			},
		}
		repo.UpdateInventory(ctx, user.ID, *inv)

		received := make(chan event.Event, 1)
		bus.Subscribe(event.Type(domain.EventTypeItemUsed), func(ctx context.Context, e event.Event) error {
			received <- e
			return nil
		})

		_, err := svc.UseItem(ctx, domain.PlatformTwitch, user.TwitchID, user.Username, domain.ItemStick, 1, "")
		require.NoError(t, err)

		select {
		case e := <-received:
			assert.Equal(t, event.Type(domain.EventTypeItemUsed), e.Type)
			payload, ok := e.Payload.(domain.ItemUsedPayload)
			require.True(t, ok)
			assert.Equal(t, domain.ItemStick, payload.ItemName)
			assert.Equal(t, 1, payload.Quantity)
		case <-ctx.Done():
			t.Fatal("timeout waiting for event")
		}
	})
}
