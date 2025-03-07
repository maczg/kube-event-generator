package scenario

import (
	"encoding/json"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	"time"
)

type Event struct {
	// Unique identifier of the event
	UID string `yaml:"id" json:"id"`
	// Name of the event
	Name string `yaml:"name" json:"name"`
	// From when the event should be applied
	From time.Duration `yaml:"from" json:"from"`
	// Duration for which the event should be applied. In case of Pods,
	//it is the time for which the pod should be running and then evicted.
	Duration time.Duration `yaml:"duration" json:"duration"`
	// Pod that will be created or evicted
	Pod v1.Pod `yaml:"pod" json:"pod"`
}

func NewEvent(name string, from, duration time.Duration, pod v1.Pod) *Event {
	return &Event{
		UID:      uuid.New().String(),
		Name:     name,
		From:     from,
		Duration: duration,
		Pod:      pod,
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// solve the conversion of time Duration and ResourceSpec in parsing
func (e *Event) UnmarshalJSON(data []byte) error {
	type alias struct {
		Name     string `yaml:"name" json:"name"`
		From     string `yaml:"from" json:"from"`
		Duration string `yaml:"duration" json:"duration"`
		Pod      v1.Pod `yaml:"pod" json:"pod"`
	}

	var a alias
	if err := yaml.Unmarshal(data, &a); err != nil {
		return err
	}
	e.Name = a.Name
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

func (e *Event) MarshalJSON() ([]byte, error) {
	type alias struct {
		Uid      string `yaml:"id" json:"id"`
		Name     string `yaml:"name" json:"name"`
		From     string `yaml:"from" json:"from"`
		Duration string `yaml:"duration" json:"duration"`
		Pod      v1.Pod `yaml:"pod" json:"pod"`
	}
	a := alias{
		Uid:      e.UID,
		Name:     e.Name,
		From:     e.From.Round(time.Second).String(),
		Duration: e.Duration.Round(time.Second).String(),
		Pod:      e.Pod,
	}
	return json.Marshal(a)
}
