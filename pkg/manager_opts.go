package pkg

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"time"
)

type ManagerOption func(*Manager)

func WithKubeClient() ManagerOption {
	return func(m *Manager) {
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
		m.kubeClient = clientset
	}
}

func WithSimulationEnd(endTime int) ManagerOption {
	return func(m *Manager) {
		after := time.Duration(endTime) * time.Second
		go func() {
			<-time.After(after)
			m.Stop()
		}()
	}
}
