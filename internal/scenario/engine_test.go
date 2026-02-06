package scenario

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProvider is a simple mock provider for testing
type MockProvider struct {
	feature     string
	scenarios   []Scenario
	stepResults map[ActionType]*StepResult
	stepErrors  map[ActionType]error
}

func NewMockProvider(feature string) *MockProvider {
	return &MockProvider{
		feature:     feature,
		scenarios:   make([]Scenario, 0),
		stepResults: make(map[ActionType]*StepResult),
		stepErrors:  make(map[ActionType]error),
	}
}

func (p *MockProvider) Feature() string {
	return p.feature
}

func (p *MockProvider) Capabilities() []CapabilityType {
	return []CapabilityType{CapabilityTimeWarp}
}

func (p *MockProvider) GetCapabilityInfo() []CapabilityInfo {
	return []CapabilityInfo{
		{
			Type:        CapabilityTimeWarp,
			Name:        "Test Time Warp",
			Description: "Test capability",
		},
	}
}

func (p *MockProvider) SupportsAction(action ActionType) bool {
	_, ok := p.stepResults[action]
	return ok
}

func (p *MockProvider) PrebuiltScenarios() []Scenario {
	return p.scenarios
}

func (p *MockProvider) ExecuteStep(ctx context.Context, step Step, state *ExecutionState) (*StepResult, error) {
	if err, ok := p.stepErrors[step.Action]; ok && err != nil {
		return nil, err
	}

	if result, ok := p.stepResults[step.Action]; ok {
		return result, nil
	}

	return NewStepResult(step.Name, 0, step.Action), nil
}

func (p *MockProvider) AddScenario(s Scenario) {
	p.scenarios = append(p.scenarios, s)
}

func (p *MockProvider) SetStepResult(action ActionType, result *StepResult) {
	p.stepResults[action] = result
}

func (p *MockProvider) SetStepError(action ActionType, err error) {
	p.stepErrors[action] = err
}

func TestNewEngine(t *testing.T) {
	registry := NewRegistry()
	engine := NewEngine(registry)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.GetRegistry())
}

func TestEngineExecute_ScenarioNotFound(t *testing.T) {
	registry := NewRegistry()
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "nonexistent", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "scenario not found")
}

func TestEngineExecute_SimpleScenario(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	// Set up a step result
	stepResult := NewStepResult("test_step", 0, ActionSetState)
	stepResult.AddOutput("value", 42)
	provider.SetStepResult(ActionSetState, stepResult)

	// Add a simple scenario
	provider.AddScenario(Scenario{
		ID:      "test_scenario",
		Name:    "Test Scenario",
		Feature: "test",
		Steps: []Step{
			{
				Name:       "test_step",
				Action:     ActionSetState,
				Parameters: map[string]interface{}{},
			},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "test_scenario", nil)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "test_scenario", result.ScenarioID)
	assert.Equal(t, "Test Scenario", result.ScenarioName)
	assert.Len(t, result.Steps, 1)
	assert.True(t, result.Steps[0].Success)
}

func TestEngineExecute_MultipleSteps(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	// Set up step results
	step1Result := NewStepResult("step1", 0, ActionSetState)
	step1Result.AddOutput("initialized", true)
	provider.SetStepResult(ActionSetState, step1Result)

	step2Result := NewStepResult("step2", 1, ActionTimeWarp)
	step2Result.AddOutput("warped_hours", 168.0)
	provider.SetStepResult(ActionTimeWarp, step2Result)

	provider.AddScenario(Scenario{
		ID:      "multi_step",
		Name:    "Multi Step Scenario",
		Feature: "test",
		Steps: []Step{
			{Name: "step1", Action: ActionSetState, Parameters: map[string]interface{}{}},
			{Name: "step2", Action: ActionTimeWarp, Parameters: map[string]interface{}{}},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "multi_step", nil)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Len(t, result.Steps, 2)
	assert.True(t, result.Steps[0].Success)
	assert.True(t, result.Steps[1].Success)
}

func TestEngineExecute_WithParameters(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	stepResult := NewStepResult("step", 0, ActionSetState)
	provider.SetStepResult(ActionSetState, stepResult)

	provider.AddScenario(Scenario{
		ID:      "param_test",
		Name:    "Parameter Test",
		Feature: "test",
		Steps: []Step{
			{Name: "step", Action: ActionSetState, Parameters: map[string]interface{}{}},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	params := map[string]interface{}{
		"custom_param": "custom_value",
	}

	result, err := engine.Execute(context.Background(), "param_test", params)

	require.NoError(t, err)
	assert.True(t, result.Success)
	// Parameters should be in final state
	assert.Equal(t, "custom_value", result.FinalState["custom_param"])
}

func TestEngineAssertions_Equals(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	stepResult := NewStepResult("step", 0, ActionSetState)
	stepResult.AddOutput("value", 42)
	provider.SetStepResult(ActionSetState, stepResult)

	provider.AddScenario(Scenario{
		ID:      "assert_test",
		Name:    "Assertion Test",
		Feature: "test",
		Steps: []Step{
			{
				Name:       "step",
				Action:     ActionSetState,
				Parameters: map[string]interface{}{},
				Assertions: []Assertion{
					{Type: AssertEquals, Path: "output.value", Value: 42},
				},
			},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "assert_test", nil)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Len(t, result.Steps[0].Assertions, 1)
	assert.True(t, result.Steps[0].Assertions[0].Passed)
}

func TestEngineAssertions_EqualsFails(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	stepResult := NewStepResult("step", 0, ActionSetState)
	stepResult.AddOutput("value", 41) // Wrong value
	provider.SetStepResult(ActionSetState, stepResult)

	provider.AddScenario(Scenario{
		ID:      "assert_fail_test",
		Name:    "Assertion Fail Test",
		Feature: "test",
		Steps: []Step{
			{
				Name:       "step",
				Action:     ActionSetState,
				Parameters: map[string]interface{}{},
				Assertions: []Assertion{
					{Type: AssertEquals, Path: "output.value", Value: 42},
				},
			},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "assert_fail_test", nil)

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.False(t, result.Steps[0].Success)
	assert.False(t, result.Steps[0].Assertions[0].Passed)
}

func TestEngineAssertions_GreaterThan(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	stepResult := NewStepResult("step", 0, ActionSetState)
	stepResult.AddOutput("hours", 168.0)
	provider.SetStepResult(ActionSetState, stepResult)

	provider.AddScenario(Scenario{
		ID:      "gt_test",
		Name:    "Greater Than Test",
		Feature: "test",
		Steps: []Step{
			{
				Name:       "step",
				Action:     ActionSetState,
				Parameters: map[string]interface{}{},
				Assertions: []Assertion{
					{Type: AssertGreaterThan, Path: "output.hours", Value: 100.0},
				},
			},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "gt_test", nil)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.Steps[0].Assertions[0].Passed)
}

func TestEngineAssertions_LessThan(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	stepResult := NewStepResult("step", 0, ActionSetState)
	stepResult.AddOutput("hours", 50.0)
	provider.SetStepResult(ActionSetState, stepResult)

	provider.AddScenario(Scenario{
		ID:      "lt_test",
		Name:    "Less Than Test",
		Feature: "test",
		Steps: []Step{
			{
				Name:       "step",
				Action:     ActionSetState,
				Parameters: map[string]interface{}{},
				Assertions: []Assertion{
					{Type: AssertLessThan, Path: "output.hours", Value: 100.0},
				},
			},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "lt_test", nil)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.Steps[0].Assertions[0].Passed)
}

func TestEngineAssertions_Between(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	stepResult := NewStepResult("step", 0, ActionSetState)
	stepResult.AddOutput("value", 50)
	provider.SetStepResult(ActionSetState, stepResult)

	provider.AddScenario(Scenario{
		ID:      "between_test",
		Name:    "Between Test",
		Feature: "test",
		Steps: []Step{
			{
				Name:       "step",
				Action:     ActionSetState,
				Parameters: map[string]interface{}{},
				Assertions: []Assertion{
					{Type: AssertBetween, Path: "output.value", Min: 10, Max: 100},
				},
			},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "between_test", nil)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.Steps[0].Assertions[0].Passed)
}

func TestEngineAssertions_Contains(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	stepResult := NewStepResult("step", 0, ActionSetState)
	stepResult.AddOutput("message", "Harvest successful!")
	provider.SetStepResult(ActionSetState, stepResult)

	provider.AddScenario(Scenario{
		ID:      "contains_test",
		Name:    "Contains Test",
		Feature: "test",
		Steps: []Step{
			{
				Name:       "step",
				Action:     ActionSetState,
				Parameters: map[string]interface{}{},
				Assertions: []Assertion{
					{Type: AssertContains, Path: "output.message", Value: "successful"},
				},
			},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "contains_test", nil)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.Steps[0].Assertions[0].Passed)
}

func TestEngineAssertions_NotEmpty(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	stepResult := NewStepResult("step", 0, ActionSetState)
	stepResult.AddOutput("items", map[string]int{"money": 100})
	provider.SetStepResult(ActionSetState, stepResult)

	provider.AddScenario(Scenario{
		ID:      "not_empty_test",
		Name:    "Not Empty Test",
		Feature: "test",
		Steps: []Step{
			{
				Name:       "step",
				Action:     ActionSetState,
				Parameters: map[string]interface{}{},
				Assertions: []Assertion{
					{Type: AssertNotEmpty, Path: "output.items"},
				},
			},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "not_empty_test", nil)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.Steps[0].Assertions[0].Passed)
}

func TestEngineAssertions_Empty(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	stepResult := NewStepResult("step", 0, ActionSetState)
	stepResult.AddOutput("items", map[string]int{})
	provider.SetStepResult(ActionSetState, stepResult)

	provider.AddScenario(Scenario{
		ID:      "empty_test",
		Name:    "Empty Test",
		Feature: "test",
		Steps: []Step{
			{
				Name:       "step",
				Action:     ActionSetState,
				Parameters: map[string]interface{}{},
				Assertions: []Assertion{
					{Type: AssertEmpty, Path: "output.items"},
				},
			},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "empty_test", nil)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.Steps[0].Assertions[0].Passed)
}

func TestEngineAssertions_True(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	stepResult := NewStepResult("step", 0, ActionSetState)
	stepResult.AddOutput("initialized", true)
	provider.SetStepResult(ActionSetState, stepResult)

	provider.AddScenario(Scenario{
		ID:      "true_test",
		Name:    "True Test",
		Feature: "test",
		Steps: []Step{
			{
				Name:       "step",
				Action:     ActionSetState,
				Parameters: map[string]interface{}{},
				Assertions: []Assertion{
					{Type: AssertTrue, Path: "output.initialized"},
				},
			},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "true_test", nil)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.Steps[0].Assertions[0].Passed)
}

func TestEngineAssertions_False(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	stepResult := NewStepResult("step", 0, ActionSetState)
	stepResult.AddOutput("spoiled", false)
	provider.SetStepResult(ActionSetState, stepResult)

	provider.AddScenario(Scenario{
		ID:      "false_test",
		Name:    "False Test",
		Feature: "test",
		Steps: []Step{
			{
				Name:       "step",
				Action:     ActionSetState,
				Parameters: map[string]interface{}{},
				Assertions: []Assertion{
					{Type: AssertFalse, Path: "output.spoiled"},
				},
			},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "false_test", nil)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.Steps[0].Assertions[0].Passed)
}

func TestEngineExecute_StopsOnFailure(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	step1Result := NewStepResult("step1", 0, ActionSetState)
	step1Result.AddOutput("value", 0) // Will fail assertion
	provider.SetStepResult(ActionSetState, step1Result)

	step2Result := NewStepResult("step2", 1, ActionTimeWarp)
	step2Result.AddOutput("executed", true)
	provider.SetStepResult(ActionTimeWarp, step2Result)

	provider.AddScenario(Scenario{
		ID:      "stop_on_fail",
		Name:    "Stop on Fail",
		Feature: "test",
		Steps: []Step{
			{
				Name:       "step1",
				Action:     ActionSetState,
				Parameters: map[string]interface{}{},
				Assertions: []Assertion{
					{Type: AssertEquals, Path: "output.value", Value: 42},
				},
			},
			{
				Name:       "step2",
				Action:     ActionTimeWarp,
				Parameters: map[string]interface{}{},
			},
		},
	})

	registry.Register(provider)
	engine := NewEngine(registry)

	result, err := engine.Execute(context.Background(), "stop_on_fail", nil)

	require.NoError(t, err)
	assert.False(t, result.Success)
	// Should only have 1 step executed (stopped after first failure)
	assert.Len(t, result.Steps, 1)
}

func TestEngineExecuteCustom_Success(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test")

	stepResult := NewStepResult("custom_step", 0, ActionSetState)
	stepResult.AddOutput("custom", true)
	provider.SetStepResult(ActionSetState, stepResult)

	registry.Register(provider)
	engine := NewEngine(registry)

	customScenario := Scenario{
		ID:      "custom_scenario",
		Name:    "Custom Scenario",
		Feature: "test",
		Steps: []Step{
			{
				Name:       "custom_step",
				Action:     ActionSetState,
				Parameters: map[string]interface{}{},
			},
		},
	}

	result, err := engine.ExecuteCustom(context.Background(), customScenario, nil)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "custom_scenario", result.ScenarioID)
}

func TestEngineExecuteCustom_ProviderNotFound(t *testing.T) {
	registry := NewRegistry()
	engine := NewEngine(registry)

	customScenario := Scenario{
		ID:      "custom_scenario",
		Name:    "Custom Scenario",
		Feature: "nonexistent_feature",
		Steps:   []Step{},
	}

	result, err := engine.ExecuteCustom(context.Background(), customScenario, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "provider not found")
}

func TestExecutionResult_Summary(t *testing.T) {
	result := NewExecutionResult("test", "Test Scenario")
	result.Steps = []StepResult{
		{StepName: "step1", Success: true, Assertions: []AssertionResult{{Passed: true}}},
		{StepName: "step2", Success: true, Assertions: []AssertionResult{{Passed: true}, {Passed: false}}},
		{StepName: "step3", Success: false, Assertions: []AssertionResult{{Passed: false}}},
	}
	result.Success = false
	result.Complete()

	summary := result.ToSummary()

	assert.Equal(t, "test", summary.ScenarioID)
	assert.False(t, summary.Success)
	assert.Equal(t, 3, summary.TotalSteps)
	assert.Equal(t, 2, summary.PassedSteps)
	assert.Equal(t, 4, summary.TotalAssertions)
	assert.Equal(t, 2, summary.PassedAssertions)
}

func TestSimulatedClock(t *testing.T) {
	start := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewSimulatedClock(start)

	assert.Equal(t, start, clock.Now())

	// Test Advance
	clock.Advance(2 * time.Hour)
	expected := time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, clock.Now())

	// Test AdvanceHours
	clock.AdvanceHours(24)
	expected = time.Date(2024, 1, 2, 14, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, clock.Now())

	// Test Set
	newTime := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	clock.Set(newTime)
	assert.Equal(t, newTime, clock.Now())
}

func TestExecutionState(t *testing.T) {
	state := NewExecutionState()

	assert.NotNil(t, state.Clock)
	assert.NotNil(t, state.Results)
	assert.Empty(t, state.Errors)

	// Test SetResult and GetResult
	state.SetResult("key", "value")
	val, ok := state.GetResult("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	// Test missing key
	_, ok = state.GetResult("nonexistent")
	assert.False(t, ok)

	// Test AddError
	assert.False(t, state.HasErrors())
	state.AddError(assert.AnError)
	assert.True(t, state.HasErrors())
	assert.Len(t, state.Errors, 1)
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// Test Register and Get
	provider := NewMockProvider("test")
	registry.Register(provider)

	got, ok := registry.Get("test")
	assert.True(t, ok)
	assert.Equal(t, provider, got)

	_, ok = registry.Get("nonexistent")
	assert.False(t, ok)

	// Test Features
	features := registry.Features()
	assert.Contains(t, features, "test")

	// Test GetAll
	providers := registry.GetAll()
	assert.Len(t, providers, 1)

	// Test HasCapability
	assert.True(t, registry.HasCapability(CapabilityTimeWarp))
	assert.False(t, registry.HasCapability(CapabilityMultiUser))

	// Test ProvidersWithCapability
	withTimeWarp := registry.ProvidersWithCapability(CapabilityTimeWarp)
	assert.Len(t, withTimeWarp, 1)
}
