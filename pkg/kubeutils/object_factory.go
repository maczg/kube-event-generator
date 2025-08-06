package kubeutils

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/apiserver/pkg/storage/names"
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

func NodeWithMetadata(name string, labels, annotations map[string]string) KubernetesObjectOpt {
	return func(c KubernetesObjectAdapter) {
		c.SetName(name)
		c.SetLabels(labels)
		c.SetAnnotations(annotations)
	}
}

func NodeWithTaints(taints []v1.Taint) KubernetesObjectOpt {
	return func(obj KubernetesObjectAdapter) {
		if innerNode, ok := obj.(*node); ok {
			innerNode.Spec.Taints = taints
		}
	}
}

func NodeWithResource(cpu, memory, pods string) KubernetesObjectOpt {
	return func(obj KubernetesObjectAdapter) {
		if innerNode, ok := obj.(*node); ok {
			innerNode.Status.Capacity = v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse(cpu),
				v1.ResourceMemory: resource.MustParse(memory),
				v1.ResourcePods:   resource.MustParse(pods),
			}
			innerNode.Status.Allocatable = innerNode.Status.Capacity
		}
	}
}

func NodeWithLabels(labels map[string]string) KubernetesObjectOpt {
	return func(obj KubernetesObjectAdapter) {
		if innerNode, ok := obj.(*node); ok {
			if innerNode.ObjectMeta.Labels == nil {
				innerNode.ObjectMeta.Labels = make(map[string]string)
			}
			for key, value := range labels {
				innerNode.ObjectMeta.Labels[key] = value
			}
		}
	}
}

func NodeWithAnnotations(annotations map[string]string) KubernetesObjectOpt {
	return func(obj KubernetesObjectAdapter) {
		if innerNode, ok := obj.(*node); ok {
			if innerNode.ObjectMeta.Annotations == nil {
				innerNode.ObjectMeta.Annotations = make(map[string]string)
			}
			for key, value := range annotations {
				innerNode.ObjectMeta.Annotations[key] = value
			}
		}
	}
}

// ObjectFactory is a singleton that provides methods to create Kubernetes objects like Pods and Nodes.
var ObjectFactory = &objectFactory{}

type objectFactory struct{}

type pod v1.Pod
type node v1.Node

func (p *objectFactory) NewPod(opts ...KubernetesObjectOpt) *v1.Pod {
	newPod := &pod{}

	for _, opt := range opts {
		opt(newPod)
	}
	return (*v1.Pod)(newPod)
}

func (p *objectFactory) NewNode(opts ...KubernetesObjectOpt) *v1.Node {
	newNode := &node{}
	for _, opt := range opts {
		opt(newNode)
	}
	return (*v1.Node)(newNode)
}
