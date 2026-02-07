package providers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/harvest"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/scenario"
	"github.com/osse101/BrandishBot_Go/internal/scenario/capabilities"
)

const (
	FeatureHarvest = "harvest"

	// Harvest-specific parameters
	ParamUserID   = "user_id"
	ParamUsername = "username"
	ParamPlatform = "platform"
	ParamHours    = "hours"

	// Default test user values
	DefaultPlatform   = "discord"
	DefaultPlatformID = "scenario_test_user"
)

// HarvestProvider implements the scenario.Provider interface for the harvest feature
type HarvestProvider struct {
	db          *pgxpool.Pool
	harvestSvc  harvest.Service
	harvestRepo repository.HarvestRepository
	userRepo    repository.User
}

// NewHarvestProvider creates a new harvest scenario provider
func NewHarvestProvider(
	db *pgxpool.Pool,
	harvestSvc harvest.Service,
	harvestRepo repository.HarvestRepository,
	userRepo repository.User,
) *HarvestProvider {
	return &HarvestProvider{
		db:          db,
		harvestSvc:  harvestSvc,
		harvestRepo: harvestRepo,
		userRepo:    userRepo,
	}
}

// Feature returns the feature name
func (p *HarvestProvider) Feature() string {
	return FeatureHarvest
}

// Capabilities returns the capabilities this provider supports
func (p *HarvestProvider) Capabilities() []scenario.CapabilityType {
	return []scenario.CapabilityType{
		scenario.CapabilityTimeWarp,
	}
}

// GetCapabilityInfo returns detailed capability information
func (p *HarvestProvider) GetCapabilityInfo() []scenario.CapabilityInfo {
	return []scenario.CapabilityInfo{
		capabilities.TimeWarpCapabilityInfo(),
	}
}

// SupportsAction returns true if the provider supports the given action
func (p *HarvestProvider) SupportsAction(action scenario.ActionType) bool {
	switch action {
	case scenario.ActionSetState,
		scenario.ActionTimeWarp,
		scenario.ActionExecuteHarvest:
		return true
	default:
		return false
	}
}

// PrebuiltScenarios returns the list of pre-built harvest scenarios
func (p *HarvestProvider) PrebuiltScenarios() []scenario.Scenario {
	return []scenario.Scenario{
		p.patientFarmerScenario(),
		p.spoiledHarvestScenario(),
		p.firstTimeFarmerScenario(),
		p.quickHarvestScenario(),
	}
}

// ExecuteStep executes a single step
func (p *HarvestProvider) ExecuteStep(ctx context.Context, step scenario.Step, state *scenario.ExecutionState) (*scenario.StepResult, error) {
	result := scenario.NewStepResult(step.Name, 0, step.Action)

	switch step.Action {
	case scenario.ActionSetState:
		return p.executeSetState(ctx, step, state, result)
	case scenario.ActionTimeWarp:
		return p.executeTimeWarp(ctx, step, state, result)
	case scenario.ActionExecuteHarvest:
		return p.executeHarvest(ctx, step, state, result)
	default:
		return nil, fmt.Errorf("%w: %s", scenario.ErrInvalidAction, step.Action)
	}
}

// executeSetState initializes or sets the harvest state
func (p *HarvestProvider) executeSetState(ctx context.Context, step scenario.Step, state *scenario.ExecutionState, result *scenario.StepResult) (*scenario.StepResult, error) {
	// Extract parameters
	username := getStringParam(step.Parameters, ParamUsername, "scenario_test_user")
	platform := getStringParam(step.Parameters, ParamPlatform, DefaultPlatform)
	platformID := getStringParam(step.Parameters, "platform_id", DefaultPlatformID)

	// Get or create the user
	user, err := p.userRepo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Create the user
			newUser := &domain.User{
				Username:  username,
				DiscordID: platformID, // Assuming discord for now
			}
			if err := p.userRepo.UpsertUser(ctx, newUser); err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}
			user, err = p.userRepo.GetUserByPlatformID(ctx, platform, platformID)
			if err != nil {
				return nil, fmt.Errorf("failed to get created user: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
	}

	// Store user in state
	state.User = &scenario.SimulatedUser{
		UserID:     user.ID,
		Username:   user.Username,
		Platform:   platform,
		PlatformID: platformID,
	}

	// Initialize harvest state if needed
	harvestState, err := p.harvestRepo.GetHarvestState(ctx, user.ID)
	if err != nil {
		if errors.Is(err, domain.ErrHarvestStateNotFound) {
			harvestState, err = p.harvestRepo.CreateHarvestState(ctx, user.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to create harvest state: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get harvest state: %w", err)
		}
	}

	result.AddOutput("user_id", user.ID)
	result.AddOutput("username", user.Username)
	result.AddOutput("harvest_state_initialized", true)
	result.AddOutput("last_harvested_at", harvestState.LastHarvestedAt.Format(time.RFC3339))

	return result, nil
}

// executeTimeWarp manipulates the harvest timestamp directly in the database
func (p *HarvestProvider) executeTimeWarp(ctx context.Context, step scenario.Step, state *scenario.ExecutionState, result *scenario.StepResult) (*scenario.StepResult, error) {
	// Ensure user is initialized
	if state.User == nil {
		return nil, scenario.ErrUserNotInitialized
	}

	// Parse time warp parameters
	params, err := capabilities.ParseTimeWarpParams(step.Parameters)
	if err != nil {
		return nil, err
	}

	// Calculate the new timestamp
	// We warp backward in time by setting last_harvested_at to (now - hours)
	// This makes it look like the user harvested `hours` ago
	newTimestamp := time.Now().Add(-time.Duration(params.Hours * float64(time.Hour)))

	// Update database directly
	userUUID, err := uuid.Parse(state.User.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	query := `UPDATE harvest_state SET last_harvested_at = $1, updated_at = NOW() WHERE user_id = $2`
	_, err = p.db.Exec(ctx, query, newTimestamp, userUUID)
	if err != nil {
		return nil, scenario.WrapDatabaseError("update harvest state", err)
	}

	// Update state
	state.SimulatedNow = time.Now()
	state.SetResult("warped_hours", params.Hours)
	state.SetResult("last_harvested_at", newTimestamp.Format(time.RFC3339))

	result.AddOutput("warped_hours", params.Hours)
	result.AddOutput("new_last_harvested_at", newTimestamp.Format(time.RFC3339))
	result.AddOutput("simulated_elapsed_hours", params.Hours)

	return result, nil
}

// executeHarvest runs the actual harvest operation
func (p *HarvestProvider) executeHarvest(ctx context.Context, _ scenario.Step, state *scenario.ExecutionState, result *scenario.StepResult) (*scenario.StepResult, error) {
	// Ensure user is initialized
	if state.User == nil {
		return nil, scenario.ErrUserNotInitialized
	}

	// Execute harvest using the real service
	harvestResult, err := p.harvestSvc.Harvest(ctx, state.User.Platform, state.User.PlatformID, state.User.Username)
	if err != nil {
		result.AddOutput("error", err.Error())
		result.SetError(err)
		return result, nil // Return result with error info, not nil
	}

	// Map results
	result.AddOutput("items_gained", harvestResult.ItemsGained)
	result.AddOutput("hours_since_harvest", harvestResult.HoursSinceHarvest)
	result.AddOutput("next_harvest_at", harvestResult.NextHarvestAt.Format(time.RFC3339))
	result.AddOutput("message", harvestResult.Message)

	// Store in state for later assertions
	state.SetResult("harvest_result", harvestResult)
	state.SetResult("items_gained", harvestResult.ItemsGained)

	return result, nil
}

// Pre-built scenario definitions

func (p *HarvestProvider) patientFarmerScenario() scenario.Scenario {
	return p.baseHarvestScenario(
		"harvest_patient_farmer",
		"The Patient Farmer",
		"Tests maximum tier harvest (168 hours). Sets up a user, warps time to simulate 168h wait, then executes harvest.",
		"patient_farmer",
		"scenario_patient_farmer",
		168.0,
		[]scenario.Assertion{
			{
				Type: scenario.AssertNotEmpty,
				Path: "output.items_gained",
			},
			{
				Type:   scenario.AssertGreaterThan,
				Path:   "output.hours_since_harvest",
				Value:  167.0,
				Reason: "Should show approximately 168 hours elapsed",
			},
		},
	)
}

func (p *HarvestProvider) spoiledHarvestScenario() scenario.Scenario {
	return scenario.Scenario{
		ID:          "harvest_spoiled",
		Name:        "Spoiled Harvest",
		Description: "Tests harvest spoilage after 336 hours (2 weeks). Verifies reduced rewards.",
		Feature:     FeatureHarvest,
		Capabilities: []scenario.CapabilityType{
			scenario.CapabilityTimeWarp,
		},
		Steps: []scenario.Step{
			{
				Name:        "initialize",
				Description: "Set up user and harvest state",
				Action:      scenario.ActionSetState,
				Parameters: map[string]interface{}{
					ParamUsername: "neglectful_farmer",
					ParamPlatform: DefaultPlatform,
					"platform_id": "scenario_neglectful_farmer",
				},
				Assertions: []scenario.Assertion{
					{
						Type: scenario.AssertNotEmpty,
						Path: "output.user_id",
					},
				},
			},
			{
				Name:        "time_warp",
				Description: "Warp time to simulate 340 hours (spoiled threshold)",
				Action:      scenario.ActionTimeWarp,
				Parameters: map[string]interface{}{
					ParamHours: 340.0,
				},
				Assertions: []scenario.Assertion{
					{
						Type:  scenario.AssertEquals,
						Path:  "output.warped_hours",
						Value: 340.0,
					},
				},
			},
			{
				Name:        "execute_harvest",
				Description: "Execute the harvest and verify spoiled results",
				Action:      scenario.ActionExecuteHarvest,
				Parameters:  map[string]interface{}{},
				Assertions: []scenario.Assertion{
					{
						Type:   scenario.AssertContains,
						Path:   "output.message",
						Value:  "spoiled",
						Reason: "Message should indicate spoilage",
					},
				},
			},
		},
	}
}

func (p *HarvestProvider) firstTimeFarmerScenario() scenario.Scenario {
	return scenario.Scenario{
		ID:          "harvest_first_time",
		Name:        "First Time Farmer",
		Description: "Tests first harvest initialization. New user should get initialization message, not rewards.",
		Feature:     FeatureHarvest,
		Capabilities: []scenario.CapabilityType{
			scenario.CapabilityTimeWarp,
		},
		Steps: []scenario.Step{
			{
				Name:        "initialize",
				Description: "Set up new user with fresh harvest state",
				Action:      scenario.ActionSetState,
				Parameters: map[string]interface{}{
					ParamUsername: "new_farmer",
					ParamPlatform: DefaultPlatform,
					"platform_id": fmt.Sprintf("scenario_new_farmer_%d", time.Now().UnixNano()),
				},
				Assertions: []scenario.Assertion{
					{
						Type: scenario.AssertNotEmpty,
						Path: "output.user_id",
					},
				},
			},
			{
				Name:        "execute_harvest",
				Description: "Attempt harvest immediately (should fail - too soon)",
				Action:      scenario.ActionExecuteHarvest,
				Parameters:  map[string]interface{}{},
				Assertions: []scenario.Assertion{
					{
						Type:   scenario.AssertContains,
						Path:   "output.error",
						Value:  "too soon",
						Reason: "Should indicate harvest is too soon",
					},
				},
			},
		},
	}
}

func (p *HarvestProvider) quickHarvestScenario() scenario.Scenario {
	return p.baseHarvestScenario(
		"harvest_quick",
		"Quick Harvest",
		"Tests minimum harvest time (1 hour). Verifies basic tier rewards.",
		"quick_farmer",
		"scenario_quick_farmer",
		2.0,
		[]scenario.Assertion{
			{
				Type: scenario.AssertNotEmpty,
				Path: "output.items_gained",
			},
			{
				Type:   scenario.AssertGreaterThan,
				Path:   "output.hours_since_harvest",
				Value:  1.0,
				Reason: "Should show at least 1 hour elapsed",
			},
		},
	)
}

func (p *HarvestProvider) baseHarvestScenario(id, name, description, username, platformID string, warpHours float64, harvestAssertions []scenario.Assertion) scenario.Scenario {
	return scenario.Scenario{
		ID:          id,
		Name:        name,
		Description: description,
		Feature:     FeatureHarvest,
		Capabilities: []scenario.CapabilityType{
			scenario.CapabilityTimeWarp,
		},
		Steps: []scenario.Step{
			{
				Name:        "initialize",
				Description: "Set up user and harvest state",
				Action:      scenario.ActionSetState,
				Parameters: map[string]interface{}{
					ParamUsername: username,
					ParamPlatform: DefaultPlatform,
					"platform_id": platformID,
				},
				Assertions: []scenario.Assertion{
					{
						Type: scenario.AssertNotEmpty,
						Path: "output.user_id",
					},
				},
			},
			{
				Name:        "time_warp",
				Description: fmt.Sprintf("Warp time to simulate %.0f hours since last harvest", warpHours),
				Action:      scenario.ActionTimeWarp,
				Parameters: map[string]interface{}{
					ParamHours: warpHours,
				},
				Assertions: []scenario.Assertion{
					{
						Type:  scenario.AssertEquals,
						Path:  "output.warped_hours",
						Value: warpHours,
					},
				},
			},
			{
				Name:        "execute_harvest",
				Description: "Execute the harvest and verify rewards",
				Action:      scenario.ActionExecuteHarvest,
				Parameters:  map[string]interface{}{},
				Assertions:  harvestAssertions,
			},
		},
	}
}

// Helper functions

func getStringParam(params map[string]interface{}, key, defaultVal string) string {
	if val, ok := params[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return defaultVal
}
