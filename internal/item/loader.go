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
	"github.com/osse101/BrandishBot_Go/internal/validation"
)

// Sentinel errors for item loader
var (
	ErrDuplicateInternalName = errors.New("duplicate internal name")

	ErrInvalidConfig = errors.New("invalid configuration")
)

// Schema paths
const (
	ItemsSchemaPath = "configs/schemas/items.schema.json"
)

// Config represents the JSON configuration for items
type Config struct {
	Version     string `json:"version"`
	Description string `json:"description"`

	Items []Def `json:"items"`
}

// Def represents a single item definition in the JSON
type Def struct {
	InternalName   string   `json:"internal_name"`
	PublicName     string   `json:"public_name"`
	Description    string   `json:"description"`
	Tier           int      `json:"tier,omitempty"`
	MaxStack       int      `json:"max_stack"`
	BaseValue      int      `json:"base_value"`
	Tags           []string `json:"tags"`
	Type           []string `json:"type"` // Content type categorization
	Handler        *string  `json:"handler,omitempty"`
	DefaultDisplay string   `json:"default_display"`
}

// Loader handles loading and validating item configuration
type Loader interface {
	Load(path string) (*Config, error)
	Validate(config *Config) error
	SyncToDatabase(ctx context.Context, config *Config, repo repository.Item, configPath string) (*SyncResult, error)
}

// SyncResult contains the result of syncing items to the database
type SyncResult struct {
	ItemsInserted int
	ItemsUpdated  int
	ItemsSkipped  int
}

type itemLoader struct {
	schemaValidator validation.SchemaValidator
}

// NewLoader creates a new Loader instance
func NewLoader() Loader {
	return &itemLoader{
		schemaValidator: validation.NewSchemaValidator(),
	}
}

// Load reads and parses an items JSON file
func (l *itemLoader) Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(ErrMsgReadConfigFileFailed, err)
	}

	// Validate against schema first
	if err := l.schemaValidator.ValidateBytes(data, ItemsSchemaPath); err != nil {
		return nil, fmt.Errorf("schema validation failed for %s: %w", path, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf(ErrMsgParseConfigFailed, err)
	}

	return &config, nil
}

// Validate checks the item configuration for errors
func (l *itemLoader) Validate(config *Config) error {
	if config == nil {
		return fmt.Errorf("%w: %s", ErrInvalidConfig, ErrMsgConfigNil)
	}

	if len(config.Items) == 0 {
		return fmt.Errorf("%w: %s", ErrInvalidConfig, ErrMsgNoItemsDefined)
	}

	// Track internal names for duplicate detection
	internalNames := make(map[string]bool, len(config.Items))

	// Validate each item
	for i := range config.Items {
		item := &config.Items[i]

		if err := l.validateItemDef(i, item, internalNames); err != nil {
			return err
		}
	}

	return nil
}

func (l *itemLoader) validateItemDef(index int, item *Def, internalNames map[string]bool) error {
	// Check for empty internal name
	if item.InternalName == "" {
		return fmt.Errorf(ErrFmtItemAtIndexEmpty, ErrInvalidConfig, index)
	}

	// Check for duplicate internal names
	if internalNames[item.InternalName] {
		return fmt.Errorf("%w: '%s'", ErrDuplicateInternalName, item.InternalName)
	}
	internalNames[item.InternalName] = true

	// Validate required fields
	if item.PublicName == "" {
		return fmt.Errorf(ErrFmtItemHasEmptyPublic, ErrInvalidConfig, item.InternalName)
	}
	if item.DefaultDisplay == "" {
		return fmt.Errorf(ErrFmtItemHasEmptyDisplay, ErrInvalidConfig, item.InternalName)
	}

	// Validate numeric fields
	if item.MaxStack < 0 {
		return fmt.Errorf(ErrFmtItemNegativeMaxStack, ErrInvalidConfig, item.InternalName)
	}
	if item.BaseValue < 0 {
		return fmt.Errorf(ErrFmtItemNegativeValue, ErrInvalidConfig, item.InternalName)
	}

	return nil
}

// SyncToDatabase syncs the item configuration to the database idempotently
func (l *itemLoader) SyncToDatabase(ctx context.Context, config *Config, repo repository.Item, configPath string) (*SyncResult, error) {
	log := logger.FromContext(ctx)

	// Check if file has changed since last sync
	hasChanged, err := hasFileChanged(ctx, repo, configPath)
	if err != nil {
		return nil, fmt.Errorf(ErrMsgCheckFileChangeFailed, err)
	}

	if !hasChanged {
		log.Info(LogMsgConfigUnchanged, "path", configPath)
		return &SyncResult{}, nil
	}

	existingByInternalName, typesByName, err := l.loadSyncData(ctx, repo)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{}
	// Process each item
	for _, itemDef := range config.Items {
		if err := l.syncOneItem(ctx, repo, itemDef, existingByInternalName, typesByName, result); err != nil {
			return nil, err
		}
	}

	// Update sync metadata
	if err := updateSyncMetadata(ctx, repo, configPath); err != nil {
		log.Warn(LogMsgUpdateMetadataFailed, "error", err)
	}

	log.Info(LogMsgSyncCompleted,
		"inserted", result.ItemsInserted,
		"updated", result.ItemsUpdated,
		"skipped", result.ItemsSkipped)

	return result, nil
}

func (l *itemLoader) loadSyncData(ctx context.Context, repo repository.Item) (map[string]*domain.Item, map[string]int, error) {
	// Get all existing items from DB
	existingItems, err := repo.GetAllItems(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf(ErrMsgGetExistingItemsFailed, err)
	}

	existingByInternalName := make(map[string]*domain.Item, len(existingItems))
	for i := range existingItems {
		existingByInternalName[existingItems[i].InternalName] = &existingItems[i]
	}

	// Get all item types for tag sync
	itemTypes, err := repo.GetAllItemTypes(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf(ErrMsgGetItemTypesFailed, err)
	}

	typesByName := make(map[string]int, len(itemTypes))
	for _, itemType := range itemTypes {
		typesByName[itemType.Name] = itemType.ID
	}

	return existingByInternalName, typesByName, nil
}

func (l *itemLoader) syncOneItem(ctx context.Context, repo repository.Item, itemDef Def, existingByInternalName map[string]*domain.Item, typesByName map[string]int, result *SyncResult) error {
	log := logger.FromContext(ctx)

	if existing, ok := existingByInternalName[itemDef.InternalName]; ok {
		// Item exists - check if update needed
		needsUpdate := existing.PublicName != itemDef.PublicName ||
			existing.Description != itemDef.Description ||
			existing.BaseValue != itemDef.BaseValue ||
			existing.DefaultDisplay != itemDef.DefaultDisplay ||
			!stringSlicesEqual(existing.ContentType, itemDef.Type) ||
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
				ContentType:    itemDef.Type,
			}); err != nil {
				return fmt.Errorf(ErrMsgUpdateItemFailed, itemDef.InternalName, err)
			}
			result.ItemsUpdated++
			log.Info(LogMsgUpdatedItem, "internal_name", itemDef.InternalName)
		} else {
			result.ItemsSkipped++
		}

		// Sync tags for this item
		if err := syncItemTags(ctx, repo, existing.ID, itemDef.Tags, typesByName); err != nil {
			return fmt.Errorf(ErrMsgSyncTagsFailed, itemDef.InternalName, err)
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
			ContentType:    itemDef.Type,
		}

		itemID, err := repo.InsertItem(ctx, newItem)
		if err != nil {
			return fmt.Errorf(ErrMsgInsertItemFailed, itemDef.InternalName, err)
		}

		result.ItemsInserted++
		log.Info(LogMsgInsertedItem, "internal_name", itemDef.InternalName, "id", itemID)

		// Sync tags for new item
		if err := syncItemTags(ctx, repo, itemID, itemDef.Tags, typesByName); err != nil {
			return fmt.Errorf(ErrMsgSyncTagsNewItemFailed, itemDef.InternalName, err)
		}
	}
	return nil
}

// hasFileChanged checks if the config file has changed since last sync
func hasFileChanged(ctx context.Context, repo repository.Item, configPath string) (bool, error) {
	// Get file info
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return false, fmt.Errorf(ErrMsgStatConfigFileFailed, err)
	}

	// Calculate file hash
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, fmt.Errorf(ErrMsgReadForHashFailed, err)
	}

	hash := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hash[:])

	// Get last sync metadata
	syncMeta, err := repo.GetSyncMetadata(ctx, ConfigFileName)
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
		return fmt.Errorf(ErrMsgStatConfigFileFailed, err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf(ErrMsgReadForHashFailed, err)
	}

	hash := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hash[:])

	return repo.UpsertSyncMetadata(ctx, &domain.SyncMetadata{
		ConfigName:   ConfigFileName,
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
				return fmt.Errorf(ErrMsgCreateItemTypeFailed, tag, err)
			}
			typesByName[tag] = newTypeID
			typeIDs = append(typeIDs, newTypeID)
		} else {
			typeIDs = append(typeIDs, typeID)
		}
	}

	// Clear existing tags and insert new ones
	if err := repo.ClearItemTags(ctx, itemID); err != nil {
		return fmt.Errorf(ErrMsgClearTagsFailed, err)
	}

	for _, typeID := range typeIDs {
		if err := repo.AssignItemTag(ctx, itemID, typeID); err != nil {
			return fmt.Errorf(ErrMsgAssignTagFailed, err)
		}
	}

	return nil
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
