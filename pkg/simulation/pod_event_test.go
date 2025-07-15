package simulation

import (
	"fmt"
	"github.com/ghodss/yaml"
	"testing"
)

// test file unmarshal pod event

var podEventYaml = `
arrivalTime: 10s
evictTime: 5s
podSpec:
  metadata:
    name: test-pod
    namespace: default
  spec:
    containers:
      - name: test-container
        image: nginx
event_type: create
`

func TestPodEvent_UnmarshalJSON(t *testing.T) {
	var event PodEvent
	if err := yaml.Unmarshal([]byte(podEventYaml), &event); err != nil {
		t.Fatalf("failed to unmarshal PodEvent: %v", err)
	}
	fmt.Println(event)
}
