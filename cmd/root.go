package cmd

import (
	"github.com/sirupsen/logrus"
	"os"
)

// Execute runs the application.
func Execute() {
	app := NewApp()
	if err := app.Execute(); err != nil {
		logrus.Errorf("Error executing application: %v", err)
		os.Exit(1)
	}
}
