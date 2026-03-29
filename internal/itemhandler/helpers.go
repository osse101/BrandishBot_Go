package itemhandler

import (
	"strings"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func Pluralize(name string, quantity int) string {
	if quantity <= 1 || name == "" {
		return name
	}

	// Check for quality emojis at the end (Legendary/Cursed)
	suffix := ""
	baseName := name
	// Emojis are multi-byte
	if strings.HasSuffix(name, "👑") {
		suffix = "👑"
		baseName = strings.TrimSuffix(name, "👑")
	} else if strings.HasSuffix(name, "👻") {
		suffix = "👻"
		baseName = strings.TrimSuffix(name, "👻")
	}

	// Handle "of" phrases: "pouch of coins" -> "pouches of coins"
	if strings.Contains(baseName, " of ") {
		parts := strings.SplitN(baseName, " of ", 2)
		return Pluralize(parts[0], quantity) + " of " + parts[1] + suffix
	}

	// Common uncountable or collective nouns in game context
	lower := strings.ToLower(baseName)
	switch lower {
	case domain.PublicNameMoney, "ghost-gold", "coins", "scrap", "junk", "credits":
		return baseName + suffix
	}
	if strings.HasSuffix(lower, " coins") {
		return baseName + suffix
	}

	// Basic pluralization rules
	if strings.HasSuffix(baseName, "y") && len(baseName) > 1 {
		vowels := "aeiouAEIOU"
		if !strings.ContainsAny(string(baseName[len(baseName)-2]), vowels) {
			return baseName[:len(baseName)-1] + "ies" + suffix
		}
	}

	if strings.HasSuffix(baseName, "s") || strings.HasSuffix(baseName, "x") ||
		strings.HasSuffix(baseName, "ch") || strings.HasSuffix(baseName, "sh") {
		return baseName + "es" + suffix
	}

	return baseName + "s" + suffix
}

func getIndefiniteArticle(word string) string {
	if len(word) == 0 {
		return "a"
	}
	first := strings.ToLower(string(word[0]))
	if strings.ContainsAny(first, "aeiou") {
		return "an"
	}
	return "a"
}

func getWeaponTimeout(itemName string) time.Duration {
	if timeout, ok := weaponTimeouts[itemName]; ok {
		return timeout
	}
	return domain.BlasterTimeoutDuration // default fallback
}
