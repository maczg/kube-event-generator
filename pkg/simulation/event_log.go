package simulation

import (
	"bytes"
	"fmt"
	"time"
)

type EventLog struct {
	Timestamp       time.Time
	EventType       string // "PodCreated", "PodRunning", "PodEvicted"
	PodName         string
	NodeName        string
	CPURequest      string
	MemoryRequest   string
	PendingDuration time.Duration
	PendingQueue    int
}

type SimulationMetrics struct {
	Events            []EventLog
	NodeResourceUsage map[string]map[string]map[time.Time]string // Node -> {CPU, Memory}
	PendingQueue      int
}

func NewSimulationMetrics() *SimulationMetrics {
	return &SimulationMetrics{
		Events:            make([]EventLog, 0),
		NodeResourceUsage: make(map[string]map[string]map[time.Time]string),
	}
}

func (m *SimulationMetrics) AddNodeResourceUsage(nodeName, cpu, memory string) {
	if _, ok := m.NodeResourceUsage[nodeName]; !ok {
		m.NodeResourceUsage[nodeName] = make(map[string]map[time.Time]string)
	}
	if _, ok := m.NodeResourceUsage[nodeName]["CPU"]; !ok {
		m.NodeResourceUsage[nodeName]["CPU"] = make(map[time.Time]string)
	}
	if _, ok := m.NodeResourceUsage[nodeName]["Memory"]; !ok {
		m.NodeResourceUsage[nodeName]["Memory"] = make(map[time.Time]string)
	}
	m.NodeResourceUsage[nodeName]["CPU"][time.Now()] = cpu
	m.NodeResourceUsage[nodeName]["Memory"][time.Now()] = memory
}

func (m *SimulationMetrics) Status() {
	var buffer bytes.Buffer
	buffer.WriteString("[metrics] status:\n")
	for nodeName, resources := range m.NodeResourceUsage {
		cpu := resources["CPU"]
		memory := resources["Memory"]
		//logrus.Infof("Node %s: CPU: %s, Memory: %s", nodeName, cpu, memory)
		buffer.WriteString(fmt.Sprintf("Node %s: CPU: %s, Memory: %s \n", nodeName, cpu, memory))
	}
	buffer.WriteString(fmt.Sprintf("Pending Queue: %d\n", m.PendingQueue))
}
