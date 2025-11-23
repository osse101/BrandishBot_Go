package handler

import (
	"errors"
	"strings"
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
		return errors.New("platform cannot be empty")
	}
	if !ValidPlatforms[platform] {
		return errors.New("unsupported platform")
	}
	return nil
}

// ValidateUsername validates username from chat platforms
func ValidateUsername(username string) error {
	const MaxUsernameLength = 100
	
	if username == "" {
		return errors.New("username cannot be empty")
	}
	
	if len(username) > MaxUsernameLength {
		return errors.New("username too long")
	}
	
	// Check for control characters that could cause issues
	if strings.ContainsAny(username, "\x00\n\r\t") {
		return errors.New("username contains invalid characters")
	}
	
	return nil
}

// ValidateItemName validates item names
func ValidateItemName(itemName string) error {
	const MaxItemNameLength = 100
	
	if itemName == "" {
		return errors.New("item name cannot be empty")
	}
	
	if len(itemName) > MaxItemNameLength {
		return errors.New("item name too long")
	}
	
	return nil
}

// ValidateQuantity validates item quantities
func ValidateQuantity(quantity int) error {
	const MinQuantity = 1
	const MaxQuantity = 10000
	
	if quantity < MinQuantity {
		return errors.New("quantity must be at least 1")
	}
	
	if quantity > MaxQuantity {
		return errors.New("quantity exceeds maximum allowed")
	}
	
	return nil
}
