package naming

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// AliasPool contains alias variants for an item
type AliasPool struct {
	Default []string            `json:"default"`
	Themes  map[string][]string `json:"themes"`
}

// ThemePeriod defines the active period for a theme
type ThemePeriod struct {
	Start string `json:"start"` // MM-DD format
	End   string `json:"end"`   // MM-DD format
}

// Resolver handles item name resolution and display name generation
type Resolver interface {
	// ResolvePublicName converts a public name to internal name
	ResolvePublicName(publicName string) (internalName string, ok bool)

	// GetDisplayName generates a display name with optional shine prefix
	GetDisplayName(internalName string, shineLevel string) string

	// GetActiveTheme returns the currently active theme based on date
	GetActiveTheme() string

	// Reload reloads the alias and theme configurations
	Reload() error

	// RegisterItem registers an item for name resolution
	RegisterItem(internalName, publicName string)
}

type resolver struct {
	mu sync.RWMutex

	// Mapping: public_name -> internal_name
	publicToInternal map[string]string

	// Mapping: internal_name -> public_name (reverse lookup)
	internalToPublic map[string]string

	// Alias pools keyed by internal_name
	aliases map[string]AliasPool

	// Theme periods
	themes map[string]ThemePeriod

	// Config paths
	aliasesPath string
	themesPath  string
}

// NewResolver creates a new naming resolver
func NewResolver(aliasesPath, themesPath string) (Resolver, error) {
	r := &resolver{
		publicToInternal: make(map[string]string),
		internalToPublic: make(map[string]string),
		aliases:          make(map[string]AliasPool),
		themes:           make(map[string]ThemePeriod),
		aliasesPath:      aliasesPath,
		themesPath:       themesPath,
	}

	if err := r.Reload(); err != nil {
		return nil, err
	}

	return r, nil
}

// RegisterItem adds a public->internal name mapping
func (r *resolver) RegisterItem(internalName, publicName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if publicName != "" {
		r.publicToInternal[strings.ToLower(publicName)] = internalName
		r.internalToPublic[internalName] = publicName
	}
}

// ResolvePublicName converts a public name to internal name
func (r *resolver) ResolvePublicName(publicName string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	internal, ok := r.publicToInternal[strings.ToLower(publicName)]
	return internal, ok
}

// GetDisplayName generates a display name with optional shine prefix
func (r *resolver) GetDisplayName(internalName string, shineLevel string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get alias pool for this item
	pool, ok := r.aliases[internalName]
	if !ok || (len(pool.Default) == 0 && len(pool.Themes) == 0) {
		// No alias pool, return internal name with shine
		return r.formatWithShine(internalName, shineLevel)
	}

	// Check for active theme
	activeTheme := r.getActiveThemeUnlocked()
	var aliases []string

	if activeTheme != "" {
		if themeAliases, exists := pool.Themes[activeTheme]; exists && len(themeAliases) > 0 {
			aliases = themeAliases
		}
	}

	// Fall back to default aliases
	if len(aliases) == 0 {
		aliases = pool.Default
	}

	if len(aliases) == 0 {
		return r.formatWithShine(internalName, shineLevel)
	}

	// Random selection from alias pool
	alias := aliases[utils.RandomInt(0, len(aliases)-1)]
	return r.formatWithShine(alias, shineLevel)
}

// formatWithShine adds shine level prefix if not COMMON
func (r *resolver) formatWithShine(name string, shineLevel string) string {
	if shineLevel == "" || shineLevel == "COMMON" {
		return name
	}
	return fmt.Sprintf("%s %s", shineLevel, name)
}

// GetActiveTheme returns the currently active theme
func (r *resolver) GetActiveTheme() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.getActiveThemeUnlocked()
}

// getActiveThemeUnlocked returns active theme (caller must hold lock)
func (r *resolver) getActiveThemeUnlocked() string {
	now := time.Now()
	for theme, period := range r.themes {
		if isInPeriod(now, period.Start, period.End) {
			return theme
		}
	}
	return ""
}

// isInPeriod checks if current time is within the period (handles year wrap)
func isInPeriod(now time.Time, startStr, endStr string) bool {
	startMonth, startDay := parseMonthDay(startStr)
	endMonth, endDay := parseMonthDay(endStr)

	if startMonth == 0 || endMonth == 0 {
		return false
	}

	currentMonth := int(now.Month())
	currentDay := now.Day()

	// Create comparable date values (month * 100 + day)
	current := currentMonth*100 + currentDay
	start := startMonth*100 + startDay
	end := endMonth*100 + endDay

	if start <= end {
		// Normal range (e.g., 10-15 to 11-02)
		return current >= start && current <= end
	}
	// Year-wrapping range (e.g., 12-15 to 01-05)
	return current >= start || current <= end
}

// parseMonthDay parses "MM-DD" format
func parseMonthDay(s string) (month, day int) {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return 0, 0
	}
	month, _ = strconv.Atoi(parts[0])
	day, _ = strconv.Atoi(parts[1])
	return
}

// Reload reloads the alias and theme configurations
func (r *resolver) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load aliases
	if r.aliasesPath != "" {
		if err := r.loadAliases(); err != nil {
			return fmt.Errorf("failed to load aliases: %w", err)
		}
	}

	// Load themes
	if r.themesPath != "" {
		if err := r.loadThemes(); err != nil {
			return fmt.Errorf("failed to load themes: %w", err)
		}
	}

	return nil
}

func (r *resolver) loadVersionedConfig(path string, target interface{}, schema string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Wrapper to handle common fields
	var wrapper struct {
		Version string `json:"version"`
		Schema  string `json:"schema"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Errorf("failed to parse config %s: %w", path, err)
	}

	if wrapper.Version == "" {
		return fmt.Errorf("%s missing version field", path)
	}
	if wrapper.Schema != schema {
		return fmt.Errorf("invalid schema in %s: expected '%s', got '%s'", path, schema, wrapper.Schema)
	}

	// Now unmarshal to actual target
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to decode data for %s: %w", path, err)
	}

	return nil
}

func (r *resolver) loadAliases() error {
	var config struct {
		Aliases map[string]AliasPool `json:"aliases"`
	}
	if err := r.loadVersionedConfig(r.aliasesPath, &config, "item-aliases"); err != nil {
		return err
	}
	if config.Aliases != nil {
		r.aliases = config.Aliases
	}
	return nil
}

func (r *resolver) loadThemes() error {
	var config struct {
		Themes map[string]ThemePeriod `json:"themes"`
	}
	if err := r.loadVersionedConfig(r.themesPath, &config, "item-themes"); err != nil {
		return err
	}
	if config.Themes != nil {
		r.themes = config.Themes
	}
	return nil
}
