package scenario

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPlotPendingPodsOverTime(t *testing.T) {
	mc := NewMetricCollector()

	duration := 120
	for i := 0; i < duration; i++ {
		mc.RecordPendingQueueLength(i, time.Now().Add(-time.Duration(i)*time.Second))
	}
	// Call the method to plot pending pods over time
	err := mc.PlotPendingPodsOverTime()
	assert.NoError(t, err, "PlotPendingPodsOverTime should not return an error")
}
