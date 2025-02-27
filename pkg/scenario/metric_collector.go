package scenario

import (
	"errors"
	"fmt"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"sync"
	"time"
)

type MetricCollector struct {
	mu                 sync.Mutex
	podStart           map[string]time.Time
	podPendingDuration map[string]time.Duration
	pendingQueue       []struct {
		timestamp time.Time
		length    int
	}
}

func NewMetricCollector() *MetricCollector {
	return &MetricCollector{
		podStart:           make(map[string]time.Time),
		podPendingDuration: make(map[string]time.Duration),
		pendingQueue: make([]struct {
			timestamp time.Time
			length    int
		}, 0),
	}
}

func (m *MetricCollector) RecordPodStart(podName string, time time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.podStart[podName] = time
}

func (m *MetricCollector) RecordPodPendingEnd(podName string, end time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	start, ok := m.podStart[podName]
	if !ok {
		return
	}
	duration := end.Sub(start)
	m.podPendingDuration[podName] = duration
}

// RecordPendingQueueLength records the length of the pending queue at a given time.
func (m *MetricCollector) RecordPendingQueueLength(l int, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pendingQueue = append(m.pendingQueue, struct {
		timestamp time.Time
		length    int
	}{timestamp: t, length: l})
}

func (m *MetricCollector) Dump() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.pendingQueue) == 0 {
		return fmt.Errorf("no queue length data to plot")
	}

	// Plot pending history
	pendingPlot := plot.New()
	pendingPlot.Title.Text = "Pending Pods Over Time"
	pendingPlot.X.Label.Text = "Time"
	pendingPlot.Y.Label.Text = "Number of Pending Pods"

	pendingPoints := make(plotter.XYs, len(m.pendingQueue))
	for i, value := range m.pendingQueue {
		pendingPoints[i].X = float64(value.timestamp.Unix())
		pendingPoints[i].Y = float64(value.length)
	}

	var times []string
	for _, snapshot := range m.pendingQueue {
		times = append(times, snapshot.timestamp.Format("15:04:05"))
	}
	pendingPlot.NominalX(times...)

	pendingLine, err := plotter.NewLine(pendingPoints)
	if err != nil {
		return errors.New(fmt.Sprintf("could not create line plot: %v", err))
	}
	pendingPlot.Add(pendingLine)

	fileName := fmt.Sprintf("pending_history_%s.png", time.Now().Format("2006-01-02_15:04:05"))
	if err := pendingPlot.Save(10*vg.Inch, 4*vg.Inch, fileName); err != nil {
		return errors.New(fmt.Sprintf("could not save plot: %v", err))
	}

	// Plot pending duration distribution
	durationPlot := plot.New()
	durationPlot.Title.Text = "Pending Duration Distribution"
	durationPlot.X.Label.Text = "Time"
	durationPlot.Y.Label.Text = "Duration (seconds)"

	durationPoints := make(plotter.XYs, len(m.podPendingDuration))

	i := 0
	for _, value := range m.podPendingDuration {
		durationPoints[i].X = float64(i)
		durationPoints[i].Y = value.Seconds()
		i++
	}

	durationLine, err := plotter.NewLine(durationPoints)
	if err != nil {
		return errors.New(fmt.Sprintf("could not create line plot: %v", err))
	}
	durationPlot.Add(durationLine)

	fileName = fmt.Sprintf("pending_duration_dist_%s.png", time.Now().Format("2006-01-02_15:04:05"))
	if err := durationPlot.Save(10*vg.Inch, 4*vg.Inch, fileName); err != nil {
		return errors.New(fmt.Sprintf("could not save plot: %v", err))
	}
	return nil
}

func (m *MetricCollector) PlotPendingPodsOverTime() error {
	fileName := fmt.Sprintf("pending_pods_%s.png", time.Now().Format("2006-01-02_15:04:05"))
	pendingPlot := plot.New()

	pendingPlot.Title.Text = "Pending Pods Over Time"
	pendingPlot.X.Label.Text = "Time"
	pendingPlot.Y.Label.Text = "Number of Pending Pods"

	values := plotter.Values{}
	for _, value := range m.pendingQueue {
		values = append(values, float64(value.length))
	}

	bars, err := plotter.NewBarChart(values, vg.Points(1))
	if err != nil {
		return fmt.Errorf("could not create bar chart: %v", err)
	}
	pendingPlot.Add(bars)

	var xs []string
	for _, sample := range m.pendingQueue {
		xs = append(xs, sample.timestamp.Format("15:04:05"))
	}
	pendingPlot.NominalX(xs...)

	if err := pendingPlot.Save(10*vg.Inch, 4*vg.Inch, fileName); err != nil {
		return fmt.Errorf("could not save plot: %v", err)
	}
	return nil
}

func (m *MetricCollector) PlotPendingDuration() error {
	fileName := fmt.Sprintf("pending_duration_%s.png", time.Now().Format("2006-01-02_15:04:05"))
	durationPlot := plot.New()

	durationPlot.Title.Text = "Pending Duration"
	durationPlot.X.Label.Text = "Pod"
	durationPlot.Y.Label.Text = "Duration (seconds)"

	values := plotter.Values{}
	for _, value := range m.podPendingDuration {
		values = append(values, value.Seconds())
	}
	bar, err := plotter.NewBarChart(values, vg.Points(1))
	if err != nil {
		return fmt.Errorf("could not create bar chart: %v", err)
	}
	durationPlot.Add(bar)

	var xs []string
	for podName := range m.podPendingDuration {
		xs = append(xs, podName)
	}
	durationPlot.NominalX(xs...)
	if err := durationPlot.Save(10*vg.Inch, 4*vg.Inch, fileName); err != nil {
		return fmt.Errorf("could not save plot: %v", err)
	}
	return nil
}
