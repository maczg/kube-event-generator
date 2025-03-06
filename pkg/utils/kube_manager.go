package utils

import "k8s.io/client-go/kubernetes"

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
