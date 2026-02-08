package scenario

import (
	"errors"
	"fmt"
)

// Common errors for the scenario engine
var (
	// ErrProviderNotFound indicates the requested provider was not found
	ErrProviderNotFound = errors.New("scenario provider not found")

	// ErrScenarioNotFound indicates the requested scenario was not found
	ErrScenarioNotFound = errors.New("scenario not found")

	// ErrInvalidAction indicates an invalid or unsupported action
	ErrInvalidAction = errors.New("invalid action")

	// ErrMissingParameter indicates a required parameter is missing
	ErrMissingParameter = errors.New("missing required parameter")

	// ErrInvalidParameter indicates a parameter has an invalid value
	ErrInvalidParameter = errors.New("invalid parameter value")

	// ErrAssertionFailed indicates an assertion failed
	ErrAssertionFailed = errors.New("assertion failed")

	// ErrExecutionFailed indicates the scenario execution failed
	ErrExecutionFailed = errors.New("scenario execution failed")

	// ErrUserNotInitialized indicates no user has been set up for the scenario
	ErrUserNotInitialized = errors.New("user not initialized for scenario")

	// ErrCapabilityNotSupported indicates the provider doesn't support the required capability
	ErrCapabilityNotSupported = errors.New("capability not supported by provider")

	// ErrInvalidTimeDelta indicates an invalid time delta value
	ErrInvalidTimeDelta = errors.New("invalid time delta")

	// ErrDatabaseOperation indicates a database operation failed
	ErrDatabaseOperation = errors.New("database operation failed")
)

// ParameterError represents an error with a specific parameter
type ParameterError struct {
	Parameter string
	Message   string
	Err       error
}

func (e *ParameterError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("parameter '%s': %s: %v", e.Parameter, e.Message, e.Err)
	}
	return fmt.Sprintf("parameter '%s': %s", e.Parameter, e.Message)
}

func (e *ParameterError) Unwrap() error {
	return e.Err
}

// NewParameterError creates a new ParameterError
func NewParameterError(param, message string) *ParameterError {
	return &ParameterError{
		Parameter: param,
		Message:   message,
	}
}

// NewParameterErrorWithCause creates a new ParameterError with a cause
func NewParameterErrorWithCause(param, message string, err error) *ParameterError {
	return &ParameterError{
		Parameter: param,
		Message:   message,
		Err:       err,
	}
}

// StepError represents an error that occurred during step execution
type StepError struct {
	StepName  string
	StepIndex int
	Action    ActionType
	Message   string
	Err       error
}

func (e *StepError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("step %d '%s' (action: %s): %s: %v",
			e.StepIndex, e.StepName, e.Action, e.Message, e.Err)
	}
	return fmt.Sprintf("step %d '%s' (action: %s): %s",
		e.StepIndex, e.StepName, e.Action, e.Message)
}

func (e *StepError) Unwrap() error {
	return e.Err
}

// NewStepError creates a new StepError
func NewStepError(step Step, index int, message string) *StepError {
	return &StepError{
		StepName:  step.Name,
		StepIndex: index,
		Action:    step.Action,
		Message:   message,
	}
}

// NewStepErrorWithCause creates a new StepError with a cause
func NewStepErrorWithCause(step Step, index int, message string, err error) *StepError {
	return &StepError{
		StepName:  step.Name,
		StepIndex: index,
		Action:    step.Action,
		Message:   message,
		Err:       err,
	}
}

// AssertionError represents a failed assertion
type AssertionError struct {
	Assertion Assertion
	Actual    interface{}
	Message   string
}

func (e *AssertionError) Error() string {
	return fmt.Sprintf("assertion '%s' on '%s' failed: expected %v, got %v - %s",
		e.Assertion.Type, e.Assertion.Path, e.Assertion.Value, e.Actual, e.Message)
}

// NewAssertionError creates a new AssertionError
func NewAssertionError(assertion Assertion, actual interface{}, message string) *AssertionError {
	return &AssertionError{
		Assertion: assertion,
		Actual:    actual,
		Message:   message,
	}
}

// WrapProviderError wraps an error from a provider with context
func WrapProviderError(provider, action string, err error) error {
	return fmt.Errorf("provider '%s' action '%s': %w", provider, action, err)
}

// WrapDatabaseError wraps a database error with context
func WrapDatabaseError(operation string, err error) error {
	return fmt.Errorf("%w: %s: %w", ErrDatabaseOperation, operation, err)
}
