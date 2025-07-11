package kubernetes

import (
	"context"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ResetPods(ctx context.Context, logger *logger.Logger) error {
	clientset, err := GetClientset()
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}
	logger.Infof("Resetting pods in cluster...")
	// Delete all pods in all namespaces
	deletePolicy := metav1.DeletePropagationForeground
	deleteOpts := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %w", err)
	}
	for _, ns := range namespaces.Items {
		if isSystemNamespace(ns.Name) {
			logger.Debugf("Skipping system namespace: %s", ns.Name)
			continue
		}
		logger.Debugf("Deleting pods in namespace: %s", ns.Name)

		if err := clientset.CoreV1().Pods(ns.Name).DeleteCollection(ctx, deleteOpts, metav1.ListOptions{}); err != nil {
			logger.Warnf("Failed to delete pods in namespace %s: %v", ns.Name, err)
		}
	}
	return nil
}

// ResetKubeSchedulerWeights resets the weights of all scheduler plugins to their default values.
func ResetKubeSchedulerWeights(ctx context.Context, logger *logger.Logger, schedulerUrl string) error {
	logger.Infof("Resetting scheduler plugin weights...")
	schedulerManager := NewHTTPKubeSchedulerManager(schedulerUrl)
	if err := schedulerManager.ResetToDefaults(ctx); err != nil {
		return err
	}
	return nil
}
