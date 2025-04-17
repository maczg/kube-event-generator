package util

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KubernetesObjectAdapter interface {
	metav1.Object
}

type KubernetesObjectOpt func(KubernetesObjectAdapter)

func WithName(name string) KubernetesObjectOpt {
	return func(c KubernetesObjectAdapter) {
		c.SetName(name)
	}
}
func WithNamespace(namespace string) KubernetesObjectOpt {
	return func(c KubernetesObjectAdapter) {
		c.SetNamespace(namespace)
	}
}

func WithAnnotations(annotations map[string]string) KubernetesObjectOpt {
	return func(c KubernetesObjectAdapter) {
		c.SetAnnotations(annotations)
	}
}

func WithLabels(labels map[string]string) KubernetesObjectOpt {
	return func(c KubernetesObjectAdapter) {
		c.SetLabels(labels)
	}
}
