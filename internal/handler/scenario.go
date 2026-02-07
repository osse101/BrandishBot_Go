package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/scenario"
)

// ScenarioHandler handles scenario API endpoints
type ScenarioHandler struct {
	engine *scenario.Engine
}

// NewScenarioHandler creates a new scenario handler
func NewScenarioHandler(engine *scenario.Engine) *ScenarioHandler {
	return &ScenarioHandler{
		engine: engine,
	}
}

// CapabilitiesResponse is the response for the capabilities endpoint
type CapabilitiesResponse struct {
	Capabilities []scenario.CapabilityInfo `json:"capabilities"`
	Features     []string                  `json:"features"`
}

// ScenariosResponse is the response for the scenarios endpoint
type ScenariosResponse struct {
	Scenarios []scenario.Summary `json:"scenarios"`
	Total     int                `json:"total"`
}

// RunScenarioRequest is the request body for running a scenario
type RunScenarioRequest struct {
	ScenarioID string                 `json:"scenario_id"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// RunCustomScenarioRequest is the request body for running a custom scenario
type RunCustomScenarioRequest struct {
	Scenario   scenario.Scenario      `json:"scenario"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// HandleGetCapabilities returns the available capabilities for UI rendering
// @Summary Get scenario capabilities
// @Description Returns all available scenario capabilities for UI rendering
// @Tags scenario
// @Produce json
// @Success 200 {object} CapabilitiesResponse
// @Router /api/v1/admin/simulate/capabilities [get]
func (h *ScenarioHandler) HandleGetCapabilities() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := h.engine.GetRegistry()

		response := CapabilitiesResponse{
			Capabilities: registry.GetAllCapabilities(),
			Features:     registry.Features(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// HandleGetScenarios returns all pre-built scenarios
// @Summary Get all scenarios
// @Description Returns all pre-built scenarios available for execution
// @Tags scenario
// @Produce json
// @Success 200 {object} ScenariosResponse
// @Router /api/v1/admin/simulate/scenarios [get]
func (h *ScenarioHandler) HandleGetScenarios() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := h.engine.GetRegistry()
		summaries := registry.GetScenarioSummaries()

		response := ScenariosResponse{
			Scenarios: summaries,
			Total:     len(summaries),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// HandleGetScenario returns a specific scenario by ID
// @Summary Get a specific scenario
// @Description Returns details of a specific scenario by ID
// @Tags scenario
// @Produce json
// @Param id query string true "Scenario ID"
// @Success 200 {object} scenario.Scenario
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/admin/simulate/scenario [get]
func (h *ScenarioHandler) HandleGetScenario() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scenarioID := r.URL.Query().Get("id")
		if scenarioID == "" {
			respondError(w, http.StatusBadRequest, "scenario ID required")
			return
		}

		registry := h.engine.GetRegistry()
		s, _, err := registry.GetScenario(scenarioID)
		if err != nil {
			if errors.Is(err, scenario.ErrScenarioNotFound) {
				respondError(w, http.StatusNotFound, "scenario not found")
				return
			}
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s)
	}
}

// HandleRunScenario executes a pre-built scenario
// @Summary Run a scenario
// @Description Executes a pre-built scenario by ID
// @Tags scenario
// @Accept json
// @Produce json
// @Param request body RunScenarioRequest true "Run scenario request"
// @Success 200 {object} scenario.ExecutionResult
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/simulate/run [post]
func (h *ScenarioHandler) HandleRunScenario() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RunScenarioRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.ScenarioID == "" {
			respondError(w, http.StatusBadRequest, "scenario_id required")
			return
		}

		result, err := h.engine.Execute(r.Context(), req.ScenarioID, req.Parameters)
		if err != nil {
			if errors.Is(err, scenario.ErrScenarioNotFound) {
				respondError(w, http.StatusNotFound, "scenario not found")
				return
			}
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// HandleRunCustomScenario executes a custom scenario definition
// @Summary Run a custom scenario
// @Description Executes a custom scenario definition provided in the request
// @Tags scenario
// @Accept json
// @Produce json
// @Param request body RunCustomScenarioRequest true "Custom scenario request"
// @Success 200 {object} scenario.ExecutionResult
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/simulate/run-custom [post]
func (h *ScenarioHandler) HandleRunCustomScenario() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RunCustomScenarioRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Scenario.ID == "" {
			respondError(w, http.StatusBadRequest, "scenario.id required")
			return
		}

		if req.Scenario.Feature == "" {
			respondError(w, http.StatusBadRequest, "scenario.feature required")
			return
		}

		result, err := h.engine.ExecuteCustom(r.Context(), req.Scenario, req.Parameters)
		if err != nil {
			if errors.Is(err, scenario.ErrProviderNotFound) {
				respondError(w, http.StatusNotFound, "provider not found for feature: "+req.Scenario.Feature)
				return
			}
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}
