package pkg

import "time"

// EventQueue is a min-heap that pops the soonest event first.
type EventQueue []IEvent

func (eq *EventQueue) First() IEvent {
	return (*eq)[0]
}
func (eq *EventQueue) Empty() bool {
	return len(*eq) == 0
}
func (eq *EventQueue) Last() IEvent {
	return (*eq)[len(*eq)-1]
}
func (eq *EventQueue) Len() int {
	return len(*eq)
}
func (eq *EventQueue) Less(i, j int) bool {
	return (*eq)[i].At().Before((*eq)[j].At())
}
func (eq *EventQueue) Swap(i, j int) {
	(*eq)[i], (*eq)[j] = (*eq)[j], (*eq)[i]
}

func (eq *EventQueue) Push(x interface{}) {
	*eq = append(*eq, x.(IEvent))
}
func (eq *EventQueue) Pop() interface{} {
	old := *eq
	n := len(old)
	item := old[n-1]
	*eq = old[0 : n-1]
	return item
}

func (eq *EventQueue) HappenNow() IEvent {
	if time.Now().After(eq.First().At()) {
		return eq.Pop().(IEvent)
	}
	return nil
}
