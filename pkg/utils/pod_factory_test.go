package utils

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestPodFactory_NewPod(t *testing.T) {
	tests := []struct {
		name string
		opts []ResourceOption
		want *corev1.Pod
	}{
		{
			name: "WithAffinity",
			opts: []ResourceOption{
				PodWithNodeAffinity(map[string]string{"key1": "value1"}, map[string]string{"key2": "value2"}),
			},
			want: &corev1.Pod{
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "key1",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{"value1"},
											},
										},
									},
								},
							},
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
								{
									Weight: 10,
									Preference: corev1.NodeSelectorTerm{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "key2",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{"value2"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "WithLabels",
			opts: []ResourceOption{
				WithLabels(map[string]string{"key1": "value1"}),
			},
			want: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"key1": "value1"},
				},
			},
		},
		{
			name: "WithAnnotations",
			opts: []ResourceOption{
				WithAnnotations(map[string]string{"annotation1": "value1"}),
			},
			want: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"annotation1": "value1"},
				},
			},
		},
		{
			name: "PodWithMetadata",
			opts: []ResourceOption{
				PodWithMetadata("test-name", "test-namespace", map[string]string{"key1": "value1"}, map[string]string{"annotation1": "value1"}),
			},
			want: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-name",
					Namespace:   "test-namespace",
					Labels:      map[string]string{"key1": "value1"},
					Annotations: map[string]string{"annotation1": "value1"},
				},
			},
		},
		{
			name: "PodWithContainer",
			opts: []ResourceOption{
				PodWithContainer("test-container", "test-image", "100m", "200Mi"),
			},
			want: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("200Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newPod := PodFactory.NewPod(tt.opts...)
			assert.Equal(t, tt.want, newPod)
		})
	}
}
