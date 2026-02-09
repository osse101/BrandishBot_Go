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

type dropInfo struct {
	Qty    int
	Chance float64
}

func (s *service) processLootTable(table []LootItem, quantity int) map[string]dropInfo {
	dropCounts := make(map[string]dropInfo)

	for _, loot := range table {
		if loot.Chance <= ZeroChanceThreshold {
			continue
		}

		if loot.Chance >= GuaranteedDropThreshold {
			s.processGuaranteedDrop(loot, quantity, dropCounts)
		} else {
			s.processChanceDrop(loot, quantity, dropCounts)
		}
	}
	return dropCounts
}

func (s *service) processGuaranteedDrop(loot LootItem, quantity int, dropCounts map[string]dropInfo) {
	totalQty := 0
	if loot.Max > loot.Min {
		for k := 0; k < quantity; k++ {
			totalQty += utils.SecureRandomIntRange(loot.Min, loot.Max)
		}
	} else {
		totalQty = quantity * loot.Min
	}

	s.updateDropCounts(loot, totalQty, dropCounts)
}

func (s *service) processChanceDrop(loot LootItem, quantity int, dropCounts map[string]dropInfo) {
	remaining := quantity
	for remaining > 0 {
		skip := utils.Geometric(loot.Chance)
		if skip >= remaining {
			break
		}

		remaining -= (skip + 1)
		qty := loot.Min
		if loot.Max > loot.Min {
			qty = utils.SecureRandomIntRange(loot.Min, loot.Max)
		}
		s.updateDropCounts(loot, qty, dropCounts)
	}
}

func (s *service) updateDropCounts(loot LootItem, qty int, dropCounts map[string]dropInfo) {
	info, exists := dropCounts[loot.ItemName]
	if !exists {
		info = dropInfo{Qty: 0, Chance: loot.Chance}
	} else if loot.Chance < info.Chance {
		info.Chance = loot.Chance
	}
	info.Qty += qty
	dropCounts[loot.ItemName] = info
}

func (s *service) convertToDroppedItems(ctx context.Context, dropCounts map[string]dropInfo, boxQuality domain.QualityLevel) ([]DroppedItem, error) {
	log := logger.FromContext(ctx)

	itemNames := make([]string, 0, len(dropCounts))
	for itemName := range dropCounts {
		itemNames = append(itemNames, itemName)
	}

	items, err := s.repo.GetItemsByNames(ctx, itemNames)
	if err != nil {
		log.Error(ErrContextFailedToGetDroppedItems, LogFieldError, err)
		return nil, err
	}

	itemMap := make(map[string]*domain.Item, len(items))
	for i := range items {
		itemMap[items[i].InternalName] = &items[i]
	}

	// Check if lucky upgrade is unlocked via progression
	canUpgrade := false
	if s.progressionSvc != nil {
		// "feature_gamble" is the key for the gamble feature which unlocks lucky upgrades
		unlocked, err := s.progressionSvc.IsNodeUnlocked(ctx, "feature_gamble", 1)
		if err == nil {
			canUpgrade = unlocked
		}
	}

	drops := make([]DroppedItem, 0, len(dropCounts))
	for itemName, info := range dropCounts {
		item, found := itemMap[itemName]
		if !found {
			log.Warn(LogMsgDroppedItemNotInDB, LogFieldItem, itemName)
			continue
		}

		// Use a random roll for quality.
		quality, mult := s.calculateQuality(s.rnd(), boxQuality, canUpgrade)

		quantity := info.Qty
		boostedValue := int(float64(item.BaseValue) * mult)

		// Currency special logic: convert quality to quantity, force COMMON quality
		if item.IsCurrency() {
			quantity = int(float64(info.Qty) * mult)
			if info.Qty > 0 && quantity == 0 {
				quantity = 1
			}
			boostedValue = item.BaseValue  // Keep base value (usually 1)
			quality = domain.QualityCommon // Force COMMON for all currency
		} else {
			// Normal item truncation protection
			if item.BaseValue > 0 && boostedValue == 0 {
				boostedValue = 1
			}
		}

		drops = append(drops, DroppedItem{
			ItemID:       item.ID,
			ItemName:     item.InternalName,
			Quantity:     quantity,
			Value:        boostedValue,
			QualityLevel: quality,
		})
	}

	return drops, nil
}

// calculateQuality determines the visual rarity "quality" and value multiplier of a drop based on a roll.
// The boxQuality level shifts the constraints: a more rare box makes it easier to get rare item quality levels.
func (s *service) calculateQuality(roll float64, boxQuality domain.QualityLevel, canUpgrade bool) (domain.QualityLevel, float64) {
	dist := s.getQualityDistance(boxQuality)
	bonus := 0.03 * float64(dist)

	quality := domain.QualityCursed
	if roll <= QualityLegendaryThreshold+bonus {
		quality = domain.QualityLegendary
	} else if roll <= QualityEpicThreshold+bonus {
		quality = domain.QualityEpic
	} else if roll <= QualityRareThreshold+bonus {
		quality = domain.QualityRare
	} else if roll <= QualityUncommonThreshold+bonus {
		quality = domain.QualityUncommon
	} else if roll <= QualityCommonThreshold+bonus {
		quality = domain.QualityCommon
	} else if roll <= QualityPoorThreshold+bonus {
		quality = domain.QualityPoor
	} else if roll <= QualityJunkThreshold+bonus {
		quality = domain.QualityJunk
	}

	// Critical Quality Upgrade: 1% chance to upgrade the quality level (locked by progression)
	if canUpgrade && s.rnd() < CriticalQualityUpgradeChance {
		quality = s.getNextQualityLevel(quality)
	}

	return quality, s.getQualityMultiplier(quality)
}

func (s *service) getNextQualityLevel(q domain.QualityLevel) domain.QualityLevel {
	switch q {
	case domain.QualityCursed:
		return domain.QualityJunk
	case domain.QualityJunk:
		return domain.QualityPoor
	case domain.QualityPoor:
		return domain.QualityCommon
	case domain.QualityCommon:
		return domain.QualityUncommon
	case domain.QualityUncommon:
		return domain.QualityRare
	case domain.QualityRare:
		return domain.QualityEpic
	case domain.QualityEpic:
		return domain.QualityLegendary
	default:
		return q
	}
}

func (s *service) getQualityDistance(quality domain.QualityLevel) int {
	switch quality {
	case domain.QualityLegendary:
		return 4
	case domain.QualityEpic:
		return 3
	case domain.QualityRare:
		return 2
	case domain.QualityUncommon:
		return 1
	case domain.QualityCommon:
		return 0
	case domain.QualityPoor:
		return -1
	case domain.QualityJunk:
		return -2
	case domain.QualityCursed:
		return -3
	default:
		return 0
	}
}

func (s *service) getQualityMultiplier(quality domain.QualityLevel) float64 {
	switch quality {
	case domain.QualityLegendary:
		return domain.MultLegendary
	case domain.QualityEpic:
		return domain.MultEpic
	case domain.QualityRare:
		return domain.MultRare
	case domain.QualityUncommon:
		return domain.MultUncommon
	case domain.QualityPoor:
		return domain.MultPoor
	case domain.QualityJunk:
		return domain.MultJunk
	case domain.QualityCursed:
		return domain.MultCursed
	default:
		return domain.MultCommon
	}
}
