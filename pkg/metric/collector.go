package metric

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

const ResultDir = "results/"

type Collector struct {
	mu        sync.Mutex
	Metrics   map[string]*Metric
	ResultDir string
}

func NewCollector(opts ...CollectorOpts) *Collector {
	c := &Collector{
		Metrics:   make(map[string]*Metric),
		ResultDir: ResultDir,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.makeResultDir()
	return c
}

func (c *Collector) makeResultDir() {
	if c.ResultDir == "" {
		c.ResultDir = ResultDir
	}
	err := os.Mkdir(c.ResultDir, os.ModePerm)
	if err != nil {
		if !os.IsExist(err) {
			logrus.Errorf("could not create results directory: %v", err)
		}
	}
}

func (c *Collector) WithMetric(m *Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.Metrics[m.Name]; ok {
		logrus.Warnf("metric %s already exists", m.Name)
	}
	c.Metrics[m.Name] = m
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

func (c *Collector) UpsertRecord(name string, value float64, timestamp *time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	metric, ok := c.Metrics[name]
	if !ok {
		metric = NewMetric(name)
		c.Metrics[name] = metric
	}
	metric.AddRecord(value, timestamp)
	return nil
}

func (c *Collector) Dump() {
	for _, metric := range c.Metrics {
		err := metric.Dump(c.ResultDir)
		if err != nil {
			logrus.Errorf("could not dump metric %s: %v", metric.Name, err)
		}
	}
}
