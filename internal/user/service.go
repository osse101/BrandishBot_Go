package user

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Repository defines the interface for user persistence
type Repository interface {
	UpsertUser(ctx context.Context, user *domain.User) error
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
	GetItemByName(ctx context.Context, itemName string) (*domain.Item, error)
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
}

// Service defines the interface for user operations
type Service interface {
	RegisterUser(ctx context.Context, user domain.User) (domain.User, error)
	FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	HandleIncomingMessage(ctx context.Context, platform, platformID, username string) (domain.User, error)
	AddItem(ctx context.Context, username, platform, itemName string, quantity int) error
	RemoveItem(ctx context.Context, username, platform, itemName string, quantity int) (int, error)
}

// service implements the Service interface
type service struct {
	repo Repository
}

// NewService creates a new user service
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// RegisterUser registers a new user
func (s *service) RegisterUser(ctx context.Context, user domain.User) (domain.User, error) {
	if err := s.repo.UpsertUser(ctx, &user); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// FindUserByPlatformID finds a user by their platform-specific ID
func (s *service) FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return s.repo.GetUserByPlatformID(ctx, platform, platformID)
}

// HandleIncomingMessage checks if a user exists for an incoming message and creates one if not.
func (s *service) HandleIncomingMessage(ctx context.Context, platform, platformID, username string) (domain.User, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err == nil {
		return *user, nil
	}

	// TODO: Check if error is actually "not found"

	newUser := domain.User{
		Username: username,
	}

	switch platform {
	case "twitch":
		newUser.TwitchID = platformID
	case "youtube":
		newUser.YoutubeID = platformID
	case "discord":
		newUser.DiscordID = platformID
	default:
		return domain.User{}, fmt.Errorf("unsupported platform: %s", platform)
	}

	if _, err := s.RegisterUser(ctx, newUser); err != nil {
		return domain.User{}, err
	}

	return newUser, nil
}

func (s *service) AddItem(ctx context.Context, username, platform, itemName string, quantity int) error {
	// 1. Get User
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", username)
	}

	// 2. Get Item
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return fmt.Errorf("item not found: %s", itemName)
	}

	// 3. Get Inventory
	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// 4. Update Inventory
	found := false
	for i, slot := range inventory.Slots {
		if slot.ItemID == item.ID {
			inventory.Slots[i].Quantity += quantity
			found = true
			break
		}
	}

	if !found {
		inventory.Slots = append(inventory.Slots, domain.InventorySlot{
			ItemID:   item.ID,
			Quantity: quantity,
		})
	}

	// 5. Save Inventory
	if err := s.repo.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	return nil
}

func (s *service) RemoveItem(ctx context.Context, username, platform, itemName string, quantity int) (int, error) {
	// 1. Get User
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return 0, fmt.Errorf("user not found: %s", username)
	}

	// 2. Get Item
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return 0, fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return 0, fmt.Errorf("item not found: %s", itemName)
	}

	// 3. Get Inventory
	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to get inventory: %w", err)
	}

	// 4. Find and remove from inventory
	var removed int
	found := false
	for i, slot := range inventory.Slots {
		if slot.ItemID == item.ID {
			found = true
			// Calculate how many we can actually remove
			removed = quantity
			if slot.Quantity < quantity {
				removed = slot.Quantity
			}
			
			// Update or remove the slot
			if slot.Quantity <= quantity {
				// Remove the slot entirely
				inventory.Slots = append(inventory.Slots[:i], inventory.Slots[i+1:]...)
			} else {
				// Just decrease the quantity
				inventory.Slots[i].Quantity -= removed
			}
			break
		}
	}

	if !found {
		return 0, fmt.Errorf("item %s not in inventory", itemName)
	}

	// 5. Save Inventory
	if err := s.repo.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return 0, fmt.Errorf("failed to update inventory: %w", err)
	}

	return removed, nil
}

