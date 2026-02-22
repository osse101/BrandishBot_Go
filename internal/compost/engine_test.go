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

func TestCalculateInputValue(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		items    []domain.CompostBinItem
		expected int
	}{
		{
			name: "Mixed items",
			items: []domain.CompostBinItem{
				{BaseValue: 100, Quantity: 2, QualityLevel: domain.QualityCommon},   // 100 * 1.0 * 2 = 200
				{BaseValue: 50, Quantity: 1, QualityLevel: domain.QualityLegendary}, // 50 * 2.0 * 1 = 100
				{BaseValue: 30, Quantity: 3, QualityLevel: domain.QualityJunk},      // 30 * 0.6 * 3 = 54
			},
			expected: 354,
		},
		{
			name:     "Empty items",
			items:    nil,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := engine.CalculateInputValue(tt.items)
			assert.Equal(t, tt.expected, value)
		})
	}
}

func TestDetermineDominantType(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		items    []domain.CompostBinItem
		expected string
	}{
		{
			name: "Single type",
			items: []domain.CompostBinItem{
				{BaseValue: 100, Quantity: 1, QualityLevel: domain.QualityCommon, ContentTypes: []string{"weapon"}},
				{BaseValue: 100, Quantity: 1, QualityLevel: domain.QualityCommon, ContentTypes: []string{"weapon"}},
			},
			expected: "weapon",
		},
		{
			name: "Mixed types",
			items: []domain.CompostBinItem{
				{BaseValue: 100, Quantity: 3, QualityLevel: domain.QualityCommon, ContentTypes: []string{"weapon"}},   // 300
				{BaseValue: 200, Quantity: 2, QualityLevel: domain.QualityCommon, ContentTypes: []string{"material"}}, // 400
			},
			expected: "material",
		},
		{
			name:     "Empty items",
			items:    nil,
			expected: domain.ContentTypeMaterial,
		},
		{
			name: "Items with no content types",
			items: []domain.CompostBinItem{
				{BaseValue: 100, Quantity: 1, QualityLevel: domain.QualityCommon, ContentTypes: nil},
			},
			expected: domain.ContentTypeMaterial,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dominant := engine.DetermineDominantType(tt.items)
			assert.Equal(t, tt.expected, dominant)
		})
	}
}

func TestCalculateOutput(t *testing.T) {
	engine := NewEngine()
	allItems := []domain.Item{
		{InternalName: "weapon_big", BaseValue: 200, ContentType: []string{"weapon"}},
		{InternalName: "weapon_med", BaseValue: 45, ContentType: []string{"weapon"}},
		{InternalName: "weapon_small", BaseValue: 30, ContentType: []string{"weapon"}},
		{InternalName: "material_iron", BaseValue: 10, ContentType: []string{"material"}},
	}

	tests := []struct {
		name         string
		inputValue   int
		dominantType string
		isSludge     bool
		items        []domain.Item
		multiplier   float64
		check        func(t *testing.T, output *domain.CompostOutput)
	}{
		{
			name:         "Normal output",
			inputValue:   100,
			dominantType: "weapon",
			isSludge:     false,
			items:        allItems,
			multiplier:   0.45, // 100 * 0.45 = 45
			check: func(t *testing.T, output *domain.CompostOutput) {
				assert.False(t, output.IsSludge)
				assert.Equal(t, 45, output.TotalValue)
				// Should pick weapon_med (base_value=45 <= 45)
				assert.Equal(t, 1, output.Items["weapon_med"])
				assert.Equal(t, "Composting complete!", output.Message)
			},
		},
		{
			name:         "Sludge output",
			inputValue:   100,
			dominantType: "weapon",
			isSludge:     true,
			items:        nil,
			multiplier:   0.5,
			check: func(t *testing.T, output *domain.CompostOutput) {
				assert.True(t, output.IsSludge)
				assert.Equal(t, 10, output.Items["compost_sludge"]) // 100/10 = 10
				assert.Equal(t, "Your compost sat too long and turned to sludge!", output.Message)
			},
		},
		{
			name:         "Sludge minimum one",
			inputValue:   5,
			dominantType: "weapon",
			isSludge:     true,
			items:        nil,
			multiplier:   0.5,
			check: func(t *testing.T, output *domain.CompostOutput) {
				assert.True(t, output.IsSludge)
				assert.Equal(t, 1, output.Items["compost_sludge"]) // 5/10 = 0, clamped to 1
			},
		},
		{
			name:         "No matching items fallback",
			inputValue:   100,
			dominantType: "weapon",
			isSludge:     false,
			items: []domain.Item{
				{InternalName: "material_iron", BaseValue: 10, ContentType: []string{"material"}},
			},
			multiplier: 0.5,
			check: func(t *testing.T, output *domain.CompostOutput) {
				assert.False(t, output.IsSludge)
				assert.Equal(t, 50, output.Items["money"])
				assert.Contains(t, output.Message, "converted to money")
			},
		},
		{
			name:         "Output value < 1 clamped",
			inputValue:   1,
			dominantType: "weapon",
			isSludge:     false,
			items:        allItems,
			multiplier:   0.1, // 1 * 0.1 = 0.1 -> 0 (round) -> clamped to 1
			check: func(t *testing.T, output *domain.CompostOutput) {
				assert.False(t, output.IsSludge)
				assert.Equal(t, 1, output.TotalValue)
				// Should fallback to money since cheapest item is 10, which is > 1.
				assert.Equal(t, 1, output.Items["money"])
			},
		},
		{
			name:         "Highest base value selection",
			inputValue:   100,
			dominantType: "weapon",
			isSludge:     false,
			items:        allItems,
			multiplier:   0.40, // 100 * 0.40 = 40
			check: func(t *testing.T, output *domain.CompostOutput) {
				// Pick highest base value <= 40: weapon_small (30) is the only one <= 40, so we get 1.
				assert.Equal(t, 1, output.Items["weapon_small"])
				assert.Equal(t, 40, output.TotalValue)
			},
		},
		{
			name:         "Quantity calculation clamped to 1",
			inputValue:   100,
			dominantType: "weapon",
			isSludge:     false,
			items:        allItems,
			multiplier:   0.35, // 100 * 0.35 = 35
			check: func(t *testing.T, output *domain.CompostOutput) {
				// Target 35 / base weapon_small 30 = 1.
				assert.Equal(t, 1, output.Items["weapon_small"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := engine.CalculateOutput(tt.inputValue, tt.dominantType, tt.isSludge, tt.items, tt.multiplier)
			tt.check(t, output)
		})
	}
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
