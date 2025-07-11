package util

import (
	"os"
	"os/signal"
	"syscall"
)

var stopCh chan os.Signal

// GetStopChan returns a channel that will receive os.Signal notifications.
func GetStopChan() <-chan os.Signal {
	if stopCh == nil {
		stopCh = make(chan os.Signal, 1)
		signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	}

	return stopCh
}

// WaitStopAndExecute waits for a stop signal and then executes the provided function.
func WaitStopAndExecute(fn func()) {
	<-GetStopChan()
	fn()
}
