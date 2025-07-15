package simulation

import (
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"testing"
)

var scenarioYaml = `
metadata:
  name: test-scenario
  createdAt: "2023-10-01T00:00:00Z"
  description: This is a test scenario
events:
  pods:
    - arrivalTime: 10s
      evictTime: 5s
      podSpec:
        metadata:
          name: test-pod
          namespace: default
        spec:
          containers:
            - name: test-container
              image: nginx
          resources:
            limits:
              cpu: "100m"
              memory: "128Mi"
`

func TestLoadFromYaml(t *testing.T) {
	var scenario Scenario

	err := yaml.Unmarshal([]byte(scenarioYaml), &scenario)
	assert.NoError(t, err, "Failed to unmarshal scenario YAML")

	assert.Equal(t, scenario.Metadata.Name, "test-scenario")
	assert.Equal(t, scenario.Metadata.CreatedAt, "2023-10-01T00:00:00Z")
	assert.Equal(t, scenario.Metadata.Description, "This is a test scenario")
	assert.Len(t, scenario.Events.Pods, 1)
	assert.Equal(t, scenario.Events.Pods[0].PodSpec.ObjectMeta.Name, "test-pod")
	assert.Equal(t, scenario.Events.Pods[0].PodSpec.ObjectMeta.Namespace, "default")
	assert.Equal(t, scenario.Events.Pods[0].PodSpec.Spec.Containers[0].Name, "test-container")
	assert.Equal(t, scenario.Events.Pods[0].PodSpec.Spec.Containers[0].Image, "nginx")
	assert.Equal(t, "100m", scenario.Events.Pods[0].PodSpec.Spec.Containers[0].Resources.Limits.Cpu().String())
	assert.Equal(t, "128Mi", scenario.Events.Pods[0].PodSpec.Spec.Containers[0].Resources.Limits.Memory().String())

}
