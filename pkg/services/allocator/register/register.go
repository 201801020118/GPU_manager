package register

import (
	// Register test allocator
	_ "tkestack.io/gpu-manager/pkg/services/allocator/dummy"
	// Register nvidia allocator
	_ "tkestack.io/gpu-manager/pkg/services/allocator/nvidia"
)
