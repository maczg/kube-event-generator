package scheduler

import "errors"

var ErrAlreadyRunning = errors.New("scheduler already running")
var ErrNotRunning = errors.New("scheduler not running")
