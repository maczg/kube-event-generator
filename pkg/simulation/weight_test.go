package simulation

import (
	"github.com/maczg/kube-event-generator/pkg/scheduler"
	"github.com/maczg/kube-event-generator/pkg/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewChangeWeightEvent(t *testing.T) {
	s := scheduler.New()

	weights := map[string]int{
		"NodeResourcesFit":                5,
		"NodeResourcesBalancedAllocation": 10,
	}
	// Create a new ChangeWeightEvent
	e := NewChangeWeightEvent(time.Duration(5)*time.Second, weights)

	s.Schedule(e)

	go func() {
		err := s.Start()
		if err != nil {
			t.Errorf("Failed to start scheduler: %v", err)
		}
	}()

	utils.WaitStopAndExecute(func() {
		err := s.Stop()
		assert.NoErrorf(t, err, "Failed to stop scheduler: %v", err)
	})
}
