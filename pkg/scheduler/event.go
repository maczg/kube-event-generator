package scheduler

import (
	"context"
	"time"
)

// EventLessFn is a function that compares two events based on their After time duration.
var EventLessFn = func(i, j Event) bool { return i.After() < j.After() }

// Event represents an event that can be scheduled
type Event interface {
	// ID returns the unique identifier of the event
	ID() string
	// Run executes the event
	Run(ctx context.Context) error
	// After returns the duration after which the event should be executed
	After() time.Duration
	// Duration returns the duration for which the event should run
	Duration() time.Duration
}
