package capabilities

import (
	"github.com/osse101/BrandishBot_Go/internal/scenario"
)

// MultiUserCapabilityInfo returns the capability info for multi-user simulation
// This is a stub for future implementation
func MultiUserCapabilityInfo() scenario.CapabilityInfo {
	return scenario.CapabilityInfo{
		Type:        scenario.CapabilityMultiUser,
		Name:        "Multi-User Simulation",
		Description: "Allows simulation of multiple users interacting with features (e.g., gamble sessions)",
		Actions: []scenario.ActionInfo{
			{
				Action:      "create_user",
				Name:        "Create User",
				Description: "Creates a simulated user for the scenario",
				Parameters: []scenario.ParameterInfo{
					{
						Name:        "username",
						Type:        "string",
						Required:    true,
						Description: "Username for the simulated user",
					},
					{
						Name:        "platform",
						Type:        "string",
						Required:    false,
						Description: "Platform for the user (default: 'discord')",
					},
				},
				Example: map[string]interface{}{
					"username": "test_user_1",
					"platform": "discord",
				},
			},
			{
				Action:      "switch_user",
				Name:        "Switch User",
				Description: "Switches the active user context for subsequent actions",
				Parameters: []scenario.ParameterInfo{
					{
						Name:        "user_index",
						Type:        "number",
						Required:    true,
						Description: "Index of the user to switch to (0-based)",
					},
				},
				Example: map[string]interface{}{
					"user_index": 1,
				},
			},
		},
	}
}

// MultiUserParams represents parameters for multi-user actions
type MultiUserParams struct {
	Username  string
	Platform  string
	UserIndex int
}

// ParseMultiUserParams extracts multi-user parameters from a step
func ParseMultiUserParams(params map[string]interface{}) (*MultiUserParams, error) {
	result := &MultiUserParams{
		Platform: "discord", // Default
	}

	// Username
	if username, ok := params["username"]; ok {
		if u, ok := username.(string); ok {
			result.Username = u
		} else {
			return nil, scenario.NewParameterError("username", "must be a string")
		}
	}

	// Platform
	if platform, ok := params["platform"]; ok {
		if p, ok := platform.(string); ok {
			result.Platform = p
		}
	}

	// User index
	if userIndex, ok := params["user_index"]; ok {
		switch ui := userIndex.(type) {
		case float64:
			result.UserIndex = int(ui)
		case int:
			result.UserIndex = ui
		case int64:
			result.UserIndex = int(ui)
		default:
			return nil, scenario.NewParameterError("user_index", "must be a number")
		}
	}

	return result, nil
}
