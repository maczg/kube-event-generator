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
