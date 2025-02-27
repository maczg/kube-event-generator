package metric

import "time"

var less = func(a, b Record) bool {
	return a.timestamp.Before(b.timestamp)
}

type Record struct {
	timestamp time.Time
	value     float64
}
