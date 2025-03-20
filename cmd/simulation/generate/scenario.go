package generate

import (
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/maczg/kube-event-generator/pkg/utils"
)

func generateDefaultScenario() *scenario.Scenario {
	logger := logger.NewLogger(logger.LevelInfo, "generate")

	scn := &scenario.Scenario{
		Name:   fmt.Sprintf("generated-%s", timeNow.Format("2006-01-02-15-04-05")),
		Events: []scenario.Event{},
	}
	nodes := utils.NodeFactory.NewNodeBatch("node", "4", "16Gi", "110", 5)
	for _, node := range nodes {
		scn.Cluster.Nodes = append(scn.Cluster.Nodes, *node)
	}
	logger.Info("generated default scenario with %d nodes. CPU 4, MEM 16, Pod count 110", len(nodes))
	return scn
}
