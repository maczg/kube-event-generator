package scenario

type EventQueue []*Event

func (eq *EventQueue) First() *Event {
	return (*eq)[0]
}
func (eq *EventQueue) Empty() bool {
	return len(*eq) == 0
}
func (eq *EventQueue) Last() *Event {
	return (*eq)[len(*eq)-1]
}
func (eq *EventQueue) Len() int {
	return len(*eq)
}
func (eq *EventQueue) Less(i, j int) bool {
	return (*eq)[i].RunAfter < (*eq)[j].RunAfter
}

func (eq *EventQueue) Swap(i, j int) {
	(*eq)[i], (*eq)[j] = (*eq)[j], (*eq)[i]
}

func (eq *EventQueue) Push(x interface{}) {
	*eq = append(*eq, x.(*Event))
}
func (eq *EventQueue) Pop() interface{} {
	old := *eq
	n := len(old)
	item := old[n-1]
	*eq = old[0 : n-1]
	return item
}
