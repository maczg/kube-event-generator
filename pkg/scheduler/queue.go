package scheduler

import (
	"container/heap"
	"sync"
)

// Queue is a type that implements heap.Interface and holds items of Schedulable type.
type Queue[T Schedulable] struct {
	items []T
	mu    sync.Mutex
}

// NewQueue creates a new Queue.
func NewQueue[T Schedulable]() *Queue[T] {
	g := &Queue[T]{
		mu:    sync.Mutex{},
		items: []T{},
	}
	heap.Init(g)

	return g
}

// Peek returns the minimum item from the heap without removing it.
func (q *Queue[T]) Peek() *T {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return nil
	}

	return &q.items[0]
}

// Len returns the number of items in the heap.
func (q *Queue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.items)
}

// Less returns whether the item at index i should sort before the item at index j.
func (q *Queue[T]) Less(i, j int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.items[i].ComparePriority(q.items[j])
}

// Swap swaps the items at the given indices.
func (q *Queue[T]) Swap(i, j int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items[i], q.items[j] = q.items[j], q.items[i]
}

// Push adds an item to the heap.
func (q *Queue[T]) Push(x any) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, x.(T))
}

// Pop removes and returns the minimum item from the heap.
func (q *Queue[T]) Pop() any {
	q.mu.Lock()
	defer q.mu.Unlock()
	old := q.items
	n := len(old)
	item := old[n-1]
	q.items = old[0 : n-1]

	return item
}
