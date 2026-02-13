package lootbox

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
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
	bus             event.Bus
	lootTablesPath  string // Stored for cache rebuilding
}

// NewService creates a new lootbox service and builds the item drop cache.
// signature is unchanged — context.Background() is used internally for GetAllItems.
func NewService(repo ItemRepository, progressionSvc ProgressionService, bus event.Bus, lootTablesPath string, opts ...Option) (Service, error) {
	svc := &service{
		repo:            repo,
		progressionSvc:  progressionSvc,
		cache:           make(map[string]*FlattenedLootbox),
		rnd:             utils.RandomFloat,
		schemaValidator: validation.NewSchemaValidator(),
		bus:             bus,
		lootTablesPath:  lootTablesPath,
	}

	for _, opt := range opts {
		opt(svc)
	}

	if err := svc.buildCache(lootTablesPath); err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToLoadLootTables, err)
	}

	// Subscribe to progression node unlocked events for cache invalidation
	if bus != nil {
		bus.Subscribe(event.ProgressionNodeUnlocked, svc.handleNodeUnlocked)
	}

	return svc, nil
}

// handleNodeUnlocked rebuilds the lootbox cache when an item node is unlocked.
// Only rebuilds if the unlocked node is an item node (starts with "item_").
func (s *service) handleNodeUnlocked(ctx context.Context, e event.Event) error {
	log := logger.FromContext(ctx)

	payload, ok := e.Payload.(map[string]interface{})
	if !ok {
		log.Warn("Invalid payload for ProgressionNodeUnlocked event")
		return nil
	}

	nodeKey, ok := payload["node_key"].(string)
	if !ok {
		log.Warn("Missing or invalid node_key in ProgressionNodeUnlocked event")
		return nil
	}

	// Only rebuild cache if an item node was unlocked
	// Item nodes follow the pattern "item_{internal_name}"
	if len(nodeKey) < 5 || nodeKey[:5] != "item_" {
		log.Debug("Ignoring non-item node unlock", "node_key", nodeKey)
		return nil
	}

	log.Info("Item node unlocked, rebuilding lootbox cache", "node_key", nodeKey)

	// Rebuild the cache
	if err := s.buildCache(s.lootTablesPath); err != nil {
		log.Error("Failed to rebuild lootbox cache after item unlock", "error", err, "node_key", nodeKey)
		return fmt.Errorf("failed to rebuild lootbox cache: %w", err)
	}

	log.Info("Lootbox cache successfully rebuilt", "node_key", nodeKey)
	return nil
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
