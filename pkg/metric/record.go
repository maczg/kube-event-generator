// Package metric Only god and ChatGPT knows what is happening here
package metric

import (
	"time"
)

type Record struct {
	// timestamp of the record
	timestamp time.Time
	// labels of the record
	labels map[string]string
	// value of the record
	value float64
}

func (r Record) Timestamp() time.Time {
	return r.timestamp
}

func (r Record) Labels() map[string]string {
	return r.labels
}

func (r Record) Value() float64 {
	return r.value
}
