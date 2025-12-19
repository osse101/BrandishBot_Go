package naming

import (
	"testing"
	"time"
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
	name := r.GetDisplayName("lootbox_tier0", "")
	if name != "A dingy box" && name != "A worn box" {
		t.Errorf("GetDisplayName() = %v, want one of default aliases", name)
	}

	// Test with shine
	name = r.GetDisplayName("lootbox_tier0", "RARE")
	if name != "RARE A dingy box" && name != "RARE A worn box" {
		t.Errorf("GetDisplayName() with shine = %v, want RARE prefix", name)
	}

	// Test COMMON shine (should not show prefix)
	name = r.GetDisplayName("lootbox_tier0", "COMMON")
	if name == "COMMON A dingy box" || name == "COMMON A worn box" {
		t.Errorf("GetDisplayName() with COMMON shine should not show prefix, got %v", name)
	}

	// Test unknown item (should return internal name)
	name = r.GetDisplayName("unknown_item", "")
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
