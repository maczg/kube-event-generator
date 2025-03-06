package utils

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNodeFactory_NewNode(t *testing.T) {
	tests := []struct {
		name string
		opts []ResourceOption
		want *corev1.Node
	}{
		// test cases
		{
			name: "WithStatus",
			opts: []ResourceOption{
				NodeWithStatus("test-name", "1", "10Gi", "110"),
			},
			want: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-name",
				},
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("10Gi"),
						corev1.ResourcePods:   resource.MustParse("110"),
					},
					Capacity: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("10Gi"),
						corev1.ResourcePods:   resource.MustParse("110"),
					},
				},
			},
		},
		{
			name: "NodeWithMetadata",
			opts: []ResourceOption{
				WithLabels(map[string]string{"key1": "value1"}),
				WithAnnotations(map[string]string{"annotation1": "value1"}),
			},
			want: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"key1": "value1"},
					Annotations: map[string]string{"annotation1": "value1"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NodeFactory.NewNode(tt.opts...)
			assert.Equal(t, tt.want, got)
		})
	}
}
