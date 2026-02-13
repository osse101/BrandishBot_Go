package lootbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
	"github.com/osse101/BrandishBot_Go/internal/validation"
)

// LootItem defines an item that can be dropped from a lootbox
type LootItem struct {
	ItemName string  `json:"item_name"`
	Min      int     `json:"min"`
	Max      int     `json:"max"`
	Chance   float64 `json:"chance"`
}

// Schema paths
const (
	LootTablesSchemaPath = "configs/schemas/loot_tables.schema.json"
)

// DroppedItem represents an item generated from opening a lootbox
type DroppedItem struct {
	ItemID       int
	ItemName     string
	Quantity     int
	Value        int
	QualityLevel domain.QualityLevel
}

// ItemRepository defines the interface for fetching item data
type ItemRepository interface {
	GetItemByName(ctx context.Context, name string) (*domain.Item, error)
	GetItemsByNames(ctx context.Context, names []string) ([]domain.Item, error)
}

// Service defines the lootbox opening interface
type Service interface {
	OpenLootbox(ctx context.Context, lootboxName string, quantity int, boxQuality domain.QualityLevel) ([]DroppedItem, error)
}

// ProgressionService defines the interface for checking feature unlocks
type ProgressionService interface {
	IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error)
}

// Option defines a functional option for the lootbox service
type Option func(*service)

// WithRnd sets a custom random number generator function
func WithRnd(rnd func() float64) Option {
	return func(s *service) {
		s.rnd = rnd
	}
}

type service struct {
	repo            ItemRepository
	progressionSvc  ProgressionService
	lootTables      map[string][]LootItem
	rnd             func() float64
	schemaValidator validation.SchemaValidator
}

// NewService creates a new lootbox service
func NewService(repo ItemRepository, progressionSvc ProgressionService, lootTablesPath string, opts ...Option) (Service, error) {
	svc := &service{
		repo:            repo,
		progressionSvc:  progressionSvc,
		lootTables:      make(map[string][]LootItem),
		rnd:             utils.RandomFloat,
		schemaValidator: validation.NewSchemaValidator(),
	}

	for _, opt := range opts {
		opt(svc)
	}

	// Load loot tables from JSON file
	if err := svc.loadLootTables(lootTablesPath); err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToLoadLootTables, err)
	}

	return svc, nil
}

func (s *service) loadLootTables(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToReadLootFile, err)
	}

	// Validate against schema first
	if err := s.schemaValidator.ValidateBytes(data, LootTablesSchemaPath); err != nil {
		return fmt.Errorf("schema validation failed for %s: %w", path, err)
	}

	// Parse the nested structure with "tables" key
	var config struct {
		Tables map[string][]LootItem `json:"tables"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToParseLootFile, err)
	}

	// Additional validation for table structure
	if len(config.Tables) == 0 {
		return fmt.Errorf("no loot tables defined in configuration")
	}

	s.lootTables = config.Tables
	return nil
}

// OpenLootbox simulates opening lootboxes and returns the dropped items
func (s *service) OpenLootbox(ctx context.Context, lootboxName string, quantity int, boxQuality domain.QualityLevel) ([]DroppedItem, error) {
	if quantity <= 0 {
		return nil, nil
	}

	table, ok := s.lootTables[lootboxName]
	if !ok {
		logger.FromContext(ctx).Warn(LogMsgNoLootTableFound, LogFieldLootbox, lootboxName)
		return nil, nil
	}

	dropCounts := s.processLootTable(table, quantity)
	if len(dropCounts) == 0 {
		return nil, nil
	}

	return s.convertToDroppedItems(ctx, dropCounts, boxQuality)
}
