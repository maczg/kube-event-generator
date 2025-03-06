package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ResourceAdapter interface {
	metav1.Object
}

type ResourceOption func(ResourceAdapter)

func WithName(name string) ResourceOption {
	return func(c ResourceAdapter) {
		c.SetName(name)
	}
}
func WithNamespace(namespace string) ResourceOption {
	return func(c ResourceAdapter) {
		c.SetNamespace(namespace)
	}
}

func WithAnnotations(annotations map[string]string) ResourceOption {
	return func(c ResourceAdapter) {
		c.SetAnnotations(annotations)
	}
}

func WithLabels(labels map[string]string) ResourceOption {
	return func(c ResourceAdapter) {
		c.SetLabels(labels)
	}
}
