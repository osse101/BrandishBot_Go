package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
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
		http.Error(w, ErrMsgInvalidRequest, http.StatusBadRequest)
		return err
	}

	// Log the decoded request at debug level
	log.Debug(fmt.Sprintf("%s request decoded", actionName))

	// Validate the request struct
	if err := GetValidator().ValidateStruct(req); err != nil {
		validationErrs := FormatValidationError(err)
		respondJSON(w, http.StatusBadRequest, ValidationErrorResponse{
			Error:  ErrMsgInvalidRequestSummary,
			Fields: validationErrs,
		})
		return err
	}

	return nil
}

// ValidationErrorResponse defines the response structure for validation errors
type ValidationErrorResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields"`
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
		http.Error(w, fmt.Sprintf(ErrMsgMissingQueryParam, paramName), http.StatusBadRequest)
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

// handleFeatureAction is a generic helper for handlers that perform an action protected by a feature flag.
// It handles feature checking, request decoding/validation, service call, error handling, and JSON response.
func handleFeatureAction[REQ any, RES any](
	w http.ResponseWriter,
	r *http.Request,
	progSvc progression.Service,
	featureKey string,
	opName string,
	action func(context.Context, REQ) (RES, error),
	responseFactory func(RES) interface{},
) {
	if CheckFeatureLocked(w, r, progSvc, featureKey) {
		return
	}

	var req REQ
	if err := DecodeAndValidateRequest(r, w, &req, opName); err != nil {
		return
	}

	res, err := action(r.Context(), req)
	if err != nil {
		respondServiceError(w, r, opName, err)
		return
	}

	respondJSON(w, http.StatusCreated, responseFactory(res))
}
