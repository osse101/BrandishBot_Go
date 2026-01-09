package item

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Sentinel errors for item loader
var (
	ErrDuplicateInternalName = errors.New("duplicate internal name")
	ErrInvalidTag            = errors.New("invalid tag")
	ErrInvalidHandler        = errors.New("invalid handler")
	ErrInvalidConfig         = errors.New("invalid configuration")
)

// ItemConfig represents the JSON configuration for items
type ItemConfig struct {
	Version       string   `json:"version"`
	Description   string   `json:"description"`
	ValidTags     []string `json:"valid_tags"`
	ValidHandlers []string `json:"valid_handlers"`
	Items         []ItemDef `json:"items"`
}

// ItemDef represents a single item definition in the JSON
type ItemDef struct {
	InternalName   string   `json:"internal_name"`
	PublicName     string   `json:"public_name"`
	Description    string   `json:"description"`
	Tier           int      `json:"tier"`
	MaxStack       int      `json:"max_stack"`
	BaseValue      int      `json:"base_value"`
	Tags           []string `json:"tags"`
	Handler        *string  `json:"handler,omitempty"`
	DefaultDisplay string   `json:"default_display"`
}

// ItemLoader handles loading and validating item configuration
type ItemLoader interface {
	Load(path string) (*ItemConfig, error)
	Validate(config *ItemConfig) error
	SyncToDatabase(ctx context.Context, config *ItemConfig, repo repository.Item, configPath string) (*SyncResult, error)
}

// SyncResult contains the result of syncing items to the database
type SyncResult struct {
	ItemsInserted int
	ItemsUpdated  int
	ItemsSkipped  int
}

type itemLoader struct{}

// NewLoader creates a new ItemLoader instance
func NewLoader() ItemLoader {
	return &itemLoader{}
}

// Load reads and parses an items JSON file
func (l *itemLoader) Load(path string) (*ItemConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read items config file: %w", err)
	}

	var config ItemConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse items config: %w", err)
	}

	return &config, nil
}

// Validate checks the item configuration for errors
func (l *itemLoader) Validate(config *ItemConfig) error {
	if config == nil {
		return fmt.Errorf("%w: config is nil", ErrInvalidConfig)
	}

	if len(config.Items) == 0 {
		return fmt.Errorf("%w: no items defined", ErrInvalidConfig)
	}

	// Build validation sets
	validTags := make(map[string]bool, len(config.ValidTags))
	for _, tag := range config.ValidTags {
		validTags[tag] = true
	}

	validHandlers := make(map[string]bool, len(config.ValidHandlers))
	for _, handler := range config.ValidHandlers {
		validHandlers[handler] = true
	}

	// Track internal names for duplicate detection
	internalNames := make(map[string]bool, len(config.Items))

	// Validate each item
	for i := range config.Items {
		item := &config.Items[i]

		// Check for empty internal name
		if item.InternalName == "" {
			return fmt.Errorf("%w: item at index %d has empty internal_name", ErrInvalidConfig, i)
		}

		// Check for duplicate internal names
		if internalNames[item.InternalName] {
			return fmt.Errorf("%w: '%s'", ErrDuplicateInternalName, item.InternalName)
		}
		internalNames[item.InternalName] = true

		// Validate required fields
		if item.PublicName == "" {
			return fmt.Errorf("%w: item '%s' has empty public_name", ErrInvalidConfig, item.InternalName)
		}
		if item.DefaultDisplay == "" {
			return fmt.Errorf("%w: item '%s' has empty default_display", ErrInvalidConfig, item.InternalName)
		}

		// Validate tags
		for _, tag := range item.Tags {
			if !validTags[tag] {
				return fmt.Errorf("%w: item '%s' has invalid tag '%s' (not in valid_tags)", ErrInvalidTag, item.InternalName, tag)
			}
		}

		// Validate handler (if present)
		if item.Handler != nil && *item.Handler != "" {
			if !validHandlers[*item.Handler] {
				return fmt.Errorf("%w: item '%s' has invalid handler '%s' (not in valid_handlers)", ErrInvalidHandler, item.InternalName, *item.Handler)
			}
		}

		// Validate numeric fields
		if item.MaxStack < 0 {
			return fmt.Errorf("%w: item '%s' has negative max_stack", ErrInvalidConfig, item.InternalName)
		}
		if item.BaseValue < 0 {
			return fmt.Errorf("%w: item '%s' has negative base_value", ErrInvalidConfig, item.InternalName)
		}
	}

	return nil
}

// SyncToDatabase syncs the item configuration to the database idempotently
func (l *itemLoader) SyncToDatabase(ctx context.Context, config *ItemConfig, repo repository.Item, configPath string) (*SyncResult, error) {
	log := logger.FromContext(ctx)

	// Check if file has changed since last sync
	hasChanged, err := hasFileChanged(ctx, repo, configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check if file changed: %w", err)
	}

	if !hasChanged {
		log.Info("Items config file unchanged, skipping sync", "path", configPath)
		return &SyncResult{}, nil
	}

	result := &SyncResult{}

	// Get all existing items from DB
	existingItems, err := repo.GetAllItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing items: %w", err)
	}

	existingByInternalName := make(map[string]*domain.Item, len(existingItems))
	for i := range existingItems {
		existingByInternalName[existingItems[i].InternalName] = &existingItems[i]
	}

	// Get all item types for tag sync
	itemTypes, err := repo.GetAllItemTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get item types: %w", err)
	}

	typesByName := make(map[string]int, len(itemTypes))
	for _, itemType := range itemTypes {
		typesByName[itemType.Name] = itemType.ID
	}

	// Process each item
	for _, itemDef := range config.Items {
		if existing, ok := existingByInternalName[itemDef.InternalName]; ok {
			// Item exists - check if update needed
			needsUpdate := existing.PublicName != itemDef.PublicName ||
				existing.Description != itemDef.Description ||
				existing.BaseValue != itemDef.BaseValue ||
				existing.DefaultDisplay != itemDef.DefaultDisplay ||
				(itemDef.Handler != nil && (existing.Handler == nil || *existing.Handler != *itemDef.Handler))

			if needsUpdate {
				// Update existing item
				if err := repo.UpdateItem(ctx, existing.ID, &domain.Item{
					InternalName:   itemDef.InternalName,
					PublicName:     itemDef.PublicName,
					Description:    itemDef.Description,
					BaseValue:      itemDef.BaseValue,
					Handler:        itemDef.Handler,
					DefaultDisplay: itemDef.DefaultDisplay,
				}); err != nil {
					return nil, fmt.Errorf("failed to update item '%s': %w", itemDef.InternalName, err)
				}
				result.ItemsUpdated++
				log.Info("Updated item", "internal_name", itemDef.InternalName)
			} else {
				result.ItemsSkipped++
			}

			// Sync tags for this item
			if err := syncItemTags(ctx, repo, existing.ID, itemDef.Tags, typesByName); err != nil {
				return nil, fmt.Errorf("failed to sync tags for '%s': %w", itemDef.InternalName, err)
			}
		} else {
			// Insert new item
			newItem := &domain.Item{
				InternalName:   itemDef.InternalName,
				PublicName:     itemDef.PublicName,
				Description:    itemDef.Description,
				BaseValue:      itemDef.BaseValue,
				Handler:        itemDef.Handler,
				DefaultDisplay: itemDef.DefaultDisplay,
			}

			itemID, err := repo.InsertItem(ctx, newItem)
			if err != nil {
				return nil, fmt.Errorf("failed to insert item '%s': %w", itemDef.InternalName, err)
			}

			result.ItemsInserted++
			log.Info("Inserted item", "internal_name", itemDef.InternalName, "id", itemID)

			// Sync tags for new item
			if err := syncItemTags(ctx, repo, itemID, itemDef.Tags, typesByName); err != nil {
				return nil, fmt.Errorf("failed to sync tags for new item '%s': %w", itemDef.InternalName, err)
			}
		}
	}

	// Update sync metadata
	if err := updateSyncMetadata(ctx, repo, configPath); err != nil {
		log.Warn("Failed to update sync metadata", "error", err)
	}

	log.Info("Items sync completed",
		"inserted", result.ItemsInserted,
		"updated", result.ItemsUpdated,
		"skipped", result.ItemsSkipped)

	return result, nil
}

// hasFileChanged checks if the config file has changed since last sync
func hasFileChanged(ctx context.Context, repo repository.Item, configPath string) (bool, error) {
	// Get file info
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return false, fmt.Errorf("failed to stat config file: %w", err)
	}

	// Calculate file hash
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, fmt.Errorf("failed to read config file: %w", err)
	}

	hash := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hash[:])

	// Get last sync metadata
	syncMeta, err := repo.GetSyncMetadata(ctx, "items.json")
	if err != nil {
		// First sync - no metadata exists
		return true, nil
	}

	// Compare hash and mod time
	if syncMeta.FileHash != fileHash || !syncMeta.FileModTime.Equal(fileInfo.ModTime()) {
		return true, nil
	}

	return false, nil
}

// updateSyncMetadata updates the sync metadata after a successful sync
func updateSyncMetadata(ctx context.Context, repo repository.Item, configPath string) error {
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	hash := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hash[:])

	return repo.UpsertSyncMetadata(ctx, &domain.SyncMetadata{
		ConfigName:   "items.json",
		LastSyncTime: time.Now(),
		FileHash:     fileHash,
		FileModTime:  fileInfo.ModTime(),
	})
}

// syncItemTags syncs the tags (item types) for an item
func syncItemTags(ctx context.Context, repo repository.Item, itemID int, tags []string, typesByName map[string]int) error {
	// Get type IDs for the tags
	typeIDs := make([]int, 0, len(tags))
	for _, tag := range tags {
		typeID, ok := typesByName[tag]
		if !ok {
			// Tag doesn't exist in DB - we could either error or create it
			// For now, create it automatically
			newTypeID, err := repo.InsertItemType(ctx, tag)
			if err != nil {
				return fmt.Errorf("failed to create item type '%s': %w", tag, err)
			}
			typesByName[tag] = newTypeID
			typeIDs = append(typeIDs, newTypeID)
		} else {
			typeIDs = append(typeIDs, typeID)
		}
	}

	// Clear existing tags and insert new ones
	if err := repo.ClearItemTags(ctx, itemID); err != nil {
		return fmt.Errorf("failed to clear existing tags: %w", err)
	}

	for _, typeID := range typeIDs {
		if err := repo.AssignItemTag(ctx, itemID, typeID); err != nil {
			return fmt.Errorf("failed to assign tag: %w", err)
		}
	}

	return nil
}
