package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLootboxEventData_ToMap(t *testing.T) {
	// Create sample event data
	eventData := &LootboxEventData{
		Item:   "lootbox_tier2",
		Drops:  []string{"item1", "item2"}, // Simulating drops as strings for test
		Value:  150,
		Source: "lootbox",
	}

	// Convert to map
	result := eventData.ToMap()

	// Verify all fields are present
	assert.Equal(t, "lootbox_tier2", result["item"])
	assert.Equal(t, []string{"item1", "item2"}, result["drops"])
	assert.Equal(t, 150, result["value"])
	assert.Equal(t, "lootbox", result["source"])

	// Verify map has exactly 4 keys
	assert.Len(t, result, 4)
}

func TestLootboxEventData_ToMap_NilDrops(t *testing.T) {
	// Test with nil drops
	eventData := &LootboxEventData{
		Item:   "test_item",
		Drops:  nil,
		Value:  0,
		Source: "test",
	}

	result := eventData.ToMap()

	assert.Nil(t, result["drops"])
	assert.Equal(t, "test_item", result["item"])
	assert.Equal(t, 0, result["value"])
	assert.Equal(t, "test", result["source"])
}
