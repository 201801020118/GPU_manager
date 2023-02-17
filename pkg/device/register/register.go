package register

import (
	// Register test device
	_ "tkestack.io/gpu-manager/pkg/device/dummy"
	// Register nvidia device
	_ "tkestack.io/gpu-manager/pkg/device/nvidia"
)
