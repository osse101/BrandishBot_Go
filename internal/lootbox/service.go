package lootbox

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
	"github.com/osse101/BrandishBot_Go/internal/validation"
)

// ============================================================================
// Config types — JSON → Go (v2 format)
// ============================================================================

// PoolItemDef is one entry in a pool. Exactly one of ItemName or ItemType must be set.
type PoolItemDef struct {
	ItemName string `json:"item_name,omitempty"`
	ItemType string `json:"item_type,omitempty"`
	Weight   int    `json:"weight"`
}

// PoolDef holds the items that make up a named pool.
type PoolDef struct {
	Items []PoolItemDef `json:"items"`
}

// PoolRef links a pool into a lootbox with a relative selection weight.
type PoolRef struct {
	PoolName string `json:"pool_name"`
	Weight   int    `json:"weight"`
}

// MoneyRange defines the consolation money range (inclusive).
type MoneyRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// Def defines one lootbox type in the config.
type Def struct {
	ItemDropRate float64    `json:"item_drop_rate"` // gatekeeper probability [0,1]
	FixedMoney   MoneyRange `json:"fixed_money"`
	Pools        []PoolRef  `json:"pools"`
}

// LootTableConfig is the top-level v2 config structure.
type LootTableConfig struct {
	Version   string             `json:"version"`
	Pools     map[string]PoolDef `json:"pools"`
	Lootboxes map[string]Def     `json:"lootboxes"`
}

// ============================================================================
// Public domain types
// ============================================================================

// DroppedItem represents an item generated from opening a lootbox.
type DroppedItem struct {
	ItemID       int
	ItemName     string
	Quantity     int
	Value        int
	QualityLevel domain.QualityLevel
}

// ============================================================================
// Interfaces
// ============================================================================

// ItemRepository defines the interface for fetching item data.
type ItemRepository interface {
	GetItemByName(ctx context.Context, name string) (*domain.Item, error)
	GetItemsByNames(ctx context.Context, names []string) ([]domain.Item, error)
	GetAllItems(ctx context.Context) ([]domain.Item, error)
}

// Service defines the lootbox opening interface.
type Service interface {
	OpenLootbox(ctx context.Context, lootboxName string, quantity int, boxQuality domain.QualityLevel) ([]DroppedItem, error)
}

// ProgressionService defines the interface for checking feature unlocks.
type ProgressionService interface {
	IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error)
}

// ============================================================================
// Service implementation
// ============================================================================

// Option defines a functional option for the lootbox service.
type Option func(*service)

// WithRnd sets a custom random number generator function.
func WithRnd(rnd func() float64) Option {
	return func(s *service) {
		s.rnd = rnd
	}
}

type service struct {
	repo            ItemRepository
	progressionSvc  ProgressionService
	cache           map[string]*FlattenedLootbox // read-only after NewService
	rnd             func() float64
	schemaValidator validation.SchemaValidator
}

// NewService creates a new lootbox service and builds the item drop cache.
// signature is unchanged — context.Background() is used internally for GetAllItems.
func NewService(repo ItemRepository, progressionSvc ProgressionService, lootTablesPath string, opts ...Option) (Service, error) {
	svc := &service{
		repo:            repo,
		progressionSvc:  progressionSvc,
		cache:           make(map[string]*FlattenedLootbox),
		rnd:             utils.RandomFloat,
		schemaValidator: validation.NewSchemaValidator(),
	}

	for _, opt := range opts {
		opt(svc)
	}

	if err := svc.buildCache(lootTablesPath); err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToLoadLootTables, err)
	}

	return svc, nil
}

// OpenLootbox simulates opening lootboxes and returns the dropped items.
func (s *service) OpenLootbox(ctx context.Context, lootboxName string, quantity int, boxQuality domain.QualityLevel) ([]DroppedItem, error) {
	if quantity <= 0 {
		return nil, nil
	}

	flat, ok := s.cache[lootboxName]
	if !ok {
		logger.FromContext(ctx).Warn(LogMsgNoLootTableFound, LogFieldLootbox, lootboxName)
		return nil, nil
	}

	dropCounts, consolationMoney := s.processLootTable(flat, quantity)
	if len(dropCounts) == 0 && consolationMoney == 0 {
		return nil, nil
	}

	return s.convertToDroppedItems(ctx, dropCounts, consolationMoney, flat.MoneyItem, boxQuality)
}
