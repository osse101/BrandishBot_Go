package naming

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestResolvePublicName(t *testing.T) {
	r := &resolver{
		publicToInternal: map[string]string{
			"missile": "weapon_blaster",
			"junkbox": "lootbox_tier0",
		},
		internalToPublic: map[string]string{
			"weapon_blaster": "missile",
			"lootbox_tier0":  "junkbox",
		},
		aliases: make(map[string]AliasPool),
		themes:  make(map[string]ThemePeriod),
	}

	tests := []struct {
		name       string
		publicName string
		wantName   string
		wantOk     bool
	}{
		{"valid missile", "missile", "weapon_blaster", true},
		{"valid junkbox", "junkbox", "lootbox_tier0", true},
		{"case insensitive", "MISSILE", "weapon_blaster", true},
		{"unknown item", "unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := r.ResolvePublicName(tt.publicName)
			if ok != tt.wantOk {
				t.Errorf("ResolvePublicName() ok = %v, want %v", ok, tt.wantOk)
			}
			if got != tt.wantName {
				t.Errorf("ResolvePublicName() = %v, want %v", got, tt.wantName)
			}
		})
	}
}

func TestGetDisplayName(t *testing.T) {
	r := &resolver{
		publicToInternal: make(map[string]string),
		internalToPublic: make(map[string]string),
		aliases: map[string]AliasPool{
			"lootbox_tier0": {
				Default: []string{"A dingy box", "A worn box"},
				Themes: map[string][]string{
					"halloween": {"A spooky box"},
				},
			},
		},
		themes: make(map[string]ThemePeriod),
	}

	// Test without shine
	name := r.GetDisplayName("lootbox_tier0", domain.ShineLevel(""))
	if name != "A dingy box" && name != "A worn box" {
		t.Errorf("GetDisplayName() = %v, want one of default aliases", name)
	}

	// Test with shine
	name = r.GetDisplayName("lootbox_tier0", domain.ShineLevel("RARE"))
	if name != "RARE A dingy box" && name != "RARE A worn box" {
		t.Errorf("GetDisplayName() with shine = %v, want RARE prefix", name)
	}

	// Test COMMON shine (should not show prefix)
	name = r.GetDisplayName("lootbox_tier0", domain.ShineCommon)
	if name == "COMMON A dingy box" || name == "COMMON A worn box" {
		t.Errorf("GetDisplayName() with COMMON shine should not show prefix, got %v", name)
	}

	// Test unknown item (should return internal name)
	name = r.GetDisplayName("unknown_item", domain.ShineLevel(""))
	if name != "unknown_item" {
		t.Errorf("GetDisplayName() for unknown = %v, want unknown_item", name)
	}
}

func TestIsInPeriod(t *testing.T) {
	tests := []struct {
		name   string
		month  time.Month
		day    int
		start  string
		end    string
		expect bool
	}{
		{"in halloween period", time.October, 20, "10-15", "11-02", true},
		{"before halloween", time.October, 10, "10-15", "11-02", false},
		{"after halloween", time.November, 10, "10-15", "11-02", false},
		{"christmas wrap start", time.December, 20, "12-15", "01-05", true},
		{"christmas wrap end", time.January, 3, "12-15", "01-05", true},
		{"not in christmas", time.February, 1, "12-15", "01-05", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2024, tt.month, tt.day, 12, 0, 0, 0, time.UTC)
			got := isInPeriod(now, tt.start, tt.end)
			if got != tt.expect {
				t.Errorf("isInPeriod() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestGetActiveTheme(t *testing.T) {
	r := &resolver{
		publicToInternal: make(map[string]string),
		internalToPublic: make(map[string]string),
		aliases:          make(map[string]AliasPool),
		themes: map[string]ThemePeriod{
			"halloween": {Start: "10-15", End: "11-02"},
			"christmas": {Start: "12-15", End: "01-05"},
		},
	}

	// We can't easily test time-dependent behavior without mocking
	// Just verify the method doesn't crash
	theme := r.GetActiveTheme()
	_ = theme // Result depends on current date
}

func TestRegisterItem(t *testing.T) {
	r := &resolver{
		publicToInternal: make(map[string]string),
		internalToPublic: make(map[string]string),
		aliases:          make(map[string]AliasPool),
		themes:           make(map[string]ThemePeriod),
	}

	r.RegisterItem("weapon_blaster", "missile")

	internal, ok := r.ResolvePublicName("missile")
	if !ok {
		t.Error("RegisterItem() failed to register item")
	}
	if internal != "weapon_blaster" {
		t.Errorf("RegisterItem() internal = %v, want weapon_blaster", internal)
	}
}

// =============================================================================
// JSON Validation Tests
// =============================================================================

func TestLoadAliases_ValidJSON(t *testing.T) {
	r := &resolver{
		aliasesPath: "testdata/valid_aliases.json",
		aliases:     make(map[string]AliasPool),
	}

	err := r.loadAliases()
	require.NoError(t, err)

	// Verify loaded correctly
	pool, ok := r.aliases["lootbox_tier0"]
	assert.True(t, ok, "Should load lootbox_tier0")
	assert.Len(t, pool.Default, 2, "Should have 2 default aliases")
	assert.Contains(t, pool.Default, "A test box")
	assert.Contains(t, pool.Themes, "test_theme")
}

func TestLoadAliases_MalformedJSON(t *testing.T) {
	r := &resolver{
		aliasesPath: "testdata/malformed.json",
		aliases:     make(map[string]AliasPool),
	}

	err := r.loadAliases()
	assert.Error(t, err, "Should fail on malformed JSON")
}

func TestLoadAliases_MissingDefault(t *testing.T) {
	// Missing "default" field should be handled gracefully
	r := &resolver{
		aliasesPath: "testdata/missing_default.json",
		aliases:     make(map[string]AliasPool),
	}

	err := r.loadAliases()
	assert.NoError(t, err, "Should handle missing default gracefully")

	pool, ok := r.aliases["lootbox_tier0"]
	assert.True(t, ok)
	assert.Len(t, pool.Default, 0, "Default should be empty array")
	assert.Len(t, pool.Themes, 1, "Should still load themes")
}

func TestLoadAliases_EmptyDefault(t *testing.T) {
	// Empty default array is valid - means no aliases
	r := &resolver{
		aliasesPath: "testdata/empty_default.json",
		aliases:     make(map[string]AliasPool),
	}

	err := r.loadAliases()
	assert.NoError(t, err, "Empty default is valid")

	pool, ok := r.aliases["lootbox_tier0"]
	assert.True(t, ok)
	assert.Len(t, pool.Default, 0)
}

func TestLoadAliases_FileNotExist(t *testing.T) {
	// Non-existent file should not error (graceful degradation)
	r := &resolver{
		aliasesPath: "testdata/nonexistent.json",
		aliases:     make(map[string]AliasPool),
	}

	err := r.loadAliases()
	assert.NoError(t, err, "Should handle missing file gracefully")
	assert.Len(t, r.aliases, 0, "Aliases should be empty")
}

func TestLoadThemes_ValidJSON(t *testing.T) {
	r := &resolver{
		themesPath: "testdata/valid_themes.json",
		themes:     make(map[string]ThemePeriod),
	}

	err := r.loadThemes()
	require.NoError(t, err)

	period, ok := r.themes["test_theme"]
	assert.True(t, ok, "Should load test_theme")
	assert.Equal(t, "10-15", period.Start)
	assert.Equal(t, "11-02", period.End)
}

func TestLoadThemes_InvalidDates(t *testing.T) {
	// Invalid date formats should still parse (strings are valid JSON)
	// The date validation happens in isInPeriod
	r := &resolver{
		themesPath: "testdata/invalid_dates.json",
		themes:     make(map[string]ThemePeriod),
	}

	err := r.loadThemes()
	assert.NoError(t, err, "Invalid dates still valid JSON")

	period, ok := r.themes["bad_theme"]
	assert.True(t, ok)
	// Dates will fail at runtime in isInPeriod, not at load time
	assert.Equal(t, "October 15", period.Start)
}

func TestParseMonthDay_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMonth int
		wantDay   int
	}{
		{"valid", "10-15", 10, 15},
		{"single digit", "5-3", 5, 3},
		{"invalid format", "October", 0, 0},
		{"missing day", "10", 0, 0},
		{"too many parts", "10-15-2024", 0, 0},
		{"empty", "", 0, 0},
		{"non-numeric", "AA-BB", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			month, day := parseMonthDay(tt.input)
			assert.Equal(t, tt.wantMonth, month, "Month mismatch")
			assert.Equal(t, tt.wantDay, day, "Day mismatch")
		})
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestReload_PartialFailure(t *testing.T) {
	// If aliases fail but themes succeed, what happens?
	r := &resolver{
		aliasesPath: "testdata/malformed.json",
		themesPath:  "testdata/valid_themes.json",
		aliases:     make(map[string]AliasPool),
		themes:      make(map[string]ThemePeriod),
	}

	err := r.Reload()
	assert.Error(t, err, "Should error on aliases failure")
	// Themes shouldn't load if aliases fail (early exit)
}

func TestReload_BothValid(t *testing.T) {
	r := &resolver{
		aliasesPath: "testdata/valid_aliases.json",
		themesPath:  "testdata/valid_themes.json",
		aliases:     make(map[string]AliasPool),
		themes:      make(map[string]ThemePeriod),
	}

	err := r.Reload()
	assert.NoError(t, err)
	assert.NotEmpty(t, r.aliases, "Aliases should load")
	assert.NotEmpty(t, r.themes, "Themes should load")
}

func TestReload_EmptyPaths(t *testing.T) {
	// Empty paths should be handled gracefully
	r := &resolver{
		aliasesPath: "",
		themesPath:  "",
		aliases:     make(map[string]AliasPool),
		themes:      make(map[string]ThemePeriod),
	}

	err := r.Reload()
	assert.NoError(t, err, "Empty paths should be OK")
}

// =============================================================================
// Helper Imports for New Tests
// =============================================================================

// Note: Add these imports at the top of the file if not present:
// "github.com/stretchr/testify/assert"
// "github.com/stretchr/testify/require"
