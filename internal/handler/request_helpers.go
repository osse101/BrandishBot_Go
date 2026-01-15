package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// DecodeAndValidateRequest decodes a JSON request body, validates it, and returns appropriate errors.
// It logs the operation and returns a standardized error response to the client.
//
// Parameters:
//   - r: The HTTP request containing the JSON body
//   - w: The HTTP response writer to send error responses
//   - req: Pointer to the request struct to decode into (must implement validation tags)
//   - actionName: Human-readable name for the action (e.g., "Add item", "Upgrade item")
//
// Returns:
//   - error: nil if successful, error if decoding or validation failed
//
// If this function returns an error, the HTTP response has already been written and the handler should return.
//
// Example usage:
//
//	var req AddItemRequest
//	if err := DecodeAndValidateRequest(r, w, &req, "Add item"); err != nil {
//	    return
//	}
func DecodeAndValidateRequest(r *http.Request, w http.ResponseWriter, req interface{}, actionName string) error {
	log := logger.FromContext(r.Context())

	// Decode JSON body
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		log.Error(fmt.Sprintf("Failed to decode %s request", actionName), "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return err
	}

	// Log the decoded request at debug level
	log.Debug(fmt.Sprintf("%s request decoded", actionName))

	// Validate the request struct
	if err := GetValidator().ValidateStruct(req); err != nil {
		log.Warn("Invalid request", "error", err)
		validationErrs := FormatValidationError(err)
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":  "Invalid request",
			"fields": validationErrs,
		})
		return err
	}

	return nil
}

// GetQueryParam retrieves and validates a required query parameter from the request.
// If the parameter is missing or empty, it writes an error response and returns false.
//
// Parameters:
//   - r: The HTTP request to extract the query parameter from
//   - w: The HTTP response writer to send error responses
//   - paramName: The name of the query parameter to retrieve
//
// Returns:
//   - value: The parameter value if present
//   - ok: true if the parameter was found and non-empty, false otherwise
//
// If ok is false, the HTTP response has already been written and the handler should return.
//
// Example usage:
//
//	username, ok := GetQueryParam(r, w, "username")
//	if !ok {
//	    return
//	}
func GetQueryParam(r *http.Request, w http.ResponseWriter, paramName string) (string, bool) {
	log := logger.FromContext(r.Context())
	value := r.URL.Query().Get(paramName)
	if value == "" {
		log.Warn(fmt.Sprintf("Missing %s query parameter", paramName))
		http.Error(w, fmt.Sprintf("Missing %s query parameter", paramName), http.StatusBadRequest)
		return "", false
	}
	return value, true
}

// GetOptionalQueryParam retrieves an optional query parameter from the request.
// Unlike GetQueryParam, this does not write an error response if the parameter is missing.
//
// Parameters:
//   - r: The HTTP request to extract the query parameter from
//   - paramName: The name of the query parameter to retrieve
//   - defaultValue: The default value to return if the parameter is missing
//
// Returns:
//   - value: The parameter value if present, otherwise the defaultValue
//
// Example usage:
//
//	limit := GetOptionalQueryParam(r, "limit", "10")
func GetOptionalQueryParam(r *http.Request, paramName string, defaultValue string) string {
	value := r.URL.Query().Get(paramName)
	if value == "" {
		return defaultValue
	}
	return value
}

// LogRequestFields is a helper to log common request fields in a structured way.
// This provides consistency across handlers when logging request details.
//
// Example usage:
//
//	LogRequestFields(log, "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)
func LogRequestFields(log *slog.Logger, keyvals ...interface{}) {
	if len(keyvals)%2 != 0 {
		log.Warn("LogRequestFields called with odd number of arguments")
		return
	}
	log.Debug("Request details", keyvals...)
}
