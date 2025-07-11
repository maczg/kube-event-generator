package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"runtime"
	"strings"
)

// Logger wraps logrus.Logger with additional functionality.
type Logger struct {
	*logrus.Logger
}

// New creates a new logger instance.
func New() *Logger {
	l := logrus.New()

	// Set default formatter.
	l.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			parts := strings.Split(f.File, "/")
			file := parts[len(parts)-1]
			return "", fmt.Sprintf("[%s:%d]", file, f.Line)
		},
	})

	// Set default level.
	l.SetLevel(logrus.InfoLevel)
	l.SetReportCaller(true)

	return &Logger{Logger: l}
}

// SetVerbose enables verbose (debug) logging.
func (l *Logger) SetVerbose(verbose bool) {
	if verbose {
		l.SetLevel(logrus.DebugLevel)
	} else {
		l.SetLevel(logrus.InfoLevel)
	}
}

// SetJSONFormat enables JSON formatting.
func (l *Logger) SetJSONFormat(enabled bool) {
	if enabled {
		l.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	}
}

// SetOutput sets the logger output.
func (l *Logger) SetOutput(filename string) error {
	if filename == "" || filename == "-" {
		l.Logger.SetOutput(os.Stdout)
		return nil
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.Logger.SetOutput(file)

	return nil
}

// Default logger instance.
var defaultLogger = New()

// Default returns the default logger instance.
func Default() *Logger {
	return defaultLogger
}
