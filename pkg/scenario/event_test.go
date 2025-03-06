package scenario

import (
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
	"testing"
	"time"
)

var testEventBytes = []byte(`
name: pod-test
type: PodCreate
from: 1s
duration: 10s
pod:
  metadata:
   name: test-pod
   namespace: default
  spec:
    containers:
    - name: nginx
      resources:
        requests:
         cpu: "100m"
         memory: "128Mi"
`)

func TestEvent_UnmarshalYaml(t *testing.T) {
	var event Event
	err := yaml.Unmarshal(testEventBytes, &event)
	if err != nil {
		t.Fatalf("error  %s with type %T\n", err, err)
	}
	assert.Equal(t, "pod-test", event.Name)
	assert.Equal(t, "PodCreate", event.Type)
	assert.Equal(t, "PodCreate", event.Type)
	assert.Equal(t, 1*time.Second, event.From)
	assert.Equal(t, 10*time.Second, event.Duration)
	assert.Equal(t, "test-pod", event.Pod.Name)
	assert.Equal(t, "default", event.Pod.Namespace)
	assert.Equal(t, "nginx", event.Pod.Spec.Containers[0].Name)
	assert.Equal(t, "100m", event.Pod.Spec.Containers[0].Resources.Requests.Cpu().String())
	assert.Equal(t, "128Mi", event.Pod.Spec.Containers[0].Resources.Requests.Memory().String())
}
