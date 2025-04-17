package scheduler

import (
	"context"
	"time"
)

type Schedulable interface {
	ID() string
	// ExecuteAfterDuration returns the duration after which the Schedulable should be executed
	ExecuteAfterDuration() time.Duration
	// ExecuteForDuration returns the duration for which the Schedulable should be executed
	ExecuteForDuration() time.Duration
	// Execute executes the Schedulable
	Execute(ctx context.Context) error
	// ComparePriority compares the Schedulable with another Schedulable
	ComparePriority(other Schedulable) bool
}
