package nvidia

import (
	"strings"

	"tkestack.io/nvml"
)

func parseToGpuTopologyLevel(str string) nvml.GpuTopologyLevel {
	switch str {
	case "PIX":
		return nvml.TOPOLOGY_SINGLE
	case "PXB":
		return nvml.TOPOLOGY_MULTIPLE
	case "PHB":
		return nvml.TOPOLOGY_HOSTBRIDGE
	case "SOC":
		return nvml.TOPOLOGY_CPU
	}

	if strings.HasPrefix(str, "GPU") {
		return nvml.TOPOLOGY_INTERNAL
	}

	return nvml.TOPOLOGY_UNKNOWN
}
