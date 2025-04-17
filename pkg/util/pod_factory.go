package util

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var PodFactory = &podFactory{}

type podFactory struct{}

type pod v1.Pod

func PodWithMetadata(name, namespace string, labels, annotations map[string]string) KubernetesObjectOpt {
	return func(c KubernetesObjectAdapter) {
		c.SetName(name)
		c.SetNamespace(namespace)
		c.SetLabels(labels)
		c.SetAnnotations(annotations)
	}
}

func PodWithContainer(name, image, cpu, memory string) KubernetesObjectOpt {
	return func(obj KubernetesObjectAdapter) {
		if innerPod, ok := obj.(*pod); ok {
			container := v1.Container{
				Name:  name,
				Image: image,
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse(cpu),
						v1.ResourceMemory: resource.MustParse(memory),
					},
				},
			}
			innerPod.Spec.Containers = append(innerPod.Spec.Containers, container)
		}
	}
}

func PodWithNodeAffinity(requiredLabel map[string]string, softLabels map[string]string) KubernetesObjectOpt {
	return func(obj KubernetesObjectAdapter) {
		if innerPod, ok := obj.(*pod); ok {
			var requiredTerms []v1.NodeSelectorRequirement
			var preferredTerms []v1.PreferredSchedulingTerm

			affinity := &v1.Affinity{
				NodeAffinity: &v1.NodeAffinity{},
			}
			if requiredLabel != nil {
				for key, val := range requiredLabel {
					requiredTerms = append(requiredTerms, v1.NodeSelectorRequirement{
						Key:      key,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{val},
					})
				}
				affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &v1.NodeSelector{
					NodeSelectorTerms: []v1.NodeSelectorTerm{
						{
							MatchExpressions: requiredTerms,
						},
					},
				}
			}

			if softLabels != nil {
				for key, val := range softLabels {
					preferredTerms = append(preferredTerms, v1.PreferredSchedulingTerm{
						// TODO the weight for this preference
						Weight: 10, // the weight for this preference
						Preference: v1.NodeSelectorTerm{
							MatchExpressions: []v1.NodeSelectorRequirement{
								{
									Key:      key,
									Operator: v1.NodeSelectorOpIn,
									Values:   []string{val},
								},
							},
						},
					})
				}
				affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = preferredTerms
			}
			innerPod.Spec.Affinity = affinity
		}
	}
}

func (p *podFactory) NewPod(opts ...KubernetesObjectOpt) *v1.Pod {
	newPod := &pod{}

	for _, opt := range opts {
		opt(newPod)
	}
	return (*v1.Pod)(newPod)
}
