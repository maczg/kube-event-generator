package scheduler

import (
	"container/heap"
	"fmt"
	"sync"
)

// Queue is a thread-safe priority queue that implements heap.Interface
type Queue[T SchedulableEvent] struct {
	items    []T
	mu       sync.RWMutex
	capacity int
}

// NewQueue creates a new Queue
func NewQueue[T SchedulableEvent]() *Queue[T] {
	q := &Queue[T]{
		items:    make([]T, 0),
		capacity: 0, // unlimited
	}
	heap.Init(q)
	return q
}

// Peek returns the minimum item from the heap without removing it
func (q *Queue[T]) Peek() (T, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var zero T
	if len(q.items) == 0 {
		return zero, ErrQueueEmpty
	}

	return q.items[0], nil
}

// Size returns the number of items in the queue
func (q *Queue[T]) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items)
}

// IsEmpty returns true if the queue is empty
func (q *Queue[T]) IsEmpty() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items) == 0
}

// IsFull returns true if the queue has reached its capacity limit
func (q *Queue[T]) IsFull() bool {
	if q.capacity == 0 {
		return false // unlimited capacity
	}
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items) >= q.capacity
}

// Clear removes all items from the queue
func (q *Queue[T]) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = q.items[:0]
	heap.Init(q)
}

// PushEvent adds an event to the queue (thread-safe wrapper for Push)
func (q *Queue[T]) PushEvent(event T) error {
	if q.IsFull() {
		return NewQueueError("push", fmt.Errorf("queue capacity exceeded: %d", q.capacity))
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	heap.Push(q, event)
	return nil
}

// PopEvent removes and returns the minimum item from the queue
func (q *Queue[T]) PopEvent() (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	var zero T
	if len(q.items) == 0 {
		return zero, ErrQueueEmpty
	}

	item := heap.Pop(q).(T)
	return item, nil
}

// GetEvents returns a copy of all events in the queue (for inspection)
func (q *Queue[T]) GetEvents() []T {
	q.mu.RLock()
	defer q.mu.RUnlock()

	events := make([]T, len(q.items))
	copy(events, q.items)
	return events
}

// FindEvent searches for an event by ID
func (q *Queue[T]) FindEvent(eventID string) (T, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var zero T
	for _, event := range q.items {
		if event.GetID() == eventID {
			return event, true
		}
	}
	return zero, false
}

// RemoveEvent removes an event by ID
func (q *Queue[T]) RemoveEvent(eventID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, event := range q.items {
		if event.GetID() == eventID {
			// Mark as canceled before removing
			event.SetStatus(EventStatusCanceled)
			heap.Remove(q, i)
			return nil
		}
	}
	return ErrEventNotFound
}

// heap.Interface implementation methods
// Note: These methods should not be called directly - use the thread-safe wrappers above

// Len returns the number of items in the heap (called by heap package)
func (q *Queue[T]) Len() int {
	return len(q.items)
}

// Less returns whether the item at index i should sort before the item at index j
func (q *Queue[T]) Less(i, j int) bool {
	return q.items[i].HappensBefore(q.items[j])
}

// Swap swaps the items at the given indices
func (q *Queue[T]) Swap(i, j int) {
	q.items[i], q.items[j] = q.items[j], q.items[i]
}

// Push adds an item to the heap (called by heap package)
func (q *Queue[T]) Push(x any) {
	q.items = append(q.items, x.(T))
}

// Pop removes and returns the minimum item from the heap (called by heap package)
func (q *Queue[T]) Pop() any {
	old := q.items
	n := len(old)
	item := old[n-1]
	q.items = old[0 : n-1]
	return item
}

// String returns a string representation of the queue
func (q *Queue[T]) String() string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return fmt.Sprintf("Queue{size: %d, capacity: %d}", len(q.items), q.capacity)
}
