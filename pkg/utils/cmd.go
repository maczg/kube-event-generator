package utils

import (
	"os"
	"os/signal"
	"syscall"
)

//func WaitStopAndExecute(fn func()) {
//	sig := make(chan os.Signal, 1)
//	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
//	<-sig
//	fn()
//}

var stopCh chan os.Signal

func GetStopChan() <-chan os.Signal {
	if stopCh == nil {
		stopCh = make(chan os.Signal, 1)
		signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	}
	return stopCh
}

func WaitStopAndExecute(fn func()) {
	<-GetStopChan()
	fn()
}
