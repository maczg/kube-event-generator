package pkg

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Pod struct {
	P *corev1.Pod
}

type PodOption func(*Pod)

func NewPod(opts ...PodOption) *corev1.Pod {
	p := &Pod{
		P: &corev1.Pod{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p.P
}

func WitMetadata(name, namespace string) PodOption {
	return func(p *Pod) {
		p.P.ObjectMeta.Name = name
		p.P.ObjectMeta.Namespace = namespace
	}
}

func WithAffinity(requiredLabel map[string]string, softLabels map[string]string) PodOption {
	var requiredTerms []corev1.NodeSelectorRequirement
	var preferredTerms []corev1.PreferredSchedulingTerm

	affinity := &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{},
	}

	return func(p *Pod) {

		if requiredLabel != nil {
			for key, val := range requiredLabel {
				requiredTerms = append(requiredTerms, corev1.NodeSelectorRequirement{
					Key:      key,
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{val},
				})
			}
			affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: requiredTerms,
					},
				},
			}
		}

		if softLabels != nil {
			for key, val := range softLabels {
				preferredTerms = append(preferredTerms, corev1.PreferredSchedulingTerm{
					// TODO the weight for this preference
					Weight: 10, // the weight for this preference
					Preference: corev1.NodeSelectorTerm{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      key,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{val},
							},
						},
					},
				})
			}
			affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = preferredTerms
		}
		p.P.Spec.Affinity = affinity
	}
}

func WithLabels(labels map[string]string) PodOption {
	return func(p *Pod) {
		p.P.ObjectMeta.Labels = labels
	}
}

func WithContainer(name, image string) PodOption {
	return func(p *Pod) {
		p.P.Spec.Containers = append(p.P.Spec.Containers, corev1.Container{
			Name:  name,
			Image: image,
		})
	}
}

func WithResource(cpu, memory string) PodOption {
	return func(p *Pod) {
		WithContainer("server", "nginx:latest")(p)
		p.P.Spec.Containers[0].Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(memory),
			},
		}
	}
}
