package pkg

import (
	"context"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// PodMetrics tracks creation time, scheduling time, and derived metrics.
type PodMetrics struct {
	PodNamespace  string
	PodName       string
	CreationTime  time.Time
	ScheduledTime time.Time

	// We'll compute derived metrics, like PendingDuration, once the pod is scheduled.
	PendingDuration time.Duration
}

// PendingQueueObservation is a snapshot of how many pods are pending at a specific time.
type PendingQueueObservation struct {
	Timestamp    time.Time
	PendingCount int
}

// StartMonitoring periodically counts how many pods are pending and stores that in memory
func (m *Manager) StartMonitoring(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// record how many pods are in Pending phase
				m.recordPendingPods()
			case <-m.StopCh:
				return
			}
		}
	}()
}

// recordPendingPods queries the cluster for all pods in Pending state and logs/stores the count
func (m *Manager) recordPendingPods() {
	pods, err := m.kubeClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logrus.Errorf("Error listing pods to recordPendingPods: %v", err)
		return
	}

	pendingCount := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodPending {
			pendingCount++
		}
	}

	obs := PendingQueueObservation{
		Timestamp:    time.Now(),
		PendingCount: pendingCount,
	}

	m.metricsMu.Lock()
	m.pqObservations = append(m.pqObservations, obs)
	m.metricsMu.Unlock()
	logrus.Debugf("PendingQueueSize at %s = %d", obs.Timestamp.Format("15:04:05"), obs.PendingCount)
}

// DumpMetrics prints or returns the collected metrics for analysis.
// In a real system, you might write to a file or a DB.
func (m *Manager) DumpMetrics() {
	m.metricsMu.Lock()
	defer m.metricsMu.Unlock()

	logrus.Info("=== Pod Pending Times ===")
	for _, pm := range m.podMetrics {
		logrus.Infof("Pod %s/%s: Pending=%.2fs",
			pm.PodNamespace, pm.PodName, pm.PendingDuration.Seconds())
	}

	logrus.Info("=== Pending Queue Observations ===")
	for _, obs := range m.pqObservations {
		logrus.Infof("Time=%s PendingCount=%d", obs.Timestamp.Format("15:04:05"), obs.PendingCount)
	}
}
