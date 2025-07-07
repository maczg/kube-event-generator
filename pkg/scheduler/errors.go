package scheduler

import (
	"github.com/maczg/kube-event-generator/pkg/errors"
)

// Deprecated: Use errors.ErrSchedulerAlreadyRunning instead.
var ErrAlreadyRunning = errors.ErrSchedulerAlreadyRunning

// Deprecated: Use errors.ErrSchedulerNotRunning instead.
var ErrNotRunning = errors.ErrSchedulerNotRunning

// Deprecated: Use errors.ErrSchedulerStopped instead.
var ErrStopped = errors.ErrSchedulerStopped
