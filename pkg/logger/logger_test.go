package logger

import (
	"testing"
)

func TestLogger_Info(t *testing.T) {
	// Create a new logger
	log := NewLogger(LevelInfo, "test")
	log.Info("This is a test")

}
