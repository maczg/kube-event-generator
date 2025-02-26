package scenario

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadScenarioFromYAML(t *testing.T) {
	yamlData := `
events:
- type: create
  podSpec:
  name: test-pod-1
  namespace: default
  image: nginx:latest
  resources:
    cpu: "100m"
    memory: "128Mi"
  delayAfter: 5s
- type: evict
  podSpec:
  name: test-pod-2
  namespace: default
  image: nginx:latest
  resources:
    cpu: "100m"
    memory: "128Mi"
  delayAfter: 10s
`
	scenario, err := Load([]byte(yamlData))
	assert.NoError(t, err, "Failed to load scenario from YAML")
	assert.Len(t, scenario.Events, 2, "Scenario should have 2 events")

}
