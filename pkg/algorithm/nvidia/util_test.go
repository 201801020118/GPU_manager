package nvidia

import (
	"tkestack.io/gpu-manager/pkg/device/nvidia"
)

func examining(expect []string, nodes []*nvidia.NvidiaNode) (pass bool, want string, actual string) {
	if len(expect) != len(nodes) {
		return false, "", ""
	}

	for i, n := range nodes {
		if expect[i] != n.MinorName() {
			return false, expect[i], n.MinorName()
		}
	}

	return true, "", ""
}
