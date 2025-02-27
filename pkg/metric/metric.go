package metric

import (
	"fmt"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
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

func (m *Metric) Dump() {
	m.mu.Lock()
	defer m.mu.Unlock()
	fmt.Printf("Metric: %s\n", m.Name)
	for _, record := range m.Values.items {
		fmt.Printf("%v: %v\n", record.timestamp, record.value)
	}
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

func (m *Metric) GetBarChart(name, x, y string) *plot.Plot {
	m.mu.Lock()
	defer m.mu.Unlock()

	pts := make(plotter.Values, len(m.Values.items))
	for i, record := range m.Values.items {
		pts[i] = record.value
	}

	bar, err := plotter.NewBarChart(pts, vg.Points(1))
	if err != nil {
		fmt.Printf("could not create bar chart: %v\n", err)
		return nil
	}

	plt := newPlot(WithTitle(name), WithLabels(x, y))
	plt.Add(bar)
	return plt
}

func (m *Metric) GetLineChart(name, x, y string) *plot.Plot {
	m.mu.Lock()
	defer m.mu.Unlock()

	pts := make(plotter.XYs, len(m.Values.items))
	for i, record := range m.Values.items {
		pts[i].X = float64(record.timestamp.Unix())
		pts[i].Y = record.value
	}

	line, err := plotter.NewLine(pts)
	if err != nil {
		fmt.Printf("could not create line: %v\n", err)
		return nil
	}

	plt := newPlot(WithTitle(name), WithLabels(x, y))
	plt.Add(line)
	return plt
}
