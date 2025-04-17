package scenario

import (
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var testScenario = []byte(`
metadata:
  name: "test-scenario"
cluster:
 nodes:
  - metadata:
     name: "node-1"
events:
 pods:
  - name: pod-test
    after: 1s
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
            cpu: 100m
            memory: 128Mi
`)

func TestScenario_UnmarshalYaml(t *testing.T) {
	var s Scenario
	err := yaml.Unmarshal(testScenario, &s)
	assert.NoError(t, err)
	assert.Equal(t, "test-scenario", s.Metadata.Name)
	assert.Equal(t, 1, len(s.Cluster.Nodes))
	assert.Equal(t, "node-1", s.Cluster.Nodes[0].GetName())
	assert.Equal(t, 1, len(s.Events.Pods))
	assert.Equal(t, 1*time.Second, time.Duration(s.Events.Pods[0].ExecuteAfter))
	assert.Equal(t, 10*time.Second, time.Duration(s.Events.Pods[0].ExecuteDuration))
}
