package metric

import (
	"encoding/csv"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

const TimestampFormat = "2006-01-02T15:04:05"

type InMemoryGaugeVec struct {
	metricName string
	gaugeVec   *prometheus.GaugeVec
	labels     Labels
	mu         sync.Mutex
	records    []Record
	lastValues map[string]float64
}

func NewInMemoryGaugeVec(opts prometheus.GaugeOpts, labels Labels) *InMemoryGaugeVec {
	g := &InMemoryGaugeVec{
		gaugeVec:   prometheus.NewGaugeVec(opts, labels),
		labels:     labels,
		records:    make([]Record, 0, 100),
		lastValues: make(map[string]float64),
		metricName: opts.Name,
	}
	return g
}

func (g *InMemoryGaugeVec) Set(value float64, labels Labels) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.gaugeVec.WithLabelValues(labels...).Set(value)

	key := labels.toMapKey()
	g.lastValues[key] = value

	g.gaugeVec.WithLabelValues(labels...).Set(value)

	labelsMap := make(map[string]string, len(g.labels))
	for i, lv := range labels {
		labelsMap[g.labels[i]] = lv
	}

	g.records = append(g.records, Record{
		timestamp: time.Now(),
		labels:    labelsMap,
		value:     value,
	})
}

// Add increments the gauge by `delta` for the given label set.
func (g *InMemoryGaugeVec) Add(delta float64, labels Labels) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := labels.toMapKey()
	oldVal := g.lastValues[key]
	newVal := oldVal + delta
	g.lastValues[key] = newVal

	g.gaugeVec.WithLabelValues(labels...).Set(newVal)

	// Record in history
	labelsMap := make(map[string]string, len(g.labels))
	for i, lv := range labels {
		labelsMap[g.labels[i]] = lv
	}

	g.records = append(g.records, Record{
		timestamp: time.Now(),
		labels:    labelsMap,
		value:     newVal,
	})
}

// Sub decrements the gauge by `delta` for the given label set.
func (g *InMemoryGaugeVec) Sub(delta float64, labels Labels) {
	g.Add(-delta, labels)
}

// Values returns a copy of all the stored records so external code
// can read them without affecting internal state.
func (g *InMemoryGaugeVec) Values() []Record {
	g.mu.Lock()
	defer g.mu.Unlock()

	cp := make([]Record, len(g.records))
	copy(cp, g.records)
	return cp
}

// Collect implement prometheus.Collector so we can register this struct
// directly with a prometheus.Registry.
func (g *InMemoryGaugeVec) Collect(ch chan<- prometheus.Metric) {
	g.gaugeVec.Collect(ch)
}

func (g *InMemoryGaugeVec) Describe(ch chan<- *prometheus.Desc) {
	g.gaugeVec.Describe(ch)
}

func (g *InMemoryGaugeVec) ExportCSV(base string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	file, err := os.Create(fmt.Sprintf("%s-%s.csv", base, g.metricName))
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			logrus.Errorf("error closing file: %v", err)
		}
	}(file)

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header
	header := []string{"timestamp"}
	uniqueLabels := make(map[string]string)
	for _, record := range g.records {
		for label := range record.labels {
			uniqueLabels[label] = label
		}
	}

	for label := range uniqueLabels {
		header = append(header, label)
	}
	header = append(header, "value")

	if err = writer.Write(header); err != nil {
		return err
	}

	records := g.records
	for _, record := range records {
		row := []string{
			record.timestamp.Format(TimestampFormat),
		}
		for _, label := range header[1 : len(header)-1] {
			row = append(row, record.labels[label])
		}
		row = append(row, fmt.Sprintf("%f", record.value))

		if err = writer.Write(row); err != nil {
			return err
		}
	}

	//header := []string{"timestamp", "labels", "value"}
	//if err = writer.Write(header); err != nil {
	//	return err
	//}
	//
	//// Write records
	//for _, record := range g.records {
	//	labels, err := json.Marshal(record.labels)
	//	if err != nil {
	//		return err
	//	}
	//	row := []string{
	//		record.timestamp.Format(time.RFC3339),
	//		string(labels),
	//		fmt.Sprintf("%f", record.value),
	//	}
	//	if err := writer.Write(row); err != nil {
	//		return err
	//	}
	//}

	return nil

}
