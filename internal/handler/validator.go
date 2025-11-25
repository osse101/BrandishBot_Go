package handler

import (
	"fmt"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Platform validation
var ValidPlatforms = map[string]bool{
	"twitch":  true,
	"youtube": true,
	"discord": true,
}

// ValidatePlatform checks if the platform is supported
func ValidatePlatform(platform string) error {
	if platform == "" {
		return fmt.Errorf("%w: platform cannot be empty", domain.ErrInvalidInput)
	}
	if !ValidPlatforms[platform] {
		return fmt.Errorf("%w: %s", domain.ErrInvalidPlatform, platform)
	}
	return nil
}

// ValidateUsername validates username from chat platforms
func ValidateUsername(username string) error {
	const MaxUsernameLength = 100
	
	if username == "" {
		return fmt.Errorf("%w: username cannot be empty", domain.ErrInvalidInput)
	}
	
	if len(username) > MaxUsernameLength {
		return fmt.Errorf("%w: username too long", domain.ErrInvalidInput)
	}
	
	// Check for control characters that could cause issues
	if strings.ContainsAny(username, "\x00\n\r\t") {
		return fmt.Errorf("%w: username contains invalid characters", domain.ErrInvalidInput)
	}
	
	return nil
}

// ValidateItemName validates item names
func ValidateItemName(itemName string) error {
	const MaxItemNameLength = 100
	
	if itemName == "" {
		return fmt.Errorf("%w: item name cannot be empty", domain.ErrInvalidInput)
	}
	
	if len(itemName) > MaxItemNameLength {
		return fmt.Errorf("%w: item name too long", domain.ErrInvalidInput)
	}
	
	return nil
}

// ValidateQuantity validates item quantities
func ValidateQuantity(quantity int) error {
	const MinQuantity = 1
	const MaxQuantity = 10000
	
	if quantity < MinQuantity {
		return fmt.Errorf("%w: quantity must be at least 1", domain.ErrInvalidInput)
	}
	
	if quantity > MaxQuantity {
		return fmt.Errorf("%w: quantity exceeds maximum allowed", domain.ErrInvalidInput)
	}
	
	return nil
}
