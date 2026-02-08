package scenario

import (
	"encoding/json"
	"time"
)

// ExecutionResult represents the complete result of a scenario execution
type ExecutionResult struct {
	ScenarioID   string                 `json:"scenario_id"`
	ScenarioName string                 `json:"scenario_name"`
	Success      bool                   `json:"success"`
	DurationMS   int64                  `json:"duration_ms"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  time.Time              `json:"completed_at"`
	Steps        []StepResult           `json:"steps"`
	Error        string                 `json:"error,omitempty"`
	User         *SimulatedUser         `json:"user,omitempty"`
	FinalState   map[string]interface{} `json:"final_state,omitempty"`
}

// StepResult represents the result of a single step execution
type StepResult struct {
	StepName   string                 `json:"step_name"`
	StepIndex  int                    `json:"step_index"`
	Action     ActionType             `json:"action"`
	Success    bool                   `json:"success"`
	DurationMS int64                  `json:"duration_ms"`
	Output     map[string]interface{} `json:"output,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Assertions []AssertionResult      `json:"assertions,omitempty"`
}

// AssertionResult represents the result of a single assertion
type AssertionResult struct {
	Type     AssertionType `json:"type"`
	Path     string        `json:"path"`
	Expected interface{}   `json:"expected,omitempty"`
	Actual   interface{}   `json:"actual,omitempty"`
	Passed   bool          `json:"passed"`
	Reason   string        `json:"reason,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// NewExecutionResult creates a new ExecutionResult with initialized values
func NewExecutionResult(scenarioID, scenarioName string) *ExecutionResult {
	return &ExecutionResult{
		ScenarioID:   scenarioID,
		ScenarioName: scenarioName,
		Success:      true, // Assume success until proven otherwise
		StartedAt:    time.Now(),
		Steps:        make([]StepResult, 0),
		FinalState:   make(map[string]interface{}),
	}
}

// Complete marks the execution as complete and calculates duration
func (r *ExecutionResult) Complete() {
	r.CompletedAt = time.Now()
	r.DurationMS = r.CompletedAt.Sub(r.StartedAt).Milliseconds()
}

// AddStepResult adds a step result and updates overall success
func (r *ExecutionResult) AddStepResult(step StepResult) {
	r.Steps = append(r.Steps, step)
	if !step.Success {
		r.Success = false
	}
}

// SetError marks the execution as failed with an error
func (r *ExecutionResult) SetError(err error) {
	r.Success = false
	r.Error = err.Error()
}

// NewStepResult creates a new StepResult with initialized values
func NewStepResult(stepName string, stepIndex int, action ActionType) *StepResult {
	return &StepResult{
		StepName:   stepName,
		StepIndex:  stepIndex,
		Action:     action,
		Success:    true,
		Output:     make(map[string]interface{}),
		Assertions: make([]AssertionResult, 0),
	}
}

// SetDuration sets the duration from start to now
func (r *StepResult) SetDuration(start time.Time) {
	r.DurationMS = time.Since(start).Milliseconds()
}

// SetError marks the step as failed with an error
func (r *StepResult) SetError(err error) {
	r.Success = false
	r.Error = err.Error()
}

// AddOutput adds a key-value pair to the output
func (r *StepResult) AddOutput(key string, value interface{}) {
	r.Output[key] = value
}

// AddAssertionResult adds an assertion result and updates step success
func (r *StepResult) AddAssertionResult(assertion AssertionResult) {
	r.Assertions = append(r.Assertions, assertion)
	if !assertion.Passed {
		r.Success = false
	}
}

// ToJSON converts the result to JSON bytes
func (r *ExecutionResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// ToPrettyJSON converts the result to indented JSON bytes
func (r *ExecutionResult) ToPrettyJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// GetStepByName finds a step result by name
func (r *ExecutionResult) GetStepByName(name string) *StepResult {
	for i := range r.Steps {
		if r.Steps[i].StepName == name {
			return &r.Steps[i]
		}
	}
	return nil
}

// TotalAssertions returns the total number of assertions across all steps
func (r *ExecutionResult) TotalAssertions() int {
	total := 0
	for _, step := range r.Steps {
		total += len(step.Assertions)
	}
	return total
}

// PassedAssertions returns the number of passed assertions
func (r *ExecutionResult) PassedAssertions() int {
	passed := 0
	for _, step := range r.Steps {
		for _, assertion := range step.Assertions {
			if assertion.Passed {
				passed++
			}
		}
	}
	return passed
}

// FailedAssertions returns the number of failed assertions
func (r *ExecutionResult) FailedAssertions() int {
	return r.TotalAssertions() - r.PassedAssertions()
}

// Summary returns a brief summary of the execution
type ExecutionSummary struct {
	ScenarioID       string `json:"scenario_id"`
	ScenarioName     string `json:"scenario_name"`
	Success          bool   `json:"success"`
	DurationMS       int64  `json:"duration_ms"`
	TotalSteps       int    `json:"total_steps"`
	PassedSteps      int    `json:"passed_steps"`
	TotalAssertions  int    `json:"total_assertions"`
	PassedAssertions int    `json:"passed_assertions"`
}

// ToSummary converts the result to a summary
func (r *ExecutionResult) ToSummary() ExecutionSummary {
	passedSteps := 0
	for _, step := range r.Steps {
		if step.Success {
			passedSteps++
		}
	}

	return ExecutionSummary{
		ScenarioID:       r.ScenarioID,
		ScenarioName:     r.ScenarioName,
		Success:          r.Success,
		DurationMS:       r.DurationMS,
		TotalSteps:       len(r.Steps),
		PassedSteps:      passedSteps,
		TotalAssertions:  r.TotalAssertions(),
		PassedAssertions: r.PassedAssertions(),
	}
}
