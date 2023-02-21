package nvidia

import (
	"github.com/mxpv/nvml-go"
	"strings"
)

func parseToGpuTopologyLevel(str string) nvml.GPUTopologyLevel {
	switch str {
	case "PIX":
		return nvml.TopologySingle
	case "PXB":
		return nvml.TopologyMultiple
	case "PHB":
		return nvml.TopologyHostbridge
	case "SOC":
		return nvml.TopologyNode
	}

	if strings.HasPrefix(str, "GPU") {
		return nvml.TopologyInternal
	}

	return nvml.TOPOLOGY_UNKNOWN
}
