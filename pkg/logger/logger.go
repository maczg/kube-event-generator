package logger

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// contextKey is a custom type for context keys.
type contextKey string

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey contextKey = "request_id"

	// SimulationIDKey is the context key for simulation ID.
	SimulationIDKey contextKey = "simulation_id"

	// EventIDKey is the context key for event ID.
	EventIDKey contextKey = "event_id"
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

// WithContext returns a logger with context fields.
func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	fields := logrus.Fields{}

	// Extract common context values.
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		fields["request_id"] = requestID
	}

	if simulationID, ok := ctx.Value(SimulationIDKey).(string); ok {
		fields["simulation_id"] = simulationID
	}

	if eventID, ok := ctx.Value(EventIDKey).(string); ok {
		fields["event_id"] = eventID
	}

	return l.WithFields(fields)
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

// Context helpers.

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context) context.Context {
	return context.WithValue(ctx, RequestIDKey, uuid.New().String())
}

// WithSimulationID adds a simulation ID to the context.
func WithSimulationID(ctx context.Context, simulationID string) context.Context {
	return context.WithValue(ctx, SimulationIDKey, simulationID)
}

// WithEventID adds an event ID to the context.
func WithEventID(ctx context.Context, eventID string) context.Context {
	return context.WithValue(ctx, EventIDKey, eventID)
}

// GetRequestID extracts request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}

	return ""
}

// Default logger instance.
var defaultLogger = New()

// Default returns the default logger instance.
func Default() *Logger {
	return defaultLogger
}

// Helper functions that use the default logger.

// Debug logs a debug message.
func Debug(args ...interface{}) {
	defaultLogger.Debug(args...)
}

// Debugf logs a formatted debug message.
func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

// Info logs an info message.
func Info(args ...interface{}) {
	defaultLogger.Info(args...)
}

// Infof logs a formatted info message.
func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Warn logs a warning message.
func Warn(args ...interface{}) {
	defaultLogger.Warn(args...)
}

// Warnf logs a formatted warning message.
func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

// Error logs an error message.
func Error(args ...interface{}) {
	defaultLogger.Error(args...)
}

// Errorf logs a formatted error message.
func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// WithContext returns a logger with context fields using the default logger.
func WithContext(ctx context.Context) *logrus.Entry {
	return defaultLogger.WithContext(ctx)
}

// WithFields returns a logger with additional fields using the default logger.
func WithFields(fields logrus.Fields) *logrus.Entry {
	return defaultLogger.WithFields(fields)
}
