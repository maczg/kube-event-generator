package simulation

import (
	"github.com/maczg/kube-event-generator/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
)

var reg = prometheus.NewRegistry()

var NodeResourceMetric = metric.NewInMemoryGaugeVec(
	prometheus.GaugeOpts{
		Name: "node_resource_usage",
		Help: "Node resource usage (with local history)",
	},
	[]string{"node", "resource"})

var eventTimelineMetric = metric.NewInMemoryGaugeVec(
	prometheus.GaugeOpts{
		Name: "event_timeline",
		Help: "Event timeline",
	},
	[]string{"pod", "status"})

var timeToSchedulePodMetric = metric.NewInMemoryGaugeVec(
	prometheus.GaugeOpts{
		Name: "time_to_schedule_pod",
		Help: "Time to schedule pod",
	},
	[]string{"pod"})

var pendingPodQueueMetric = metric.NewInMemoryGaugeVec(
	prometheus.GaugeOpts{
		Name: "pending_pods",
		Help: "Number of pending pods",
	},
	[]string{"status"})

func init() {
	reg.MustRegister(NodeResourceMetric)
	reg.MustRegister(eventTimelineMetric)
	reg.MustRegister(timeToSchedulePodMetric)
	reg.MustRegister(pendingPodQueueMetric)
}
