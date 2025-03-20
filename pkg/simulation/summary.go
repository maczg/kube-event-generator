package simulation

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"
)

func (s *Simulation) exportMetrics() {
	resultDir := fmt.Sprintf("%s/%s", s.resultDir, s.ID)

	if err := NodeResourceMetric.ExportCSV(resultDir, ""); err != nil {
		s.logger.Error("error exporting csv: %v", err)
	}
	if err := eventTimelineMetric.ExportCSV(resultDir, ""); err != nil {
		s.logger.Error("error exporting csv: %v", err)
	}
	if err := timeToSchedulePodMetric.ExportCSV(resultDir, ""); err != nil {
		s.logger.Error("error exporting csv: %v", err)
	}
	if err := pendingPodQueueMetric.ExportCSV(resultDir, ""); err != nil {
		s.logger.Error("error exporting csv: %v", err)
	}

	utilMap, errUtil := s.nodeUtilization(fmt.Sprintf("%s/node_resource_utilization.csv", resultDir))
	if errUtil != nil {
		s.logger.Error("error exporting cpu utilization: %v", errUtil)
	}
	summary := NewSummary(s.ID)

	for node, utilizationMap := range utilMap {
		for res, values := range utilizationMap {
			if _, ok := summary.NodeResourceUtilization[node]; !ok {
				summary.NodeResourceUtilization[node] = make(map[string]Stats)
			}
			summary.NodeResourceUtilization[node][res] = computeStats(values)
		}
	}

	if err := summary.Save(resultDir); err != nil {
		s.logger.Error("error saving summary: %v", err)
	}
}

type Stats struct {
	Count int
	Avg   float64
	Min   float64
	Max   float64
	P50   float64
	P90   float64
	P95   float64
	At    time.Time
}

type Summary struct {
	SimID                   string
	TimeToSchedulePod       Stats                       `json:"timeToSchedulePod"`
	NodeResourceUsage       map[string]map[string]Stats `json:"nodeResourceUsage"`
	NodeResourceUtilization map[string]map[string]Stats `json:"nodeResourceUtilization"`
}

func NewSummary(id string) *Summary {
	ttsFloat := make([]float64, 0, len(timeToSchedulePodMetric.Values()))
	for _, v := range timeToSchedulePodMetric.Values() {
		ttsFloat = append(ttsFloat, v.Value())
	}
	timeToSchedule := computeStats(ttsFloat)
	nodeUsageSummary := summarizeNodeResourceUsage()

	return &Summary{
		SimID:                   id,
		TimeToSchedulePod:       timeToSchedule,
		NodeResourceUsage:       nodeUsageSummary,
		NodeResourceUtilization: make(map[string]map[string]Stats),
	}
}

func (s *Summary) Save(dir string) error {
	if dir != "" {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
	} else {
		dir = "."
	}

	filename := fmt.Sprintf("%s/summary.json", dir)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filename, data, 0644); err != nil {
		return err
	}
	return nil
}

func computeStats(values []float64) Stats {
	if len(values) == 0 {
		return Stats{At: time.Now()}
	}

	sort.Float64s(values)
	sum := 0.0
	for _, v := range values {
		sum += v
	}

	avg := sum / float64(len(values))
	minVal := values[0]
	maxVal := values[len(values)-1]

	percentile := func(pct float64) float64 {
		if len(values) == 1 {
			return values[0]
		}
		idx := int((pct / 100.0) * float64(len(values)-1))
		return values[idx]
	}

	return Stats{
		Count: len(values),
		Avg:   avg,
		Min:   minVal,
		Max:   maxVal,
		P50:   percentile(50),
		P90:   percentile(90),
		P95:   percentile(95),
		At:    time.Now(),
	}
}

func summarizeNodeResourceUsage() map[string]map[string]Stats {
	// Outer map: node => (resource => []float64)
	usageMap := make(map[string]map[string][]float64)

	records := NodeResourceMetric.Values()
	for _, r := range records {
		node := r.Labels()["node"]
		res := r.Labels()["resource"]

		if _, ok := usageMap[node]; !ok {
			usageMap[node] = make(map[string][]float64)
		}
		usageMap[node][res] = append(usageMap[node][res], r.Value())
	}

	// Convert slices to SummaryStats
	summary := make(map[string]map[string]Stats)
	for node, resourceMap := range usageMap {
		summary[node] = make(map[string]Stats)
		for res, values := range resourceMap {
			summary[node][res] = computeStats(values)
		}
	}
	return summary
}

func (s *Simulation) nodeUtilization(fileName string) (map[string]map[string][]float64, error) {
	// Outer key: nodeName
	// Inner key: "cpu" / "memory"
	// Value: Slice of utilization values
	utilizationMap := make(map[string]map[string][]float64)

	file, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	headers := []string{"timestamp", "node", "cpu_utilization", "memory_utilization"}
	if err := w.Write(headers); err != nil {
		return nil, fmt.Errorf("could not write headers: %w", err)
	}

	for _, node := range s.Scenario.Cluster.Nodes {
		nodeName := node.Name

		cpus := NodeResourceMetric.GetValueByLabel(map[string]string{
			"node":     nodeName,
			"resource": "cpu",
		})
		mems := NodeResourceMetric.GetValueByLabel(map[string]string{
			"node":     nodeName,
			"resource": "memory",
		})

		if len(cpus) < 2 || len(mems) < 2 {
			continue
		}

		cpuCap := cpus[0].Value()
		memCap := mems[0].Value()

		var cpuUtilization []float64
		for _, cpuUsage := range cpus[1:] {
			diff := cpuCap - cpuUsage.Value()
			if diff <= 0 {
				cpuUtilization = append(cpuUtilization, 0)
				continue
			}
			cpuUtil := diff / cpuCap * 100
			cpuUtilization = append(cpuUtilization, cpuUtil)
		}

		var memUtilization []float64
		for _, memUsage := range mems[1:] {
			diff := memCap - memUsage.Value()
			if diff <= 0 {
				memUtilization = append(memUtilization, 0)
				continue
			}
			memUtil := diff / memCap * 100
			memUtilization = append(memUtilization, memUtil)
		}

		limit := len(cpuUtilization)
		if len(memUtilization) < limit {
			limit = len(memUtilization)
		}

		// Initialize the map entries for this node if not present
		if _, ok := utilizationMap[nodeName]; !ok {
			utilizationMap[nodeName] = make(map[string][]float64)
		}
		utilizationMap[nodeName]["cpu"] = cpuUtilization
		utilizationMap[nodeName]["memory"] = memUtilization

		for i := 0; i < limit; i++ {
			row := []string{
				cpus[i+1].Timestamp().String(),
				nodeName,
				fmt.Sprintf("%.2f", cpuUtilization[i]),
				fmt.Sprintf("%.2f", memUtilization[i]),
			}
			if err := w.Write(row); err != nil {
				return nil, fmt.Errorf("could not write CSV row: %w", err)
			}
		}
	}

	return utilizationMap, nil
}
