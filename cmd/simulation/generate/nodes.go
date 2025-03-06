package generate

import (
	"github.com/maczg/kube-event-generator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"strconv"
)

type NodeParams struct {
	Count      int
	CpuPerNode string
	MemPerNode string
	PodCount   int
}

// generateNodes generates a list of nodes based on the given parameter
func generateNodes(params NodeParams) []corev1.Node {
	nodes := make([]corev1.Node, params.Count)
	utils.NodeFactory.NewNodeBatch("node", params.CpuPerNode, params.MemPerNode, strconv.Itoa(params.PodCount), params.Count)
	return nodes
}
