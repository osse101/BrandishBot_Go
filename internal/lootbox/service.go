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

// Shine levels
const (
	ShineCommon    = "COMMON"
	ShineUncommon  = "UNCOMMON"
	ShineRare      = "RARE"
	ShineEpic      = "EPIC"
	ShineLegendary = "LEGENDARY"
	ShinePoor      = "POOR"
	ShineJunk      = "JUNK"
	ShineCursed    = "CURSED"
)

// Shine multipliers (Boosts Gamble Score)
const (
	MultCommon    = 1.0
	MultUncommon  = 1.1
	MultRare      = 1.25
	MultEpic      = 1.5
	MultLegendary = 2.0
	MultPoor      = 0.8
	MultJunk      = 0.6
	MultCursed    = 0.4
)

// Schema paths
const (
	LootTablesSchemaPath = "configs/schemas/loot_tables.schema.json"
)

// DroppedItem represents an item generated from opening a lootbox
type DroppedItem struct {
	ItemID     int
	ItemName   string
	Quantity   int
	Value      int
	ShineLevel string
}

// ItemRepository defines the interface for fetching item data
type ItemRepository interface {
	GetItemByName(ctx context.Context, name string) (*domain.Item, error)
	GetItemsByNames(ctx context.Context, names []string) ([]domain.Item, error)
}

// Service defines the lootbox opening interface
type Service interface {
	OpenLootbox(ctx context.Context, lootboxName string, quantity int, boxShine string) ([]DroppedItem, error)
}

// ProgressionService defines the interface for checking feature unlocks
type ProgressionService interface {
	IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error)
}

type service struct {
	repo            ItemRepository
	progressionSvc  ProgressionService
	lootTables      map[string][]LootItem
	rnd             func() float64
	schemaValidator validation.SchemaValidator
}

// NewService creates a new lootbox service
func NewService(repo ItemRepository, progressionSvc ProgressionService, lootTablesPath string) (Service, error) {
	svc := &service{
		repo:            repo,
		progressionSvc:  progressionSvc,
		lootTables:      make(map[string][]LootItem),
		rnd:             utils.RandomFloat,
		schemaValidator: validation.NewSchemaValidator(),
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
func (s *service) OpenLootbox(ctx context.Context, lootboxName string, quantity int, boxShine string) ([]DroppedItem, error) {
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

	return s.convertToDroppedItems(ctx, dropCounts, boxShine)
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

func (s *service) convertToDroppedItems(ctx context.Context, dropCounts map[string]dropInfo, boxShine string) ([]DroppedItem, error) {
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

		shine, mult := s.calculateShine(info.Chance, boxShine, canUpgrade)

		quantity := info.Qty
		boostedValue := int(float64(item.BaseValue) * mult)

		// Money special logic: scale quantity instead of individual value
		if itemName == "money" {
			quantity = int(float64(info.Qty) * mult)
			if info.Qty > 0 && quantity == 0 {
				quantity = 1
			}
			boostedValue = item.BaseValue // Keep base value (usually 1)
		} else {
			// Normal item truncation protection
			if item.BaseValue > 0 && boostedValue == 0 {
				boostedValue = 1
			}
		}

		drops = append(drops, DroppedItem{
			ItemID:     item.ID,
			ItemName:   item.InternalName,
			Quantity:   quantity,
			Value:      boostedValue,
			ShineLevel: shine,
		})
	}

	return drops, nil
}

// calculateShine determines the visual rarity "shine" and value multiplier of a drop based on its chance
// The boxShine level shifts the constraints: a more rare box makes it easier to get rare item shine levels.
func (s *service) calculateShine(chance float64, boxShine string, canUpgrade bool) (string, float64) {
	dist := s.getShineDistance(boxShine)
	bonus := 0.03 * float64(dist)

	shine := ShineCommon
	if chance <= ShineLegendaryThreshold+bonus {
		shine = ShineLegendary
	} else if chance <= ShineEpicThreshold+bonus {
		shine = ShineEpic
	} else if chance <= ShineRareThreshold+bonus {
		shine = ShineRare
	} else if chance <= ShineUncommonThreshold+bonus {
		shine = ShineUncommon
	} else if chance <= ShineCommonThreshold+bonus {
		shine = ShineCommon
	} else if chance <= ShinePoorThreshold+bonus {
		shine = ShinePoor
	} else if chance <= ShineJunkThreshold+bonus {
		shine = ShineJunk
	} else {
		shine = ShineCursed
	}

	// Critical Shine Upgrade: 1% chance to upgrade the shine level (locked by progression)
	if canUpgrade && s.rnd() < CriticalShineUpgradeChance {
		switch shine {
		case ShineCursed:
			shine = ShineJunk
		case ShineJunk:
			shine = ShinePoor
		case ShinePoor:
			shine = ShineCommon
		case ShineCommon:
			shine = ShineUncommon
		case ShineUncommon:
			shine = ShineRare
		case ShineRare:
			shine = ShineEpic
		case ShineEpic:
			shine = ShineLegendary
		}
	}

	return shine, s.getShineMultiplier(shine)
}

func (s *service) getShineDistance(shine string) int {
	switch shine {
	case ShineLegendary:
		return 4
	case ShineEpic:
		return 3
	case ShineRare:
		return 2
	case ShineUncommon:
		return 1
	case ShineCommon:
		return 0
	case ShinePoor:
		return -1
	case ShineJunk:
		return -2
	case ShineCursed:
		return -3
	default:
		return 0
	}
}

func (s *service) getShineMultiplier(shine string) float64 {
	switch shine {
	case ShineLegendary:
		return MultLegendary
	case ShineEpic:
		return MultEpic
	case ShineRare:
		return MultRare
	case ShineUncommon:
		return MultUncommon
	case ShinePoor:
		return MultPoor
	case ShineJunk:
		return MultJunk
	case ShineCursed:
		return MultCursed
	default:
		return MultCommon
	}
}
