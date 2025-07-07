package errors

import (
	"errors"
	"fmt"
)

// Define sentinel errors for common cases.
var (
	ErrSchedulerAlreadyRunning = errors.New("scheduler is already running")
	ErrSchedulerNotRunning     = errors.New("scheduler is not running")
	ErrSchedulerStopped        = errors.New("scheduler has been stopped")

	ErrSimulationFailed = errors.New("simulation failed")
	ErrInvalidScenario  = errors.New("invalid scenario")
	ErrNoPodsInScenario = errors.New("no pods defined in scenario")

	ErrInvalidConfiguration = errors.New("invalid configuration")
	ErrMissingRequired      = errors.New("missing required field")

	ErrPodCreationFailed = errors.New("failed to create pod")
	ErrPodDeletionFailed = errors.New("failed to delete pod")
	ErrWatchFailed       = errors.New("failed to watch resources")
)

// Error types for more complex errors.

// SchedulerError represents scheduler-specific errors.
type SchedulerError struct {
	Wrapped error
	Op      string
	Event   string
}

func (e *SchedulerError) Error() string {
	if e.Event != "" {
		return fmt.Sprintf("scheduler %s failed for event %s: %v", e.Op, e.Event, e.Wrapped)
	}

	return fmt.Sprintf("scheduler %s failed: %v", e.Op, e.Wrapped)
}

func (e *SchedulerError) Unwrap() error {
	return e.Wrapped
}

// SimulationError represents simulation-specific errors.
type SimulationError struct {
	Wrapped error
	Name    string
	Phase   string
}

func (e *SimulationError) Error() string {
	return fmt.Sprintf("simulation %s failed during %s: %v", e.Name, e.Phase, e.Wrapped)
}

func (e *SimulationError) Unwrap() error {
	return e.Wrapped
}

// ValidationError represents validation errors.
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %s with value %v: %s", e.Field, e.Value, e.Message)
}

// Helper functions.

// WrapSchedulerError wraps an error with scheduler context.
func WrapSchedulerError(op, event string, err error) error {
	if err == nil {
		return nil
	}

	return &SchedulerError{
		Op:      op,
		Event:   event,
		Wrapped: err,
	}
}

// WrapSimulationError wraps an error with simulation context.
func WrapSimulationError(name, phase string, err error) error {
	if err == nil {
		return nil
	}

	return &SimulationError{
		Name:    name,
		Phase:   phase,
		Wrapped: err,
	}
}

// NewValidationError creates a new validation error.
func NewValidationError(field string, value interface{}, message string) error {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}
