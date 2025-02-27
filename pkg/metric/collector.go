package metric

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gonum.org/v1/plot/vg"
	"os"
	"sync"
	"time"
)

const ResultPath = "results/"

type Collector struct {
	mu        sync.Mutex
	Metrics   map[string]*Metric
	ResultDir string
}

func NewCollector(opts ...CollectorOpts) *Collector {
	c := &Collector{
		Metrics:   make(map[string]*Metric),
		ResultDir: ResultPath,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.makeResultDir()
	return c
}

func (c *Collector) makeResultDir() {
	if c.ResultDir == "" {
		c.ResultDir = ResultPath
	}
	err := os.Mkdir(c.ResultDir, os.ModePerm)
	if err != nil {
		logrus.Errorf("could not create results directory: %v", err)
	}
}

func (c *Collector) AddMetric(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Metrics[name] = NewMetric(name)
}

func (c *Collector) AddRecord(name string, value float64, timestamp *time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	metric, ok := c.Metrics[name]
	if !ok {
		return fmt.Errorf("metric %s not found", name)
	}
	metric.AddRecord(value, timestamp)
	return nil
}

// TODO refactor me
func (c *Collector) Dump() {
	for _, metric := range c.Metrics {
		metric.Dump()
		// TODO refactor
		if metric.Name == "pending_queue_length" {
			plt := metric.GetLineChart("Pending Pods Over Time", "Time", "Number of Pending Pods")
			if plt != nil {
				err := plt.Save(10*vg.Inch, 4*vg.Inch, fmt.Sprintf("%s/pending_pods_%s.png", c.ResultDir, time.Now().Format("2006-01-02_15:04:05")))
				if err != nil {
					logrus.Errorf("could not save plot: %v", err)
				}
			}
		}
	}
}
