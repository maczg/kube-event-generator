package scenario

import (
	"encoding/json"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/factory"
	corev1 "k8s.io/api/core/v1"
	"time"
)

type Event struct {
	After    time.Duration `yaml:"after" json:"after"`
	Duration time.Duration `yaml:"duration" json:"duration"`
	PodSpec  PodSpec       `yaml:"pod,omitempty" json:"pod"`
	Pod      *corev1.Pod   `yaml:"-"`
}

func (e *Event) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type type_ Event
	var var_ type_
	if err := unmarshal(&var_); err != nil {
		return err
	}
	if var_.After == 0 {
		return fmt.Errorf("missing 'after' field")
	}
	p := factory.NewPod(
		factory.WithMetadata(var_.PodSpec.Name, var_.PodSpec.Namespace),
		factory.WithAffinity(var_.PodSpec.RequiredAffinity, var_.PodSpec.SoftAffinity),
		factory.WithContainer(var_.PodSpec.ContainerName, var_.PodSpec.Image, var_.PodSpec.CPU, var_.PodSpec.Mem),
	)
	var_.Pod = p
	*e = Event(var_)
	return nil
}

func (e *Event) UnmarshalJSON(data []byte) error {
	type type_ Event
	var var_ type_
	if err := json.Unmarshal(data, &var_); err != nil {
		return err
	}
	if var_.After == 0 {
		return fmt.Errorf("missing 'after' field")
	}
	p := factory.NewPod(
		factory.WithMetadata(var_.PodSpec.Name, var_.PodSpec.Namespace),
		factory.WithAffinity(var_.PodSpec.RequiredAffinity, var_.PodSpec.SoftAffinity),
		factory.WithContainer(var_.PodSpec.ContainerName, var_.PodSpec.Image, var_.PodSpec.CPU, var_.PodSpec.Mem),
	)
	var_.Pod = p
	*e = Event(var_)
	return nil
}
