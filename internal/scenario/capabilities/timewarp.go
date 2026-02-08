package capabilities

import (
	"github.com/osse101/BrandishBot_Go/internal/scenario"
)

// TimeWarpCapabilityInfo returns the capability info for time warping
func TimeWarpCapabilityInfo() scenario.CapabilityInfo {
	return scenario.CapabilityInfo{
		Type:        scenario.CapabilityTimeWarp,
		Name:        "Time Warp",
		Description: "Allows manipulation of time-dependent features by adjusting database timestamps",
		Actions: []scenario.ActionInfo{
			{
				Action:      scenario.ActionTimeWarp,
				Name:        "Time Warp",
				Description: "Advances or rewinds simulated time by modifying database timestamps",
				Parameters: []scenario.ParameterInfo{
					{
						Name:        "hours",
						Type:        "number",
						Required:    true,
						Description: "Number of hours to warp (positive = forward, negative = backward)",
					},
					{
						Name:        "target",
						Type:        "string",
						Required:    false,
						Description: "Specific field to warp (e.g., 'last_harvested_at'). If omitted, uses feature default.",
					},
				},
				Example: map[string]interface{}{
					"hours":  168,
					"target": "last_harvested_at",
				},
			},
		},
	}
}

// TimeWarpParams represents parameters for a time warp action
type TimeWarpParams struct {
	Hours  float64
	Target string
}

// ParseTimeWarpParams extracts time warp parameters from a step
func ParseTimeWarpParams(params map[string]interface{}) (*TimeWarpParams, error) {
	result := &TimeWarpParams{}

	// Hours (required)
	if hours, ok := params["hours"]; ok {
		switch h := hours.(type) {
		case float64:
			result.Hours = h
		case int:
			result.Hours = float64(h)
		case int64:
			result.Hours = float64(h)
		default:
			return nil, scenario.NewParameterError("hours", "must be a number")
		}
	} else {
		return nil, scenario.NewParameterError("hours", "is required")
	}

	// Target (optional)
	if target, ok := params["target"]; ok {
		if t, ok := target.(string); ok {
			result.Target = t
		}
	}

	return result, nil
}
