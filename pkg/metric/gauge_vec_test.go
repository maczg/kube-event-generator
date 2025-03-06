package metric

import (
	"encoding/csv"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestExportCSV(t *testing.T) {
	labels := Labels{"node", "resource"}
	dir := "results"
	fileName := fmt.Sprintf("test_node_resource_usage-%s", time.Now().Format("15-04-05"))
	fullPath := fmt.Sprintf("%s/%s-node_resource_usage.csv", dir, fileName)

	gaugeVec := NewInMemoryGaugeVec(prometheus.GaugeOpts{
		Name: "node_resource_usage",
		Help: "Node resource usage (with local history)",
	}, labels)

	// Add some records
	gaugeVec.Set(10.5, Labels{"node1", "cpu"})
	gaugeVec.Set(20.0, Labels{"node2", "memory"})
	gaugeVec.Set(15.0, Labels{"node1", "memory"})

	err := gaugeVec.ExportCSV(dir, fileName)

	assert.NoError(t, err)

	file, err := os.Open(fullPath)
	assert.NoError(t, err)
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	assert.NoError(t, err)

	expectedHeader := []string{"timestamp", "node", "resource", "value"}
	assert.ElementsMatch(t, expectedHeader, records[0])
	assert.Equal(t, 4, len(records)) // 1 header + 3 records
	assert.NoError(t, err)
}
