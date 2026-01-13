package handler

import (
	"errors"
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

// FormatValidationError formats validation errors into a user-friendly map
// This prevents leaking internal struct names and provides cleaner error messages
func FormatValidationError(err error) map[string]string {
	if err == nil {
		return nil
	}

	errs := make(map[string]string)

	// Check if it's a validator.ValidationErrors
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		errs["error"] = "Invalid request format"
		return errs
	}

	for _, e := range validationErrors {
		field := strings.ToLower(e.Field())
		switch e.Tag() {
		case "required":
			errs[field] = "This field is required"
		case "email":
			errs[field] = "Invalid email format"
		case "platform":
			errs[field] = "Invalid platform"
		case "max":
			errs[field] = fmt.Sprintf("Must be at most %s characters", e.Param())
		case "min":
			errs[field] = fmt.Sprintf("Must be at least %s characters", e.Param())
		case "excludesall":
			errs[field] = "Contains invalid characters"
		default:
			errs[field] = "Invalid value"
		}
	}

	return errs
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
