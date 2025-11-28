package handler

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps the validator instance
type Validator struct {
	validate *validator.Validate
}

// Global validator instance
var validate *Validator

// InitValidator initializes the global validator
func InitValidator() {
	v := validator.New()
	
	// Register custom validation for platform
	_ = v.RegisterValidation("platform", validatePlatform)
	
	validate = &Validator{validate: v}
}

// GetValidator returns the global validator instance
func GetValidator() *Validator {
	if validate == nil {
		InitValidator()
	}
	return validate
}

// ValidateStruct validates a struct using tags
func (v *Validator) ValidateStruct(s interface{}) error {
	return v.validate.Struct(s)
}

// ValidPlatforms defines supported platforms
var ValidPlatforms = map[string]bool{
	"twitch":  true,
	"youtube": true,
	"discord": true,
}

// Custom validation function for platform
func validatePlatform(fl validator.FieldLevel) bool {
	platform := fl.Field().String()
	// Allow empty if not required (handled by 'required' tag if needed)
	if platform == "" {
		return true
	}
	return ValidPlatforms[strings.ToLower(platform)]
}

// Legacy validation functions (deprecated, use struct tags instead)

// ValidatePlatform checks if the platform is supported
// Deprecated: Use struct tag `validate:"platform"` instead
func ValidatePlatform(platform string) error {
	if platform == "" {
		return fmt.Errorf("platform cannot be empty")
	}
	if !ValidPlatforms[strings.ToLower(platform)] {
		return fmt.Errorf("invalid platform: %s", platform)
	}
	return nil
}

// ValidateUsername validates username from chat platforms
// Deprecated: Use struct tag `validate:"required,max=100,excludesall=\x00\n\r\t"` instead
func ValidateUsername(username string) error {
	const MaxUsernameLength = 100

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if len(username) > MaxUsernameLength {
		return fmt.Errorf("username too long")
	}

	// Check for control characters that could cause issues
	if strings.ContainsAny(username, "\x00\n\r\t") {
		return fmt.Errorf("username contains invalid characters")
	}

	return nil
}

// ValidateItemName validates item names
// Deprecated: Use struct tag `validate:"required,max=100"` instead
func ValidateItemName(itemName string) error {
	const MaxItemNameLength = 100

	if itemName == "" {
		return fmt.Errorf("item name cannot be empty")
	}

	if len(itemName) > MaxItemNameLength {
		return fmt.Errorf("item name too long")
	}

	return nil
}

// ValidateQuantity validates item quantities
// Deprecated: Use struct tag `validate:"min=1,max=10000"` instead
func ValidateQuantity(quantity int) error {
	const MinQuantity = 1
	const MaxQuantity = 10000

	if quantity < MinQuantity {
		return fmt.Errorf("quantity must be at least 1")
	}

	if quantity > MaxQuantity {
		return fmt.Errorf("quantity exceeds maximum allowed")
	}

	return nil
}
