package economy

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Helper function to create inventory slots for testing
func createSlot(itemID, quantity int, quality ...domain.QualityLevel) domain.InventorySlot {
	q := domain.QualityCommon
	if len(quality) > 0 {
		q = quality[0]
	}
	return domain.InventorySlot{
		ItemID:       itemID,
		Quantity:     quantity,
		QualityLevel: q,
	}
}

func TestProcessBuyTransaction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		initialInv    []domain.InventorySlot
		moneySlotIdx  int
		itemToBuyID   int
		quantityToBuy int
		cost          int
		expectedInv   []domain.InventorySlot
		desc          string
	}{
		{
			name: "Buy new item with exact money",
			initialInv: []domain.InventorySlot{
				createSlot(1, 100), // Money
			},
			moneySlotIdx:  0,
			itemToBuyID:   10,
			quantityToBuy: 5,
			cost:          100,
			expectedInv: []domain.InventorySlot{
				createSlot(10, 5),
			},
			desc: "Should remove money slot and add new item slot",
		},
		{
			name: "Buy new item with leftover money",
			initialInv: []domain.InventorySlot{
				createSlot(1, 200), // Money
			},
			moneySlotIdx:  0,
			itemToBuyID:   10,
			quantityToBuy: 5,
			cost:          100,
			expectedInv: []domain.InventorySlot{
				createSlot(1, 100), // Remaining money
				createSlot(10, 5),  // New item
			},
			desc: "Should reduce money and add new item slot",
		},
		{
			name: "Buy existing item (stacking)",
			initialInv: []domain.InventorySlot{
				createSlot(1, 200), // Money
				createSlot(10, 2),  // Existing item
			},
			moneySlotIdx:  0,
			itemToBuyID:   10,
			quantityToBuy: 5,
			cost:          100,
			expectedInv: []domain.InventorySlot{
				createSlot(1, 100), // Remaining money
				createSlot(10, 7),  // Stacked item (2 + 5)
			},
			desc: "Should stack with existing item slot",
		},
		{
			name: "Buy multiple distinct items",
			initialInv: []domain.InventorySlot{
				createSlot(1, 300), // Money
				createSlot(20, 1),  // Other item
			},
			moneySlotIdx:  0,
			itemToBuyID:   10,
			quantityToBuy: 5,
			cost:          100,
			expectedInv: []domain.InventorySlot{
				createSlot(1, 200), // Remaining money
				createSlot(20, 1),  // Other item untouched
				createSlot(10, 5),  // New item
			},
			desc: "Should add new item without disturbing others",
		},
		{
			name: "Buy existing item (different quality)",
			initialInv: []domain.InventorySlot{
				createSlot(1, 200),                    // Money
				createSlot(10, 2, domain.QualityRare), // Existing rare item
			},
			moneySlotIdx:  0,
			itemToBuyID:   10,
			quantityToBuy: 5,
			cost:          100,
			expectedInv: []domain.InventorySlot{
				createSlot(1, 100),                      // Remaining money
				createSlot(10, 2, domain.QualityRare),   // Rare item unaltered
				createSlot(10, 5, domain.QualityCommon), // New common item stack
			},
			desc: "Should add new common stack when only different quality exists",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inv := &domain.Inventory{Slots: append([]domain.InventorySlot{}, tt.initialInv...)}

			processBuyTransaction(inv, tt.itemToBuyID, tt.moneySlotIdx, tt.quantityToBuy, tt.cost)

			// Helper to check if inventory matches expected (order-independent for added items usually, but slice order matters here)
			assert.ElementsMatch(t, tt.expectedInv, inv.Slots, tt.desc)
		})
	}
}

func TestProcessSellTransaction(t *testing.T) {
	t.Parallel()

	moneyItemID := 1

	tests := []struct {
		name         string
		initialInv   []domain.InventorySlot
		itemSlotIdx  int
		sellQuantity int
		moneyGained  int
		expectedInv  []domain.InventorySlot
		desc         string
	}{
		{
			name: "Sell entire stack",
			initialInv: []domain.InventorySlot{
				createSlot(10, 5), // Item to sell
			},
			itemSlotIdx:  0,
			sellQuantity: 5,
			moneyGained:  100,
			expectedInv: []domain.InventorySlot{
				createSlot(moneyItemID, 100), // Money gained
			},
			desc: "Should remove item slot and add money slot",
		},
		{
			name: "Sell partial stack",
			initialInv: []domain.InventorySlot{
				createSlot(10, 10), // Item to sell
			},
			itemSlotIdx:  0,
			sellQuantity: 5,
			moneyGained:  100,
			expectedInv: []domain.InventorySlot{
				createSlot(10, 5),            // Remaining item
				createSlot(moneyItemID, 100), // Money gained
			},
			desc: "Should reduce item quantity and add money slot",
		},
		{
			name: "Sell with existing money",
			initialInv: []domain.InventorySlot{
				createSlot(10, 5),           // Item to sell
				createSlot(moneyItemID, 50), // Existing money
			},
			itemSlotIdx:  0,
			sellQuantity: 5,
			moneyGained:  100,
			expectedInv: []domain.InventorySlot{
				createSlot(moneyItemID, 150), // Stacked money (50 + 100)
			},
			desc: "Should remove item and stack money",
		},
		{
			name: "Sell specific quality slot",
			initialInv: []domain.InventorySlot{
				createSlot(10, 5, domain.QualityCommon), // Common
				createSlot(10, 1, domain.QualityRare),   // Rare (index 1)
			},
			itemSlotIdx:  1, // Selling Rare item
			sellQuantity: 1,
			moneyGained:  500,
			expectedInv: []domain.InventorySlot{
				createSlot(10, 5, domain.QualityCommon), // Common remains
				createSlot(moneyItemID, 500),            // Money added
			},
			desc: "Should remove correct slot based on index",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inv := &domain.Inventory{Slots: append([]domain.InventorySlot{}, tt.initialInv...)}

			processSellTransaction(inv, moneyItemID, tt.itemSlotIdx, tt.sellQuantity, tt.moneyGained)

			assert.ElementsMatch(t, tt.expectedInv, inv.Slots, tt.desc)
		})
	}
}

// Edge case: Selling into an inventory that has money in a weird state (e.g. split stacks)
// Our logic just finds *first* money slot.
func TestProcessSellTransaction_SplitMoneyStacks(t *testing.T) {
	t.Parallel()
	moneyItemID := 1

	// Setup: Inventory has two money slots (shouldn't happen normally, but robust code handles it)
	inv := &domain.Inventory{
		Slots: []domain.InventorySlot{
			createSlot(10, 5),
			createSlot(moneyItemID, 100),
			createSlot(moneyItemID, 50),
		},
	}

	// Act: Sell item
	processSellTransaction(inv, moneyItemID, 0, 5, 200)

	// Assert: Logic searches for first matching money slot to add to.
	expected := []domain.InventorySlot{
		createSlot(moneyItemID, 300), // 100 + 200
		createSlot(moneyItemID, 50),
	}

	assert.ElementsMatch(t, expected, inv.Slots)
}

// Edge case: Buying when money is the last slot and gets removed
func TestProcessBuyTransaction_RemoveLastSlot(t *testing.T) {
	t.Parallel()

	// Setup: Money is at end
	inv := &domain.Inventory{
		Slots: []domain.InventorySlot{
			createSlot(20, 1),  // Item A
			createSlot(1, 100), // Money (exact amount)
		},
	}

	// Act: Buy item B for 100
	processBuyTransaction(inv, 30, 1, 1, 100)

	// Assert
	expected := []domain.InventorySlot{
		createSlot(20, 1), // Item A
		createSlot(30, 1), // Item B
	}

	assert.ElementsMatch(t, expected, inv.Slots)
}
