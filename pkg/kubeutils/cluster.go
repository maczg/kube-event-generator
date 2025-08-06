package kubeutils

import (
	"context"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ResetCluster() error {
	logger.Default().Infof("resetting cluster state...")
	ctx := context.Background()

	if err := ResetPods(ctx, logger.Default()); err != nil {
		return fmt.Errorf("failed to reset pods: %w", err)
	}

	if err := ResetNodes(ctx, logger.Default()); err != nil {
		return fmt.Errorf("failed to reset nodes: %w", err)
	}

	if err := ResetKubeSchedulerWeights(ctx, logger.Default(), "http://localhost:1212/api/v1/schedulerconfiguration"); err != nil {
		return fmt.Errorf("failed to reset scheduler weights: %w", err)
	}

	logger.Default().Infof("cluster state reset successfully.")
	return nil
}

func CreateClusterWithNodes(nodes []*v1.Node) error {
	logger.Default().Infof("creating cluster with %d nodes...", len(nodes))
	clientset, err := GetClientset()
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}
	for _, n := range nodes {
		logger.Default().Debugf("creating node: %s", n.Name)
		if _, err := clientset.CoreV1().Nodes().Create(context.TODO(), n, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create node %s: %w", n.Name, err)
		}
	}
	logger.Default().Infof("cluster created with %d nodes.", len(nodes))
	return nil
}
