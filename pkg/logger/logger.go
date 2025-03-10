package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

type Logger interface {
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
}

var _ Logger = (*logger)(nil)

// LogLevel is the level of logging that should be logged
// when using the basic NewLogger.
type LogLevel int

// The different log levels that can be used.
const (
	LevelError LogLevel = iota
	LevelWarn
	LevelInfo
	LevelDebug
)

type logger struct {
	log       *logrus.Logger
	level     LogLevel
	component string
}

// NewLogger returns a new Logger that logs at the given level.
func NewLogger(level LogLevel, component string) Logger {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	return &logger{
		log:       log,
		level:     level,
		component: component,
	}
}

func (l *logger) Debug(msg string, args ...any) {
	msg = fmt.Sprintf("[%s] %s", l.component, msg)
	l.log.Debugf(msg, args...)
}

func (l *logger) Error(msg string, args ...any) {
	msg = fmt.Sprintf("[%s] %s", l.component, msg)
	l.log.Errorf(msg, args...)
}

func (l *logger) Info(msg string, args ...any) {
	msg = fmt.Sprintf("[%s] %s", l.component, msg)
	l.log.Infof(msg, args...)
}

func (l *logger) Warn(msg string, args ...any) {
	msg = fmt.Sprintf("[%s] %s", l.component, msg)
	l.log.Warnf(msg, args...)
}
