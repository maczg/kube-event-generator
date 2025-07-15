package cache

import "time"

// Record is a generic type that holds a value and a timestamp.
// It is used to store the history of a value over time.
type Record[T any] struct {
	Value T
	At    time.Time
}

// NewRecord creates a new Record object with the given value and the current time.
func NewRecord[T any](value T) Record[T] {
	return Record[T]{
		Value: value,
		At:    time.Now(),
	}
}
