package container

import (
	"container/heap"
)

// Heap is a type that implements heap.Interface and holds items of any type.
type Heap[T any] struct {
	items []T
	less  func(a, b T) bool
	push  func(x T)
	pop   func() T
}

// NewHeap creates a new Heap.
func NewHeap[T any](less func(a, b T) bool) *Heap[T] {
	g := &Heap[T]{
		items: []T{},
		less:  less,
	}
	heap.Init(g)
	return g
}

// Add adds an item to the heap.
func (h *Heap[T]) Add(item T) { heap.Push(h, item) }

// Items returns the items in the heap.
func (h *Heap[T]) Items() []T { return h.items }

// Peek returns the minimum item from the heap without removing it.
func (h *Heap[T]) Peek() T { return h.items[0] }

// Remove removes and returns the minimum item from the heap.
func (h *Heap[T]) Remove() T { return heap.Pop(h).(T) }

// Len returns the number of items in the heap.
func (h *Heap[T]) Len() int { return len(h.items) }

// Less returns whether the item at index i should sort before the item at index j.
func (h *Heap[T]) Less(i, j int) bool {
	return h.less(h.items[i], h.items[j])
}

// Swap swaps the items at the given indices.
func (h *Heap[T]) Swap(i, j int) { h.items[i], h.items[j] = h.items[j], h.items[i] }

// Push adds an item to the heap.
func (h *Heap[T]) Push(x any) { h.items = append(h.items, x.(T)) }

// Pop removes and returns the minimum item from the heap.
func (h *Heap[T]) Pop() any {
	old := h.items
	n := len(old)
	item := old[n-1]
	h.items = old[0 : n-1]
	return item
}
