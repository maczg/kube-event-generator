package pkg

import (
	corev1 "k8s.io/api/core/v1"
	"time"
)

type EventType string

const (
	Submit EventType = "Submit"
	Evict  EventType = "Evict"
)

type Event struct {
	// The time at which this event should fire in real-time
	At time.Time
	// Action type: Submit or Evict
	Type EventType
	// The pod we are dealing with
	Pod *corev1.Pod
	// Optional: how long the pod should stay alive before eviction
	// Not used for Evict events directly, but used for scheduling eviction after the pod is Running
	Duration time.Duration
}

// EventQueue is a min-heap that pops the soonest event first.
type EventQueue []*Event

func (eq EventQueue) Len() int { return len(eq) }
func (eq EventQueue) Less(i, j int) bool {
	return eq[i].At.Before(eq[j].At)
}
func (eq EventQueue) Swap(i, j int) {
	eq[i], eq[j] = eq[j], eq[i]
}

func (eq *EventQueue) Push(x interface{}) {
	*eq = append(*eq, x.(*Event))
}

func (eq *EventQueue) Pop() interface{} {
	old := *eq
	n := len(old)
	item := old[n-1]
	*eq = old[0 : n-1]
	return item
}
