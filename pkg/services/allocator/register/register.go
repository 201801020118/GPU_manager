package register

import (
	// Register test allocator
	_ "tkestack.io/gpu-manager/pkg/services/allocator/dummy"
	// Register nvidia allocator
	_ "tkestack.io/gpu-manager/pkg/services/allocator/nvidia"
	// Register cambricon allocator
	//有一个问题时tkestack.io是什么?
	_ "tkestack.io/gpu-manager/pkg/services/allocator/cambricon"
)
