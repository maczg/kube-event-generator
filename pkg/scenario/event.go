package scenario

import v1 "k8s.io/api/core/v1"

type EventType string

const (
	PodCreateType EventType = "PodCreate"
	PodEvictType  EventType = "PodEvict"
)

type Event struct {
	// Name of the event
	Name string `yaml:"name" json:"name"`
	// Type of the event
	Type string `yaml:"type" json:"type"`
	// From when the event should be applied
	From string `yaml:"after" json:"after"`
	// Duration for which the event should be applied. In case of Pods,
	//it is the time for which the pod should be running and then evicted.
	Duration string `yaml:"duration" json:"duration"`
	// Pod that will be created or evicted
	Pod v1.Pod `yaml:"pod" json:"pod"`
}
