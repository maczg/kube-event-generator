package kubernetes

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetRestConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		logrus.Warnf("%v", err)

		kubeconfig := fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubeconfig: %v", err)
		}
	}

	return config, nil
}

func GetClientset() (*kubernetes.Clientset, error) {
	re, err := GetRestConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(re)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %v", err)
	}

	return clientset, nil
}
