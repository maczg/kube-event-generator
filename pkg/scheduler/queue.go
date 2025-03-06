package scheduler

import (
	"container/heap"
)

// Queue is a type that implements heap.Interface and holds items of any type.
type Queue[T any] struct {
	items []T
	less  func(a, b T) bool
}

// NewQueue creates a new Queue.
func NewQueue[T any](less func(a, b T) bool) *Queue[T] {
	g := &Queue[T]{
		items: []T{},
		less:  less,
	}
	heap.Init(g)
	return g
}

// Add adds an item to the heap.
func (h *Queue[T]) Add(item T) { heap.Push(h, item) }

// Items returns the items in the heap.
func (h *Queue[T]) Items() []T { return h.items }

// Peek returns the minimum item from the heap without removing it.
func (h *Queue[T]) Peek() T { return h.items[0] }

// Remove removes and returns the minimum item from the heap.
func (h *Queue[T]) Remove() T { return heap.Pop(h).(T) }

// Len returns the number of items in the heap.
func (h *Queue[T]) Len() int { return len(h.items) }

// Less returns whether the item at index i should sort before the item at index j.
func (h *Queue[T]) Less(i, j int) bool {
	return h.less(h.items[i], h.items[j])
}

// Swap swaps the items at the given indices.
func (h *Queue[T]) Swap(i, j int) { h.items[i], h.items[j] = h.items[j], h.items[i] }

// Push adds an item to the heap.
func (h *Queue[T]) Push(x any) { h.items = append(h.items, x.(T)) }

// Pop removes and returns the minimum item from the heap.
func (h *Queue[T]) Pop() any {
	old := h.items
	n := len(old)
	item := old[n-1]
	h.items = old[0 : n-1]
	return item
}
