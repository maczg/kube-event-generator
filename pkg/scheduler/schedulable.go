package scheduler

import (
	"context"
	"time"
)

// SchedulableCmpFn is a function that compares two Schedulable based on Schedulable.After() time duration.
var SchedulableCmpFn = func(i, j Schedulable) bool { return i.After() < j.After() }

// Schedulable represents an object that can be scheduled by a Scheduler
type Schedulable interface {
	// ID returns the unique identifier of a Schedulable object
	ID() string
	// Run executes the function of the Schedulable
	Run(ctx context.Context) error
	// After returns the duration after which the Schedulable should be executed
	After() time.Duration
	// For returns the duration for which the Schedulable should run
	For() time.Duration
}
