package nvidia

import (
	"flag"
	"testing"

	"tkestack.io/gpu-manager/pkg/types"
)

func init() {
	flag.Set("v", "4")
	flag.Set("logtostderr", "true")
}

func TestSort(t *testing.T) {
	flag.Parse()
	//init tree
	obj := NewNvidiaTree(nil)
	tree, _ := obj.(*NvidiaTree)
	testCase1 :=
		`    GPU0    GPU1    GPU2    GPU3    GPU4    GPU5
GPU0      X      PIX     PHB     PHB     SOC     SOC
GPU1     PIX      X      PHB     PHB     SOC     SOC
GPU2     PHB     PHB      X      PIX     SOC     SOC
GPU3     PHB     PHB     PIX      X      SOC     SOC
GPU4     SOC     SOC     SOC     SOC      X      PIX
GPU5     SOC     SOC     SOC     SOC     PIX      X
`
	tree.Init(testCase1)
	for idx, n := range tree.Leaves() {
		n.AllocatableMeta.Cores = HundredCore
		n.AllocatableMeta.Memory = 1024 - int64(idx)
	}

	//test sort
	expectLeaves := []string{"GPU5", "GPU0", "GPU1", "GPU2", "GPU3", "GPU4"}
	leaves := tree.Leaves()
	tree.MarkOccupied(leaves[5], 100, 1*types.MemoryBlockSize)
	ps := &printSort{
		less: []LessFunc{ByAllocatableCores,
			ByAvailable,
			ByType,
			ByAllocatableMemory,
			ByMinorID,
			ByPids,
			ByMemory},
	}
	ps.Sort(leaves)
	for i, s := range expectLeaves {
		if s != leaves[i].String() {
			t.Fatalf("sort went wrong")
		}
	}
}
