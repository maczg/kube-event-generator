package scheduler

import (
	"context"
	"time"
)

// EventStatus represents the current status of an event
type EventStatus string

const (
	EventStatusPending   EventStatus = "pending"
	EventStatusExecuting EventStatus = "executing"
	EventStatusCompleted EventStatus = "completed"
	EventStatusFailed    EventStatus = "failed"
	EventStatusCanceled  EventStatus = "canceled"
)

// SchedulableEvent interface defines the contract for all schedulable events
type SchedulableEvent interface {
	GetID() string
	GetStatus() EventStatus
	SetStatus(status EventStatus)
	Arrival() time.Duration
	Eviction() time.Duration
	Execute(ctx context.Context) error
	// GetExecuteTimeout bounds the API calls an event makes. The scheduler
	// runs every event in its own goroutine with a context that lasts as
	// long as the scheduler.
	GetExecuteTimeout() time.Duration
	SetExecuteTimeout(timeout time.Duration)
	HappensBefore(other SchedulableEvent) bool
}
