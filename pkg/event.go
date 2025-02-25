package pkg

import (
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"time"
)

type EventType string

const (
	Submit EventType = "Submit"
	Evict  EventType = "Evict"
)

type IEvent interface {
	Handle(m *Manager)
	At() time.Time
	OfType() string
}

type Event struct {
	Type     EventType
	at       *time.Time
	duration *time.Duration
	PodSpec  *corev1.Pod
}

func (e *Event) OfType() string {
	return "Generic"
}

func (e *Event) Handle(m *Manager) {
	logrus.Infof("Generic event handling %s", e.Type)
}

func (e *Event) At() time.Time {
	return *e.at
}

func (e *Event) EndAt(t time.Time) time.Time {
	return t.Add(*e.duration)
}

func NewEvent(at *time.Time, duration *time.Duration) *Event {
	return &Event{
		at:       at,
		duration: duration,
	}
}

func NewEventWithType(t EventType, at *time.Time, duration *time.Duration) *Event {
	e := NewEvent(at, duration)
	e.Type = t
	return e
}
