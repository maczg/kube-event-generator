package internal

import (
	"fmt"
	"os"

	"github.com/maczg/kube-event-generator/cmd/internal/app"
)

// Execute runs the application.
func Execute() {
	application := app.New()
	if err := application.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
