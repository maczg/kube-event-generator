package utils

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var PodFactory = &podFactory{}

type podFactory struct{}

func (p *podFactory) CreatePod(ctx context.Context, client kubernetes.Clientset, pod *corev1.Pod) (*corev1.Pod, error) {
	pod, err := client.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	return pod, err
}

func (p *podFactory) DeletePod(ctx context.Context, client kubernetes.Clientset, pod *corev1.Pod) error {
	return client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
}

func (p *podFactory) DeleteAllPods(ctx context.Context, client kubernetes.Clientset, namespaces []string) error {
	for _, ns := range namespaces {
		err := client.CoreV1().Pods(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *podFactory) GetPod(ctx context.Context, client kubernetes.Clientset, namespace, name string) (*corev1.Pod, error) {
	return client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (p *podFactory) ListPods(ctx context.Context, client kubernetes.Clientset, namespace string) (*corev1.PodList, error) {
	return client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
}

type pod corev1.Pod

func (p *podFactory) NewPod(opts ...ResourceOption) *corev1.Pod {
	newPod := &pod{}

	for _, opt := range opts {
		opt(newPod)
	}
	return (*corev1.Pod)(newPod)
}

func PodWithMetadata(name, namespace string, labels, annotations map[string]string) ResourceOption {
	return func(c ResourceAdapter) {
		c.SetName(name)
		c.SetNamespace(namespace)
		c.SetLabels(labels)
		c.SetAnnotations(annotations)
	}
}

func PodWithContainer(name, image, cpu, memory string) ResourceOption {
	return func(obj ResourceAdapter) {
		if innerPod, ok := obj.(*pod); ok {
			container := corev1.Container{
				Name:  name,
				Image: image,
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(cpu),
						corev1.ResourceMemory: resource.MustParse(memory),
					},
				},
			}
			innerPod.Spec.Containers = append(innerPod.Spec.Containers, container)
		}
	}
}

func PodWithNodeAffinity(requiredLabel map[string]string, softLabels map[string]string) ResourceOption {
	return func(obj ResourceAdapter) {
		if innerPod, ok := obj.(*pod); ok {
			var requiredTerms []corev1.NodeSelectorRequirement
			var preferredTerms []corev1.PreferredSchedulingTerm

			affinity := &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{},
			}
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
			innerPod.Spec.Affinity = affinity
		}
	}
}
