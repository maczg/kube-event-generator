package scheduler

import "testing"

func TestHeapInt(t *testing.T) {
	h := NewQueue[int](func(a, b int) bool {
		return a < b
	})
	inputs := []int{5, 3, 8, 1, 2}
	for _, v := range inputs {
		h.Add(v)
	}
	if got := h.Peek(); got != 1 {
		t.Errorf("Peek() = %d; want %d", got, 1)
	}
	expected := []int{1, 2, 3, 5, 8}
	for i, want := range expected {
		if got := h.Remove(); got != want {
			t.Errorf("Remove() #%d = %d; want %d", i, got, want)
		}
	}
	if h.Len() != 0 {
		t.Errorf("Len() = %d; want 0", h.Len())
	}
}

func TestHeapString(t *testing.T) {
	h := NewQueue[string](func(a, b string) bool {
		return a < b
	})
	inputs := []string{"banana", "apple", "cherry", "date"}
	for _, v := range inputs {
		h.Add(v)
	}
	if got := h.Peek(); got != "apple" {
		t.Errorf("Peek() = %s; want %s", got, "apple")
	}
	expected := []string{"apple", "banana", "cherry", "date"}
	for i, want := range expected {
		if got := h.Remove(); got != want {
			t.Errorf("Remove() #%d = %s; want %s", i, got, want)
		}
	}
}
