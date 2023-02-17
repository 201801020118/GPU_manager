package nvidia

import (
	node "tkestack.io/gpu-manager/pkg/device/nvidia"
)

//Evaluator api for schedule algorithm
type Evaluator interface {
	Evaluate(cores int64, memory int64) []*node.NvidiaNode
}
