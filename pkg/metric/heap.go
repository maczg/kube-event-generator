package metric

import (
	"container/heap"
)

// GenericHeap is a type that implements heap.Interface and holds items of any type.
type GenericHeap[T any] struct {
	items []T
	less  func(a, b T) bool
	push  func(x T)
	pop   func() T
}

// NewGenericHeap creates a new GenericHeap.
func NewGenericHeap[T any](less func(a, b T) bool) *GenericHeap[T] {
	g := &GenericHeap[T]{
		items: []T{},
		less:  less,
	}
	heap.Init(g)
	return g
}

// Len returns the number of items in the heap.
func (h GenericHeap[T]) Len() int {
	return len(h.items)
}

// Less returns whether the item at index i should sort before the item at index j.
func (h GenericHeap[T]) Less(i, j int) bool {
	return h.less(h.items[i], h.items[j])
}

// Swap swaps the items at the given indices.
func (h GenericHeap[T]) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
}

// Push adds an item to the heap.
func (h *GenericHeap[T]) Push(x any) {
	h.items = append(h.items, x.(T))
}

// Pop removes and returns the minimum item from the heap.
func (h *GenericHeap[T]) Pop() any {
	old := h.items
	n := len(old)
	item := old[n-1]
	h.items = old[0 : n-1]
	return item
}

// Peek returns the minimum item from the heap without removing it.
func (h *GenericHeap[T]) Peek() T {
	return h.items[0]
}

// Add adds an item to the heap.
func (h *GenericHeap[T]) Add(item T) {
	heap.Push(h, item)
}

// Remove removes and returns the minimum item from the heap.
func (h *GenericHeap[T]) Remove() T {
	return heap.Pop(h).(T)
}
