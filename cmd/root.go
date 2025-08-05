package cmd

import (
	"github.com/maczg/kube-event-generator/pkg/logger"
	"os"
)

// Execute runs the application.
func Execute() {
	app := NewApp()
	if err := app.Execute(); err != nil {
		logger.Default().Errorf("Error executing application: %v", err)
		os.Exit(1)
	}
}
