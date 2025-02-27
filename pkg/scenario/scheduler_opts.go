package scenario

import (
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/metric"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"time"
)

type SchedulerOption func(*Scheduler)

func WithKubeClient() SchedulerOption {
	return func(s *Scheduler) {
		config, err := rest.InClusterConfig()
		if err != nil {
			logrus.Errorf("Failed to get in-cluster config: %v", err)
			// If running locally, you might use:
			kubeconfig := fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				logrus.Fatalf("Failed to get kubeconfig: %v", err)
			}
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			logrus.Fatalf("Failed to create clientset: %v", err)
		}
		s.KubeClient = clientset
	}
}

// WithDeadline sets the duration of the simulation.
// The manager will stop after the given number of seconds.
func WithDeadline(seconds int) SchedulerOption {
	return func(s *Scheduler) {
		go func() {
			<-s.startCh
			after := time.Duration(seconds) * time.Second
			<-time.After(after)
			s.Stop()
		}()
	}
}

// WithMetrics sets the metrics collector.
func WithMetricCollector(mc *metric.Collector) SchedulerOption {
	return func(s *Scheduler) {
		s.MetricCollector = mc
	}
}
