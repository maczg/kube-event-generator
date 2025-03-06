package scenario

import (
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testClusterBytes = []byte(`
schedulerWeights:
 plugin-1: 1
 plugin-2: 2
nodes:
 - metadata:
     name: "node-1"
   status:
     capacity:
       cpu: "1"
       memory: "8Gi"
       pods: "110"
     allocatable:
       cpu: "1"
       memory: "8Gi"
       pods: "110"
`)

func TestCluster_ParseYaml(t *testing.T) {
	var c Cluster
	err := yaml.Unmarshal(testClusterBytes, &c)
	if err != nil {
		t.Fatalf("error  %s with type %T\n", err, err)
	}
	assert.Equal(t, 2, len(c.SchedulerWeights))
	assert.Equal(t, 1, c.SchedulerWeights["plugin-1"])
	assert.Equal(t, 2, c.SchedulerWeights["plugin-2"])
	assert.Equal(t, 1, len(c.Nodes))
	assert.Equal(t, "node-1", c.Nodes[0].Name)
	assert.Equal(t, "1", c.Nodes[0].Status.Capacity.Cpu().String())
	assert.Equal(t, "8Gi", c.Nodes[0].Status.Capacity.Memory().String())
	assert.Equal(t, "110", c.Nodes[0].Status.Capacity.Pods().String())
	assert.Equal(t, "1", c.Nodes[0].Status.Allocatable.Cpu().String())
	assert.Equal(t, "8Gi", c.Nodes[0].Status.Allocatable.Memory().String())
	assert.Equal(t, "110", c.Nodes[0].Status.Allocatable.Pods().String())
}
