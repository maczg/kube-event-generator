package scenario

import "github.com/maczg/kube-event-generator/pkg/metric"

const (
	pendingQueueLengthMetricName = "pod_pending_queue_length"
	pendingPodDurationMetricName = "pod_pending_durations"
)

var PendingPodQueueMetric = metric.NewMetric(pendingQueueLengthMetricName)
var PendingPodDurationMetric = metric.NewMetric(pendingPodDurationMetricName)
