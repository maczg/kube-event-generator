package scenario

import (
	"github.com/ghodss/yaml"
	v1 "k8s.io/api/core/v1"
	"time"
)

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
	From time.Duration `yaml:"from" json:"from"`
	// Duration for which the event should be applied. In case of Pods,
	//it is the time for which the pod should be running and then evicted.
	Duration time.Duration `yaml:"duration" json:"duration"`
	// Pod that will be created or evicted
	Pod v1.Pod `yaml:"pod" json:"pod"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// solve the conversion of time Duration and ResourceSpec in parsing
func (e *Event) UnmarshalJSON(data []byte) error {
	type alias struct {
		Name     string `yaml:"name" json:"name"`
		Type     string `yaml:"type" json:"type"`
		From     string `yaml:"from" json:"from"`
		Duration string `yaml:"duration" json:"duration"`
		Pod      v1.Pod `yaml:"pod" json:"pod"`
	}

	var a alias
	if err := yaml.Unmarshal(data, &a); err != nil {
		return err
	}
	e.Name = a.Name
	e.Type = a.Type
	e.Pod = a.Pod
	from, err := time.ParseDuration(a.From)
	if err != nil {
		return err
	}
	e.From = from
	duration, err := time.ParseDuration(a.Duration)
	if err != nil {
		return err
	}
	e.Duration = duration
	return nil
}
