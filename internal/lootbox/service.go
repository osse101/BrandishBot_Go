package lootbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
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
)

// Shine multipliers (Boosts Gamble Score)
const (
	MultCommon    = 1.0
	MultUncommon  = 1.1
	MultRare      = 1.25
	MultEpic      = 1.5
	MultLegendary = 2.0
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
	OpenLootbox(ctx context.Context, lootboxName string, quantity int) ([]DroppedItem, error)
}

type service struct {
	repo       ItemRepository
	lootTables map[string][]LootItem
	rnd        func() float64
}

// NewService creates a new lootbox service
func NewService(repo ItemRepository, lootTablesPath string) (Service, error) {
	svc := &service{
		repo:       repo,
		lootTables: make(map[string][]LootItem),
		rnd:        utils.RandomFloat,
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

	// Parse the nested structure with "tables" key
	var config struct {
		Tables map[string][]LootItem `json:"tables"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToParseLootFile, err)
	}

	s.lootTables = config.Tables
	return nil
}

// OpenLootbox simulates opening lootboxes and returns the dropped items
func (s *service) OpenLootbox(ctx context.Context, lootboxName string, quantity int) ([]DroppedItem, error) {
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

	return s.convertToDroppedItems(ctx, dropCounts)
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

func (s *service) convertToDroppedItems(ctx context.Context, dropCounts map[string]dropInfo) ([]DroppedItem, error) {
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

	drops := make([]DroppedItem, 0, len(dropCounts))
	for itemName, info := range dropCounts {
		item, found := itemMap[itemName]
		if !found {
			log.Warn(LogMsgDroppedItemNotInDB, LogFieldItem, itemName)
			continue
		}

		shine, mult := s.calculateShine(info.Chance)
		boostedValue := int(float64(item.BaseValue) * mult)

		drops = append(drops, DroppedItem{
			ItemID:     item.ID,
			ItemName:   item.InternalName,
			Quantity:   info.Qty,
			Value:      boostedValue,
			ShineLevel: shine,
		})
	}

	return drops, nil
}

// calculateShine determines the visual rarity "shine" and value multiplier of a drop based on its chance
func (s *service) calculateShine(chance float64) (string, float64) {
	shine := ShineCommon
	if chance <= ShineLegendaryThreshold {
		shine = ShineLegendary
	} else if chance <= ShineEpicThreshold {
		shine = ShineEpic
	} else if chance <= ShineRareThreshold {
		shine = ShineRare
	} else if chance <= ShineUncommonThreshold {
		shine = ShineUncommon
	}

	// Critical Shine Upgrade: 1% chance to upgrade the shine level
	// This adds a fun "Lucky!" moment for players
	if s.rnd() < CriticalShineUpgradeChance {
		switch shine {
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

	var mult float64
	switch shine {
	case ShineLegendary:
		mult = MultLegendary
	case ShineEpic:
		mult = MultEpic
	case ShineRare:
		mult = MultRare
	case ShineUncommon:
		mult = MultUncommon
	default:
		mult = MultCommon
	}

	return shine, mult
}
