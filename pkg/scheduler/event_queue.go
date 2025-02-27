package scheduler

import (
	"container/heap"
	"github.com/maczg/kube-event-generator/pkg/container"
)

var less = func(a, b Event) bool {
	return a.At() < b.At()
}

type EventQueue struct {
	container.Heap[Event]
}

func NewEventQueue() *EventQueue {
	q := container.NewHeap[Event](less)
	heap.Init(q)
	return &EventQueue{
		Heap: *q,
	}
}
