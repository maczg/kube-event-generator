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
  - name: pod-1
    from: 1s
    duration: 10s
    pod:
     metadata:
       name: pod-1
       namespace: default
     spec:
       containers:
       - name: nginx
         image: nginx
         resources:
          requests:
            cpu: 1
            memory: 128Mi
  - name: pod-2
    from: 5s
    duration: 11s
    pod:
     metadata:
       name: pod-2
       namespace: default 
     spec:
       containers:
       - name: nginx
         image: nginx
         resources:
           requests:
             cpu: 1
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
