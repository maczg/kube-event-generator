package scheduler

import (
	"errors"
	"fmt"
)

// Common scheduler errors
var (
	ErrSchedulerNotStarted     = errors.New("scheduler not started")
	ErrSchedulerAlreadyStarted = errors.New("scheduler already started")
	ErrSchedulerStopped        = errors.New("scheduler stopped")
	ErrInvalidEvent            = errors.New("invalid event")
	ErrEventNotFound           = errors.New("event not found")
	ErrQueueEmpty              = errors.New("queue is empty")
	ErrEventTimeout            = errors.New("event execution timeout")
	ErrHandlerNotFound         = errors.New("handler not found for event type")
)

// EventError represents an error that occurred during event processing
type EventError struct {
	EventID string
	Type    string
	Err     error
}

func (e *EventError) Error() string {
	return fmt.Sprintf("event error [%s:%s]: %v", e.Type, e.EventID, e.Err)
}

func (e *EventError) Unwrap() error {
	return e.Err
}

// NewEventError creates a new EventError
func NewEventError(eventID, eventType string, err error) *EventError {
	return &EventError{
		EventID: eventID,
		Type:    eventType,
		Err:     err,
	}
}

// QueueError represents an error that occurred during queue operations
type QueueError struct {
	Operation string
	Err       error
}

func (e *QueueError) Error() string {
	return fmt.Sprintf("queue error [%s]: %v", e.Operation, e.Err)
}

func (e *QueueError) Unwrap() error {
	return e.Err
}

// NewQueueError creates a new QueueError
func NewQueueError(operation string, err error) *QueueError {
	return &QueueError{
		Operation: operation,
		Err:       err,
	}
}
