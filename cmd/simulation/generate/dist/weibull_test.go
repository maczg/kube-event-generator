package dist

import (
	"math/rand"
	"testing"
)

func TestWeibull_Next(t *testing.T) {
	r := rand.New(rand.NewSource(0))
	w := NewWeibull(r, 1.5, 512)
	for i := 0; i < 10; i++ {
		t.Log(w.Next())
	}
}
