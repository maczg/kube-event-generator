package scenario

import (
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var testScenarioBytes = []byte(`
name: "Scenario 1"
cluster:
 schedulerWeights:
   plugin-1: 1
   plugin-2: 2
 nodes:
   - metadata:
      name: "node-1"
     status:
       capacity:
         cpu: 1
         memory: 8Gi
         pods: 110
       allocatable:
         cpu: 1
         memory: 8Gi
         pods: 110
events:
  - name: pod-test
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
            cpu: 100m
            memory: 128Mi
`)

func TestScenario_UnmarshalYaml(t *testing.T) {
	var s Scenario
	err := yaml.Unmarshal(testScenarioBytes, &s)
	if err != nil {
		t.Fatalf("error  %s with type %T\n", err, err)
	}
	assert.Equal(t, "Scenario 1", s.Metadata.Name)
	assert.Equal(t, 2, len(s.Cluster.SchedulerWeights))
	assert.Equal(t, 1, s.Cluster.SchedulerWeights["plugin-1"])
	assert.Equal(t, 2, s.Cluster.SchedulerWeights["plugin-2"])
	assert.Equal(t, 1, len(s.Cluster.Nodes))
	assert.Equal(t, "node-1", s.Cluster.Nodes[0].Name)
	assert.Equal(t, "1", s.Cluster.Nodes[0].Status.Capacity.Cpu().String())
	assert.Equal(t, "8Gi", s.Cluster.Nodes[0].Status.Capacity.Memory().String())
	assert.Equal(t, "110", s.Cluster.Nodes[0].Status.Capacity.Pods().String())
	assert.Equal(t, "110", s.Cluster.Nodes[0].Status.Allocatable.Pods().String())
	assert.Equal(t, "pod-test", s.Events[0].Name)
	assert.Equal(t, 1*time.Second, s.Events[0].From)
	assert.Equal(t, 10*time.Second, s.Events[0].Duration)
	assert.Equal(t, "test-pod", s.Events[0].Pod.Name)
	assert.Equal(t, "default", s.Events[0].Pod.Namespace)
	assert.Equal(t, "nginx", s.Events[0].Pod.Spec.Containers[0].Name)
	assert.Equal(t, "100m", s.Events[0].Pod.Spec.Containers[0].Resources.Requests.Cpu().String())
	assert.Equal(t, "128Mi", s.Events[0].Pod.Spec.Containers[0].Resources.Requests.Memory().String())
}
