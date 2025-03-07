package simulation

import (
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testScenarioBytes = []byte(`
name: "Scenario 1"
cluster:
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
    from: 5s
    duration: 20s
    pod:
     metadata:
       name: test-pod
       namespace: default
     spec:
       containers:
       - name: nginx
         image: nginx
         resources:
          requests:
            cpu: 100m
            memory: 128Mi
`)

func Test_RunStart(t *testing.T) {
	scn, err := scenario.Load(testScenarioBytes)
	assert.Nil(t, err)

	client, err := utils.MakeClientSet()
	assert.Nil(t, err)

	mgr := utils.NewKubernetesManager(client)

	sim := New(scn, mgr)

	err = sim.Start()
	assert.Nil(t, err)

}
