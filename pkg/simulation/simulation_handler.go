package simulation

import (
	"github.com/maczg/kube-event-generator/pkg/metric"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func (s *Simulation) handlePodEvent(e watch.Event, runningPod map[string]bool) {
	p, ok := e.Object.(*corev1.Pod)
	if !ok {
		return
	}

	switch e.Type {
	case watch.Added:
		s.logger.Info("pod %s %s - status %s", p.Name, e.Type, p.Status.Phase)
		eventTimelineMetric.Set(1, metric.WithLabels(p.Name, "added"))
	case watch.Modified:
		s.handlePodModified(p, runningPod)
	case watch.Deleted:
		s.handlePodDeleted(p)
	}
}

func (s *Simulation) handlePodModified(p *corev1.Pod, runningPod map[string]bool) {
	if p.Status.Phase == corev1.PodPending {
		s.logger.Info("pod %s %s - status %s", p.Name, watch.Modified, p.Status.Phase)
		eventTimelineMetric.Set(1, metric.WithLabels(p.Name, "pending"))
	}
	if p.Status.Phase == corev1.PodRunning && p.ObjectMeta.DeletionTimestamp == nil {
		if _, exists := runningPod[p.Name]; !exists {
			s.logger.Info("pod %s %s - status %s", p.Name, watch.Modified, p.Status.Phase)
			runningPod[p.Name] = true
			s.updateNodeResourceMetrics(p, -1)
			eventTimelineMetric.Set(1, metric.WithLabels(p.Name, "running"))
		}
	}
}

func (s *Simulation) handlePodDeleted(p *corev1.Pod) {
	s.logger.Info("pod %s %s - status %s, was in node %s", p.Name, watch.Deleted, p.Status.Phase, p.Spec.NodeName)
	s.updateNodeResourceMetrics(p, 1)
	s.removePodFromMap(p.Name)
}

func (s *Simulation) updateNodeResourceMetrics(p *corev1.Pod, factor float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	nodeName := p.Spec.NodeName
	cpu := p.Spec.Containers[0].Resources.Requests.Cpu().AsApproximateFloat64() * factor
	mem := p.Spec.Containers[0].Resources.Requests.Memory().AsApproximateFloat64() * factor

	NodeResourceMetric.Add(cpu, metric.WithLabels(nodeName, "cpu"))
	NodeResourceMetric.Add(mem, metric.WithLabels(nodeName, "memory"))
}

func (s *Simulation) removePodFromMap(podName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, pod := range s.podMap {
		if pod == podName {
			s.logger.Info("pod %s is done", podName)
			s.podMap = append(s.podMap[:i], s.podMap[i+1:]...)
			break
		}
	}
	s.logger.Info("queue len %d", len(s.podMap))
}
