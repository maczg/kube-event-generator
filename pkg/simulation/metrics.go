package simulation

import (
	"github.com/maczg/kube-event-generator/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
)

var reg = prometheus.NewRegistry()

var NodeAllocatedResourceMetric = metric.NewInMemoryGaugeVec(
	prometheus.GaugeOpts{
		Name: "node_allocated_resource",
		Help: "Node allocated resource",
	},
	metric.Labels{"node", "resource"})

var EventTimelineMetric = metric.NewInMemoryGaugeVec(
	prometheus.GaugeOpts{
		Name: "event_timeline",
		Help: "Event timeline",
	},
	metric.Labels{"pod", "node", "status", "event_type", "request_cpu", "request_memory"})

func init() {
	reg.MustRegister(NodeAllocatedResourceMetric)
	reg.MustRegister(EventTimelineMetric)
}
