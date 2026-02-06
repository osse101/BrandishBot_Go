package scenario

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Engine executes scenarios using registered providers
type Engine struct {
	registry *Registry
}

// NewEngine creates a new scenario execution engine
func NewEngine(registry *Registry) *Engine {
	return &Engine{
		registry: registry,
	}
}

// Execute runs a pre-built scenario by ID
func (e *Engine) Execute(ctx context.Context, scenarioID string, params map[string]interface{}) (*ExecutionResult, error) {
	scenario, provider, err := e.registry.GetScenario(scenarioID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrScenarioNotFound, scenarioID)
	}

	return e.ExecuteScenario(ctx, *scenario, provider, params)
}

// ExecuteScenario runs a scenario with the given provider
func (e *Engine) ExecuteScenario(ctx context.Context, scenario Scenario, provider Provider, params map[string]interface{}) (*ExecutionResult, error) {
	result := NewExecutionResult(scenario.ID, scenario.Name)
	state := NewExecutionState()

	// Apply any provided parameters to state
	if params != nil {
		for k, v := range params {
			state.SetResult(k, v)
		}
	}

	// Execute each step
	for i, step := range scenario.Steps {
		select {
		case <-ctx.Done():
			result.SetError(ctx.Err())
			result.Complete()
			return result, ctx.Err()
		default:
		}

		stepResult := e.executeStep(ctx, step, i, provider, state)
		result.AddStepResult(*stepResult)

		// Stop on step failure
		if !stepResult.Success {
			break
		}
	}

	// Copy final state to result
	result.FinalState = state.Results
	result.User = state.User
	result.Complete()

	return result, nil
}

// ExecuteCustom runs a custom scenario definition
func (e *Engine) ExecuteCustom(ctx context.Context, scenario Scenario, params map[string]interface{}) (*ExecutionResult, error) {
	// Find the provider for this feature
	provider, ok := e.registry.Get(scenario.Feature)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, scenario.Feature)
	}

	return e.ExecuteScenario(ctx, scenario, provider, params)
}

// executeStep executes a single step and handles assertions
func (e *Engine) executeStep(ctx context.Context, step Step, index int, provider Provider, state *ExecutionState) *StepResult {
	stepStart := time.Now()
	stepResult := NewStepResult(step.Name, index, step.Action)

	// Check if provider supports this action
	if !provider.SupportsAction(step.Action) {
		stepResult.SetError(fmt.Errorf("%w: %s", ErrInvalidAction, step.Action))
		stepResult.SetDuration(stepStart)
		return stepResult
	}

	// Execute the step
	providerResult, err := provider.ExecuteStep(ctx, step, state)
	if err != nil {
		stepResult.SetError(err)
		stepResult.SetDuration(stepStart)
		return stepResult
	}

	// Merge provider result output
	if providerResult != nil && providerResult.Output != nil {
		for k, v := range providerResult.Output {
			stepResult.AddOutput(k, v)
		}
	}

	// Run assertions
	for _, assertion := range step.Assertions {
		assertResult := e.checkAssertion(assertion, stepResult.Output, state)
		stepResult.AddAssertionResult(assertResult)
	}

	stepResult.SetDuration(stepStart)
	return stepResult
}

// checkAssertion evaluates an assertion against the output
func (e *Engine) checkAssertion(assertion Assertion, output map[string]interface{}, state *ExecutionState) AssertionResult {
	result := AssertionResult{
		Type:     assertion.Type,
		Path:     assertion.Path,
		Expected: assertion.Value,
		Reason:   assertion.Reason,
		Passed:   true,
	}

	// Get the actual value from output or state
	actual, found := e.getValueByPath(assertion.Path, output, state)
	result.Actual = actual

	// Handle not found case based on assertion type
	if !found {
		switch assertion.Type {
		case AssertEmpty:
			result.Passed = true
			return result
		case AssertNotEmpty:
			result.Passed = false
			result.Error = fmt.Sprintf("path '%s' not found", assertion.Path)
			return result
		default:
			result.Passed = false
			result.Error = fmt.Sprintf("path '%s' not found", assertion.Path)
			return result
		}
	}

	// Evaluate based on assertion type
	switch assertion.Type {
	case AssertEquals:
		result.Passed = e.valuesEqual(actual, assertion.Value)
		if !result.Passed {
			result.Error = fmt.Sprintf("expected %v, got %v", assertion.Value, actual)
		}

	case AssertGreaterThan:
		passed, err := e.compareNumeric(actual, assertion.Value, ">")
		result.Passed = passed
		if err != nil {
			result.Error = err.Error()
		}

	case AssertLessThan:
		passed, err := e.compareNumeric(actual, assertion.Value, "<")
		result.Passed = passed
		if err != nil {
			result.Error = err.Error()
		}

	case AssertBetween:
		passedMin, err1 := e.compareNumeric(actual, assertion.Min, ">=")
		passedMax, err2 := e.compareNumeric(actual, assertion.Max, "<=")
		result.Passed = passedMin && passedMax
		if err1 != nil || err2 != nil {
			result.Error = fmt.Sprintf("between comparison failed: min=%v, max=%v", err1, err2)
		}
		result.Expected = fmt.Sprintf("between %v and %v", assertion.Min, assertion.Max)

	case AssertContains:
		str, ok := actual.(string)
		expected, expectedOk := assertion.Value.(string)
		if !ok || !expectedOk {
			result.Passed = false
			result.Error = "contains assertion requires string values"
		} else {
			result.Passed = strings.Contains(str, expected)
			if !result.Passed {
				result.Error = fmt.Sprintf("'%s' does not contain '%s'", str, expected)
			}
		}

	case AssertNotEmpty:
		result.Passed = !e.isEmpty(actual)
		if !result.Passed {
			result.Error = "value is empty"
		}

	case AssertEmpty:
		result.Passed = e.isEmpty(actual)
		if !result.Passed {
			result.Error = fmt.Sprintf("expected empty, got %v", actual)
		}

	case AssertTrue:
		b, ok := actual.(bool)
		result.Passed = ok && b
		if !result.Passed {
			result.Error = fmt.Sprintf("expected true, got %v", actual)
		}

	case AssertFalse:
		b, ok := actual.(bool)
		result.Passed = ok && !b
		if !result.Passed {
			result.Error = fmt.Sprintf("expected false, got %v", actual)
		}

	case AssertErrorContains:
		str, ok := actual.(string)
		expected, expectedOk := assertion.Value.(string)
		if !ok || !expectedOk {
			result.Passed = false
			result.Error = "error_contains assertion requires string values"
		} else {
			result.Passed = strings.Contains(strings.ToLower(str), strings.ToLower(expected))
			if !result.Passed {
				result.Error = fmt.Sprintf("error '%s' does not contain '%s'", str, expected)
			}
		}

	default:
		result.Passed = false
		result.Error = fmt.Sprintf("unknown assertion type: %s", assertion.Type)
	}

	return result
}

// getValueByPath retrieves a value using a simple path notation (e.g., "output.items_gained.money")
func (e *Engine) getValueByPath(path string, output map[string]interface{}, state *ExecutionState) (interface{}, bool) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, false
	}

	// Determine the root source
	var current interface{}
	switch parts[0] {
	case "output":
		current = output
		parts = parts[1:]
	case "state":
		current = state.Results
		parts = parts[1:]
	case "user":
		if state.User == nil {
			return nil, false
		}
		current = map[string]interface{}{
			"user_id":     state.User.UserID,
			"username":    state.User.Username,
			"platform":    state.User.Platform,
			"platform_id": state.User.PlatformID,
		}
		parts = parts[1:]
	default:
		// Try output first, then state
		if v, ok := output[parts[0]]; ok {
			current = v
			parts = parts[1:]
		} else if v, ok := state.Results[parts[0]]; ok {
			current = v
			parts = parts[1:]
		} else {
			return nil, false
		}
	}

	// Navigate the path
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil, false
			}
		case map[string]int:
			val, ok := v[part]
			if !ok {
				return nil, false
			}
			current = val
		default:
			return nil, false
		}
	}

	return current, true
}

// valuesEqual compares two values for equality
func (e *Engine) valuesEqual(a, b interface{}) bool {
	// Handle numeric comparisons specially
	aNum, aIsNum := e.toFloat64(a)
	bNum, bIsNum := e.toFloat64(b)
	if aIsNum && bIsNum {
		return aNum == bNum
	}

	return reflect.DeepEqual(a, b)
}

// compareNumeric compares two numeric values
func (e *Engine) compareNumeric(actual, expected interface{}, op string) (bool, error) {
	a, aOk := e.toFloat64(actual)
	b, bOk := e.toFloat64(expected)

	if !aOk || !bOk {
		return false, fmt.Errorf("cannot compare non-numeric values: %v, %v", actual, expected)
	}

	switch op {
	case ">":
		return a > b, nil
	case ">=":
		return a >= b, nil
	case "<":
		return a < b, nil
	case "<=":
		return a <= b, nil
	case "==":
		return a == b, nil
	default:
		return false, fmt.Errorf("unknown comparison operator: %s", op)
	}
}

// toFloat64 converts a value to float64 if possible
func (e *Engine) toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

// isEmpty checks if a value is empty
func (e *Engine) isEmpty(v interface{}) bool {
	if v == nil {
		return true
	}

	switch val := v.(type) {
	case string:
		return val == ""
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	case map[string]int:
		return len(val) == 0
	default:
		return false
	}
}

// GetRegistry returns the engine's registry
func (e *Engine) GetRegistry() *Registry {
	return e.registry
}
