package generate

import (
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/utils"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"time"
)

func setScenarioDefaultValue(scn *scenario.Scenario) {
	scn.Metadata.Name = fmt.Sprintf("scenario-%s", time.Now().Format("020106-15_04_05"))
	scn.Metadata.CreatedAt = time.Now().Format("2006-01-02T15:04:05Z")
	if len(scn.Cluster.Nodes) == 0 {
		logrus.Infof("No nodes found in scenario, creating new nodes")
		scn.Cluster.Nodes = make([]v1.Node, 0)
		randNodes := rng.Intn(10) + 1
		nodes := utils.NodeFactory.NewNodeBatch("node", "4", "16Gi", "110", randNodes)
		for _, node := range nodes {
			scn.Cluster.Nodes = append(scn.Cluster.Nodes, *node)
		}
	}
}
