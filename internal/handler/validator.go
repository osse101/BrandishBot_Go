package handler

import (
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
