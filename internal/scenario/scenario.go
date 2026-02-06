package scenario

import (
	"time"
)

// CapabilityType defines the type of capability a provider supports
type CapabilityType string

const (
	// CapabilityTimeWarp indicates the provider supports time manipulation
	CapabilityTimeWarp CapabilityType = "time_warp"
	// CapabilityEventInjector indicates the provider supports event injection
	CapabilityEventInjector CapabilityType = "event_injector"
	// CapabilityMultiUser indicates the provider supports multi-user simulation
	CapabilityMultiUser CapabilityType = "multi_user"
)

// ActionType defines the type of action in a scenario step
type ActionType string

const (
	// Common actions
	ActionSetState    ActionType = "set_state"
	ActionTimeWarp    ActionType = "time_warp"
	ActionInjectEvent ActionType = "inject_event"
	ActionAssert      ActionType = "assert"

	// Harvest-specific actions
	ActionExecuteHarvest ActionType = "execute_harvest"

	// Quest-specific actions
	ActionInjectQuest   ActionType = "inject_quest"
	ActionExecuteQuest  ActionType = "execute_quest"
	ActionTriggerSearch ActionType = "trigger_search"
	ActionClaimReward   ActionType = "claim_reward"
)

// AssertionType defines the type of assertion
type AssertionType string

const (
	AssertEquals        AssertionType = "equals"
	AssertGreaterThan   AssertionType = "greater_than"
	AssertLessThan      AssertionType = "less_than"
	AssertContains      AssertionType = "contains"
	AssertNotEmpty      AssertionType = "not_empty"
	AssertEmpty         AssertionType = "empty"
	AssertTrue          AssertionType = "true"
	AssertFalse         AssertionType = "false"
	AssertBetween       AssertionType = "between"
	AssertErrorContains AssertionType = "error_contains"
)

// Scenario defines a complete test scenario
type Scenario struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Feature      string           `json:"feature"`
	Capabilities []CapabilityType `json:"capabilities"`
	Steps        []Step           `json:"steps"`
}

// Step defines a single step within a scenario
type Step struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Action      ActionType             `json:"action"`
	Parameters  map[string]interface{} `json:"parameters"`
	Assertions  []Assertion            `json:"assertions"`
}

// Assertion defines an expected outcome for a step
type Assertion struct {
	Type   AssertionType `json:"type"`
	Path   string        `json:"path"` // JSONPath-like path to the value to check
	Value  interface{}   `json:"value,omitempty"`
	Min    interface{}   `json:"min,omitempty"`    // For between assertions
	Max    interface{}   `json:"max,omitempty"`    // For between assertions
	Reason string        `json:"reason,omitempty"` // Human-readable reason for the assertion
}

// SimulatedUser represents a user in scenario execution
type SimulatedUser struct {
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
}

// ExecutionState holds the state during scenario execution
type ExecutionState struct {
	Clock        Clock
	SimulatedNow time.Time
	User         *SimulatedUser
	Results      map[string]interface{}
	Errors       []error
}

// NewExecutionState creates a new execution state with default values
func NewExecutionState() *ExecutionState {
	return &ExecutionState{
		Clock:        NewRealClock(),
		SimulatedNow: time.Now(),
		Results:      make(map[string]interface{}),
		Errors:       make([]error, 0),
	}
}

// SetResult stores a result under the given key
func (s *ExecutionState) SetResult(key string, value interface{}) {
	s.Results[key] = value
}

// GetResult retrieves a result by key
func (s *ExecutionState) GetResult(key string) (interface{}, bool) {
	v, ok := s.Results[key]
	return v, ok
}

// AddError appends an error to the state
func (s *ExecutionState) AddError(err error) {
	s.Errors = append(s.Errors, err)
}

// HasErrors returns true if there are any errors
func (s *ExecutionState) HasErrors() bool {
	return len(s.Errors) > 0
}

// CapabilityInfo provides metadata about a capability for API responses
type CapabilityInfo struct {
	Type        CapabilityType `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Actions     []ActionInfo   `json:"actions"`
}

// ActionInfo provides metadata about an action for API responses
type ActionInfo struct {
	Action      ActionType             `json:"action"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  []ParameterInfo        `json:"parameters"`
	Example     map[string]interface{} `json:"example,omitempty"`
}

// ParameterInfo describes a parameter for an action
type ParameterInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// ScenarioSummary provides a brief overview of a scenario for listing
type ScenarioSummary struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Feature      string           `json:"feature"`
	Capabilities []CapabilityType `json:"capabilities"`
	StepCount    int              `json:"step_count"`
}

// ToSummary converts a Scenario to a ScenarioSummary
func (s *Scenario) ToSummary() ScenarioSummary {
	return ScenarioSummary{
		ID:           s.ID,
		Name:         s.Name,
		Description:  s.Description,
		Feature:      s.Feature,
		Capabilities: s.Capabilities,
		StepCount:    len(s.Steps),
	}
}
