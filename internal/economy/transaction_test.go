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
		name         string
		initialInv   []domain.InventorySlot
		moneySlotIdx int
		itemToBuyID  int
		qtyToBuy     int
		cost         int
		expectedInv  []domain.InventorySlot
		desc         string
	}{
		{
			name: "Buy new item with exact money",
			initialInv: []domain.InventorySlot{
				createSlot(1, 100),
			},
			moneySlotIdx: 0,
			itemToBuyID:  10,
			qtyToBuy:     5,
			cost:         100,
			expectedInv: []domain.InventorySlot{
				createSlot(10, 5),
			},
			desc: "Should remove money slot and add new item slot",
		},
		{
			name: "Buy new item with leftover money",
			initialInv: []domain.InventorySlot{
				createSlot(1, 200),
			},
			moneySlotIdx: 0,
			itemToBuyID:  10,
			qtyToBuy:     5,
			cost:         100,
			expectedInv: []domain.InventorySlot{
				createSlot(1, 100),
				createSlot(10, 5),
			},
			desc: "Should reduce money and add new item slot",
		},
		{
			name: "Buy existing item (stacking)",
			initialInv: []domain.InventorySlot{
				createSlot(1, 200),
				createSlot(10, 2),
			},
			moneySlotIdx: 0,
			itemToBuyID:  10,
			qtyToBuy:     5,
			cost:         100,
			expectedInv: []domain.InventorySlot{
				createSlot(1, 100),
				createSlot(10, 7),
			},
			desc: "Should stack with existing item slot",
		},
		{
			name: "Buy multiple distinct items",
			initialInv: []domain.InventorySlot{
				createSlot(1, 300),
				createSlot(20, 1),
			},
			moneySlotIdx: 0,
			itemToBuyID:  10,
			qtyToBuy:     5,
			cost:         100,
			expectedInv: []domain.InventorySlot{
				createSlot(1, 200),
				createSlot(20, 1),
				createSlot(10, 5),
			},
			desc: "Should add new item without disturbing others",
		},
		{
			name: "Buy existing item (different quality)",
			initialInv: []domain.InventorySlot{
				createSlot(1, 200),
				createSlot(10, 2, domain.QualityRare),
			},
			moneySlotIdx: 0,
			itemToBuyID:  10,
			qtyToBuy:     5,
			cost:         100,
			expectedInv: []domain.InventorySlot{
				createSlot(1, 100),
				createSlot(10, 2, domain.QualityRare),
				createSlot(10, 5, domain.QualityCommon),
			},
			desc: "Should add new common stack when only different quality exists",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inv := &domain.Inventory{Slots: append([]domain.InventorySlot{}, tt.initialInv...)}

			processBuyTransaction(inv, tt.itemToBuyID, tt.moneySlotIdx, tt.qtyToBuy, tt.cost)

			// Helper to check if inventory matches expected (order-independent for added items usually, but slice order matters here)
			assert.ElementsMatch(t, tt.expectedInv, inv.Slots, tt.desc)
		})
	}
}

func TestProcessSellTransaction(t *testing.T) {
	t.Parallel()

	moneyItemID := 1

	tests := []struct {
		name        string
		initialInv  []domain.InventorySlot
		itemSlotIdx int
		sellQty     int
		moneyGained int
		expectedInv []domain.InventorySlot
		desc        string
	}{
		{
			name: "Sell entire stack",
			initialInv: []domain.InventorySlot{
				createSlot(10, 5),
			},
			itemSlotIdx: 0,
			sellQty:     5,
			moneyGained: 100,
			expectedInv: []domain.InventorySlot{
				createSlot(moneyItemID, 100),
			},
			desc: "Should remove item slot and add money slot",
		},
		{
			name: "Sell partial stack",
			initialInv: []domain.InventorySlot{
				createSlot(10, 10),
			},
			itemSlotIdx: 0,
			sellQty:     5,
			moneyGained: 100,
			expectedInv: []domain.InventorySlot{
				createSlot(10, 5),
				createSlot(moneyItemID, 100),
			},
			desc: "Should reduce item quantity and add money slot",
		},
		{
			name: "Sell with existing money",
			initialInv: []domain.InventorySlot{
				createSlot(10, 5),
				createSlot(moneyItemID, 50),
			},
			itemSlotIdx: 0,
			sellQty:     5,
			moneyGained: 100,
			expectedInv: []domain.InventorySlot{
				createSlot(moneyItemID, 150),
			},
			desc: "Should remove item and stack money",
		},
		{
			name: "Sell specific quality slot",
			initialInv: []domain.InventorySlot{
				createSlot(10, 5, domain.QualityCommon),
				createSlot(10, 1, domain.QualityRare),
			},
			itemSlotIdx: 1,
			sellQty:     1,
			moneyGained: 500,
			expectedInv: []domain.InventorySlot{
				createSlot(10, 5, domain.QualityCommon),
				createSlot(moneyItemID, 500),
			},
			desc: "Should remove correct slot based on index",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inv := &domain.Inventory{Slots: append([]domain.InventorySlot{}, tt.initialInv...)}

			processSellTransaction(inv, moneyItemID, tt.itemSlotIdx, tt.sellQty, tt.moneyGained)

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

// Edge case: Buying an item when the user already has multiple stacks of that exact item
func TestProcessBuyTransaction_SplitItemStacks(t *testing.T) {
	t.Parallel()

	// Setup: Inventory has two sub-stacks of the item being bought
	inv := &domain.Inventory{
		Slots: []domain.InventorySlot{
			createSlot(10, 2),  // Item A (stack 1)
			createSlot(1, 100), // Money
			createSlot(10, 3),  // Item A (stack 2)
		},
	}

	// Act: Buy 5 more of Item A (ID: 10) for 100 money, money is at index 1
	processBuyTransaction(inv, 10, 1, 5, 100)

	// Assert: Logic should add to the first encountered stack (the one at index 0)
	expected := []domain.InventorySlot{
		createSlot(10, 7), // First stack gets added to: 2 + 5
		createSlot(10, 3), // Second stack remains
	}

	assert.ElementsMatch(t, expected, inv.Slots)
}

// Edge case: Buying an item with a cost of 0
func TestProcessBuyTransaction_ZeroCost(t *testing.T) {
	t.Parallel()

	// Setup: Money slot is present, but cost is 0
	inv := &domain.Inventory{
		Slots: []domain.InventorySlot{
			createSlot(1, 100), // Money
		},
	}

	// Act: Buy item with 0 cost
	processBuyTransaction(inv, 20, 0, 1, 0)

	// Assert: Money should not decrease
	expected := []domain.InventorySlot{
		createSlot(1, 100), // Money unchanged
		createSlot(20, 1),  // New item
	}

	assert.ElementsMatch(t, expected, inv.Slots)
}
