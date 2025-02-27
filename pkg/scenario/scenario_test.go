package scenario

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadScenarioFromYAML(t *testing.T) {
	yamlData := `
name: test-scenario
nodes:
- name: node-1
  memAllocatable: 1Gi
  cpuAllocatable: 1
  pod: 10
events:
 - pod:
    cpu: "100m"
    mem: "128Mi"
   after: 5s
   duration: 10s
 - pod:
    cpu: "100m"
    mem: "128Mi"
   after: 5s
   duration: 10s
`
	scenario, err := Load([]byte(yamlData))
	fmt.Println(scenario)
	assert.NoError(t, err, "Failed to load scenario from YAML")
	assert.Len(t, scenario.Events, 2, "Scenario should have 2 events")

}
