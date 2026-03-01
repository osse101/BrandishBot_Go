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

func TestInventoryEvents(t *testing.T) {
	// Setup
	bus := event.NewMemoryBus()
	tmpFile, err := os.CreateTemp("", "deadletter")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	publisher, err := event.NewResilientPublisher(bus, 1, time.Millisecond, tmpFile.Name())
	require.NoError(t, err)
	defer publisher.Shutdown(context.Background())

	repo := NewFakeRepository()
	// Populate items
	stickItem := &domain.Item{
		ID:           1,
		InternalName: domain.ItemStick,
		PublicName:   domain.PublicNameStick,
		BaseValue:    10,
	}
	repo.items[domain.ItemStick] = stickItem

	svc := NewService(repo, repo, nil, publisher, nil, NewMockNamingResolver(), nil, nil, nil, false).(*service)
	ctx := context.Background()

	t.Run("ItemAdded Event", func(t *testing.T) {
		// Setup user
		user := &domain.User{ID: "user_added", Username: "user_added", TwitchID: "twitch_added"}
		repo.UpsertUser(ctx, user)

		received := make(chan event.Event, 1)
		bus.Subscribe(event.Type(domain.EventTypeItemAdded), func(ctx context.Context, e event.Event) error {
			received <- e
			return nil
		})

		err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, "user_added", domain.ItemStick, 5)
		require.NoError(t, err)

		select {
		case e := <-received:
			assert.Equal(t, event.Type(domain.EventTypeItemAdded), e.Type)
			payload, ok := e.Payload.(domain.ItemAddedPayload)
			assert.True(t, ok)
			assert.Equal(t, domain.ItemStick, payload.ItemName)
			assert.Equal(t, 5, payload.Quantity)
		case <-time.After(2 * time.Second): // Wait longer for async worker
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("ItemRemoved Event", func(t *testing.T) {
		// Setup user with items
		user := &domain.User{ID: "user_removed", Username: "user_removed", TwitchID: "twitch_removed"}
		repo.UpsertUser(ctx, user)
		inv := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: stickItem.ID, Quantity: 10, QualityLevel: domain.QualityCommon},
			},
		}
		repo.UpdateInventory(ctx, user.ID, *inv)

		received := make(chan event.Event, 1)
		bus.Subscribe(event.Type(domain.EventTypeItemRemoved), func(ctx context.Context, e event.Event) error {
			received <- e
			return nil
		})

		_, err := svc.RemoveItemByUsername(ctx, domain.PlatformTwitch, "user_removed", domain.ItemStick, 3)
		require.NoError(t, err)

		select {
		case e := <-received:
			assert.Equal(t, event.Type(domain.EventTypeItemRemoved), e.Type)
			payload, ok := e.Payload.(domain.ItemRemovedPayload)
			assert.True(t, ok)
			assert.Equal(t, domain.ItemStick, payload.ItemName)
			assert.Equal(t, 3, payload.Quantity)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("ItemTransferred Event", func(t *testing.T) {
		// Setup users
		owner := &domain.User{ID: "owner", Username: "owner", TwitchID: "owner_id"}
		receiver := &domain.User{ID: "receiver", Username: "receiver", TwitchID: "receiver_id"}
		repo.UpsertUser(ctx, owner)
		repo.UpsertUser(ctx, receiver)

		inv := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: stickItem.ID, Quantity: 10, QualityLevel: domain.QualityCommon},
			},
		}
		repo.UpdateInventory(ctx, owner.ID, *inv)
		repo.UpdateInventory(ctx, receiver.ID, domain.Inventory{Slots: []domain.InventorySlot{}})

		received := make(chan event.Event, 1)
		bus.Subscribe(event.Type(domain.EventTypeItemTransferred), func(ctx context.Context, e event.Event) error {
			received <- e
			return nil
		})

		err := svc.GiveItem(ctx, domain.PlatformTwitch, "owner_id", "owner", domain.PlatformTwitch, "receiver", domain.ItemStick, 2)
		require.NoError(t, err)

		select {
		case e := <-received:
			assert.Equal(t, event.Type(domain.EventTypeItemTransferred), e.Type)
			payload, ok := e.Payload.(domain.ItemTransferredPayload)
			assert.True(t, ok)
			assert.Equal(t, owner.ID, payload.FromUserID)
			assert.Equal(t, receiver.ID, payload.ToUserID)
			assert.Equal(t, domain.ItemStick, payload.ItemName)
			assert.Equal(t, 2, payload.Quantity)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("AddItems (Bulk) Event", func(t *testing.T) {
		received := make(chan event.Event, 2)
		bus.Subscribe(event.Type(domain.EventTypeItemAdded), func(ctx context.Context, e event.Event) error {
			received <- e
			return nil
		})

		items := map[string]int{
			domain.ItemStick: 2,
		}

		err := svc.AddItems(ctx, domain.PlatformTwitch, "bulk_user_id", "bulk_user", items)
		require.NoError(t, err)

		timeout := time.After(2 * time.Second)
		count := 0
		for count < 1 {
			select {
			case e := <-received:
				assert.Equal(t, event.Type(domain.EventTypeItemAdded), e.Type)
				count++
			case <-timeout:
				t.Fatal("timeout waiting for events")
			}
		}
	})

	t.Run("ItemUsed Event", func(t *testing.T) {
		// Setup user with item
		user := &domain.User{ID: "user_used", Username: "user_used", TwitchID: "twitch_used"}
		repo.UpsertUser(ctx, user)

		inv := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: stickItem.ID, Quantity: 5, QualityLevel: domain.QualityCommon},
			},
		}
		repo.UpdateInventory(ctx, user.ID, *inv)

		received := make(chan event.Event, 1)
		bus.Subscribe(event.Type(domain.EventTypeItemUsed), func(ctx context.Context, e event.Event) error {
			received <- e
			return nil
		})

		_, err := svc.UseItem(ctx, domain.PlatformTwitch, "twitch_used", "user_used", domain.ItemStick, 1, "")
		require.NoError(t, err)

		select {
		case e := <-received:
			assert.Equal(t, event.Type(domain.EventTypeItemUsed), e.Type)
			payload, ok := e.Payload.(domain.ItemUsedPayload)
			assert.True(t, ok)
			assert.Equal(t, domain.ItemStick, payload.ItemName)
			assert.Equal(t, 1, payload.Quantity)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	})
}
