package utils

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

type KubernetesManager struct {
	clientset *kubernetes.Clientset
	nodeFactory
	podFactory
}

func NewKubernetesManager(clientset *kubernetes.Clientset) *KubernetesManager {
	return &KubernetesManager{
		clientset: clientset,
	}
}

func (m *KubernetesManager) Clientset() *kubernetes.Clientset {
	return m.clientset
}

func MakeClientSet() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		logrus.Warnf("%v", err)
		kubeconfig := fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubeconfig: %v", err)
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}
	return clientset, nil
}
