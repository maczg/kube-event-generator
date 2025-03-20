package utils

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

type KubernetesManager struct {
	clientset *kubernetes.Clientset
	restCfg   *rest.Config
	nodeFactory
	podFactory
}

func NewKubernetesManager(clientset *kubernetes.Clientset, restCfg *rest.Config) *KubernetesManager {
	return &KubernetesManager{
		clientset: clientset,
		restCfg:   restCfg,
	}
}

func (m *KubernetesManager) RestCfg() *rest.Config {
	return m.restCfg
}

func (m *KubernetesManager) Clientset() *kubernetes.Clientset {
	return m.clientset
}

func MakeClientSet() (*kubernetes.Clientset, *rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		logrus.Warnf("%v", err)
		kubeconfig := fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get kubeconfig: %v", err)
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create clientset: %v", err)
	}
	return clientset, config, nil
}

func (m *KubernetesManager) ResetNodes() error {
	nodes, err := m.nodeFactory.GetNodes(context.Background(), *m.clientset)
	if err != nil {
		return fmt.Errorf("failed to get nodes: %v", err)
	}
	items := nodes.Items
	for _, n := range items {
		err = m.nodeFactory.DeleteNode(context.Background(), *m.clientset, n.Name)
		if err != nil {
			logrus.Errorf("failed to delete node %s: %v", n.Name, err)
		}
	}
	return nil
}

func (m *KubernetesManager) ResetPods() error {
	pods, err := m.podFactory.ListPods(context.Background(), *m.clientset, "")
	if err != nil {
		return fmt.Errorf("failed to get pods: %v", err)
	}
	items := pods.Items
	for _, p := range items {
		err = m.podFactory.DeletePod(context.Background(), *m.clientset, &p)
		if err != nil {
			logrus.Errorf("failed to delete pod %s: %v", p.Name, err)
		}
	}
	return nil
}
