package scenario

import (
	"k8s.io/apiserver/pkg/storage/names"
)

var podNameGenerator = func() func(base string) string {
	return func(base string) string {
		return names.SimpleNameGenerator.GenerateName(base)
	}
}()

type PodSpec struct {
	Name             string            `yaml:"name" json:"name"`
	Namespace        string            `json:"namespace" yaml:"namespace"`
	Image            string            `json:"image" yaml:"image"`
	Labels           map[string]string `json:"labels" yaml:"labels"`
	RequiredAffinity map[string]string `json:"requiredAffinity" yaml:"requiredAffinity"`
	SoftAffinity     map[string]string `json:"softAffinity" yaml:"softAffinity"`
	ContainerName    string            `json:"containerName" yaml:"containerName"`
	CPU              string            `yaml:"cpu" json:"cpu"`
	Mem              string            `yaml:"mem" json:"mem"`
}

func (p *PodSpec) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type pod PodSpec
	var v pod
	if err := unmarshal(&v); err != nil {
		return err
	}
	if v.Name == "" {
		v.Name = podNameGenerator("pod-")
	}
	if v.Namespace == "" {
		v.Namespace = "default"
	}
	if v.Image == "" {
		v.Image = "nginx"
	}
	if v.ContainerName == "" {
		v.ContainerName = "server"
	}
	if v.Labels == nil {
		v.Labels = make(map[string]string)
	}
	*p = PodSpec(v)
	return nil
}
