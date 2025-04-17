package scenario

import (
	"k8s.io/apimachinery/pkg/util/json"
	"time"
)

// EventDuration is a custom type for representing event durations in JSON/YAML.
type EventDuration time.Duration

func (d *EventDuration) UnmarshalJSON(data []byte) error {
	var durationStr string
	if err := json.Unmarshal(data, &durationStr); err != nil {
		return err
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return err
	}
	*d = EventDuration(duration)
	return nil
}

func (d *EventDuration) MarshalJSON() ([]byte, error) {
	durationStr := time.Duration(*d).String()
	return json.Marshal(durationStr)
}
