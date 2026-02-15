package compost

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestCalculateReadyAt(t *testing.T) {
	engine := NewEngine()
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// Warmup (1h) + 3 items * 30m = 2h 30m
	readyAt := engine.CalculateReadyAt(start, 3)
	expected := start.Add(1*time.Hour + 3*30*time.Minute)
	assert.Equal(t, expected, readyAt)
}

func TestCalculateReadyAt_ExtendExisting(t *testing.T) {
	engine := NewEngine()
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// First deposit: 2 items
	readyAt1 := engine.CalculateReadyAt(start, 2)
	assert.Equal(t, start.Add(1*time.Hour+2*30*time.Minute), readyAt1)

	// After adding 3 more, total = 5
	readyAt2 := engine.CalculateReadyAt(start, 5)
	assert.Equal(t, start.Add(1*time.Hour+5*30*time.Minute), readyAt2)

	// Adding more items extends the time
	assert.True(t, readyAt2.After(readyAt1))
}

func TestCalculateSludgeAt(t *testing.T) {
	engine := NewEngine()
	readyAt := time.Date(2026, 1, 2, 14, 30, 0, 0, time.UTC)

	sludgeAt := engine.CalculateSludgeAt(readyAt)
	assert.Equal(t, readyAt.Add(168*time.Hour), sludgeAt)
}

func TestCalculateInputValue_Mixed(t *testing.T) {
	engine := NewEngine()
	items := []domain.CompostBinItem{
		{BaseValue: 100, Quantity: 2, QualityLevel: domain.QualityCommon},   // 100 * 1.0 * 2 = 200
		{BaseValue: 50, Quantity: 1, QualityLevel: domain.QualityLegendary}, // 50 * 2.0 * 1 = 100
		{BaseValue: 30, Quantity: 3, QualityLevel: domain.QualityJunk},      // 30 * 0.6 * 3 = 54
	}

	value := engine.CalculateInputValue(items)
	assert.Equal(t, 354, value)
}

func TestCalculateInputValue_Empty(t *testing.T) {
	engine := NewEngine()
	value := engine.CalculateInputValue(nil)
	assert.Equal(t, 0, value)
}

func TestDetermineDominantType_Single(t *testing.T) {
	engine := NewEngine()
	items := []domain.CompostBinItem{
		{BaseValue: 100, Quantity: 1, QualityLevel: domain.QualityCommon, ContentTypes: []string{"weapon"}},
		{BaseValue: 100, Quantity: 1, QualityLevel: domain.QualityCommon, ContentTypes: []string{"weapon"}},
	}

	dominant := engine.DetermineDominantType(items)
	assert.Equal(t, "weapon", dominant)
}

func TestDetermineDominantType_Mixed(t *testing.T) {
	engine := NewEngine()
	items := []domain.CompostBinItem{
		{BaseValue: 100, Quantity: 3, QualityLevel: domain.QualityCommon, ContentTypes: []string{"weapon"}},   // 300
		{BaseValue: 200, Quantity: 2, QualityLevel: domain.QualityCommon, ContentTypes: []string{"material"}}, // 400
	}

	dominant := engine.DetermineDominantType(items)
	assert.Equal(t, "material", dominant)
}

func TestDetermineDominantType_Empty(t *testing.T) {
	engine := NewEngine()

	// No items
	dominant := engine.DetermineDominantType(nil)
	assert.Equal(t, domain.ContentTypeMaterial, dominant)

	// Items with no content types
	items := []domain.CompostBinItem{
		{BaseValue: 100, Quantity: 1, QualityLevel: domain.QualityCommon, ContentTypes: nil},
	}
	dominant = engine.DetermineDominantType(items)
	assert.Equal(t, domain.ContentTypeMaterial, dominant)
}

func TestCalculateOutput_Normal(t *testing.T) {
	engine := NewEngine()
	allItems := []domain.Item{
		{InternalName: "weapon_big", BaseValue: 200, ContentType: []string{"weapon"}},
		{InternalName: "weapon_small", BaseValue: 50, ContentType: []string{"weapon"}},
		{InternalName: "material_iron", BaseValue: 10, ContentType: []string{"material"}},
	}

	// inputValue=100, multiplier=0.5 -> outputValue=50
	output := engine.CalculateOutput(100, "weapon", false, allItems, 0.5)
	assert.False(t, output.IsSludge)
	assert.Equal(t, 50, output.TotalValue)
	// Should pick weapon_small (base_value=50 <= 50)
	assert.Equal(t, 1, output.Items["weapon_small"])
	assert.Equal(t, "Composting complete!", output.Message)
}

func TestCalculateOutput_Sludge(t *testing.T) {
	engine := NewEngine()

	output := engine.CalculateOutput(100, "weapon", true, nil, 0.5)
	assert.True(t, output.IsSludge)
	assert.Equal(t, 10, output.Items["compost_sludge"]) // 100/10 = 10
	assert.Equal(t, "Your compost sat too long and turned to sludge!", output.Message)
}

func TestCalculateOutput_SludgeMinimumOne(t *testing.T) {
	engine := NewEngine()

	output := engine.CalculateOutput(5, "weapon", true, nil, 0.5)
	assert.True(t, output.IsSludge)
	assert.Equal(t, 1, output.Items["compost_sludge"]) // 5/10 = 0, clamped to 1
}

func TestCalculateOutput_NoMatchingItemsFallback(t *testing.T) {
	engine := NewEngine()
	allItems := []domain.Item{
		{InternalName: "material_iron", BaseValue: 10, ContentType: []string{"material"}},
	}

	// Dominant type is "weapon" but no weapon items exist -> fallback to money
	output := engine.CalculateOutput(100, "weapon", false, allItems, 0.5)
	assert.False(t, output.IsSludge)
	assert.Equal(t, 50, output.Items["money"])
	assert.Contains(t, output.Message, "converted to money")
}

func TestTotalItemCount(t *testing.T) {
	engine := NewEngine()
	items := []domain.CompostBinItem{
		{Quantity: 2},
		{Quantity: 3},
		{Quantity: 1},
	}

	assert.Equal(t, 6, engine.TotalItemCount(items))
}

func TestTotalItemCount_Empty(t *testing.T) {
	engine := NewEngine()
	assert.Equal(t, 0, engine.TotalItemCount(nil))
}
