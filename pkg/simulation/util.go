package simulation

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

type Record struct {
	Value float64
	time.Time
}

func (r Record) String() string {
	return fmt.Sprintf("[time: %s, value: %f]", r.Time, r.Value)
}

// NodeResource is a map of node name to resource type to resource value
type NodeResource map[string]map[string][]Record

func NewNodeResource() NodeResource {
	return make(NodeResource)
}

func (nr NodeResource) AddNode(name string) {
	nr[name] = make(map[string][]Record)

}

func (nr NodeResource) AddResource(name, resource string, value float64) {
	nr[name][resource] = append(nr[name][resource], Record{Value: value, Time: time.Now()})
}

func (nr NodeResource) LastRecord(name, resource string) Record {
	return nr[name][resource][len(nr[name][resource])-1]
}

func (nr NodeResource) Status() {
	var buffer bytes.Buffer
	buffer.WriteString("[metrics] status:\n")
	for nodeName, resources := range nr {
		for resource, records := range resources {
			buffer.WriteString(nodeName + " " + resource + " ")
			for _, record := range records {
				buffer.WriteString(record.String())
			}
			buffer.WriteString("\n")
		}
	}
	logrus.Info(buffer.String())
	//for nodeName, resources := range m.NodeResourceUsage {
	//	cpu := resources["CPU"]
	//	memory := resources["Memory"]
	//	//logrus.Infof("Node %s: CPU: %s, Memory: %s", nodeName, cpu, memory)
	//	buffer.WriteString(fmt.Sprintf("Node %s: CPU: %s, Memory: %s \n", nodeName, cpu, memory))
	//}
	//buffer.WriteString(fmt.Sprintf("Pending Queue: %d\n", m.PendingQueue))
}
