package metric

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

// for the pending duration of each pod we should add the duration on the Creation event
// on creation, when a pod is scheduled (running) make the time delta now - p.creationTime and store it

type Metric struct {
	mu     sync.Mutex
	Name   string
	Values *GenericHeap[Record]
}

func NewMetric(name string) *Metric {
	return &Metric{
		Name:   name,
		Values: NewGenericHeap[Record](less),
	}
}

func (m *Metric) Dump(outputDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	filepath := fmt.Sprintf("%s/%s_%s.csv", outputDir, m.Name, time.Now().Format("15:04:05")) + ".csv"
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("could not create file: %v\n", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err = writer.Write([]string{"timestamp", "value"}); err != nil {
		return fmt.Errorf("could not write header to file: %v\n", err)
	}
	for _, record := range m.Values.items {
		line := []string{
			record.timestamp.Format("15:04:05"),
			strconv.FormatFloat(record.value, 'f', -1, 64),
		}
		if err = writer.Write(line); err != nil {
			return err
		}
	}
	return writer.Error()
}

func (m *Metric) addRecord(r Record) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Values.Push(r)
}

func (m *Metric) AddRecord(value float64, timestamp *time.Time) {
	r := Record{}
	if timestamp == nil {
		r = Record{timestamp: time.Now(), value: value}
	} else {
		r = Record{timestamp: *timestamp, value: value}
	}
	m.addRecord(r)
}
