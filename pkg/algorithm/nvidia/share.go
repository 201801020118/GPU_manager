package nvidia

import (
	"k8s.io/klog"
	"sort"

	"tkestack.io/gpu-manager/pkg/device/nvidia"
)

type shareMode struct {
	tree *nvidia.NvidiaTree
}

//NewShareMode returns a new shareMode struct.
//
//Evaluate() of shareMode returns one node with minimum available cores
//which fullfil the request.
//
//Share mode means multiple application may share one GPU node which uses
//GPU more efficiently.
func NewShareMode(t *nvidia.NvidiaTree) *shareMode {
	return &shareMode{t}
}

//func (al *shareMode) Evaluate(cores int64, memory int64) []*nvidia.NvidiaNode {
//	var (
//		nodes    []*nvidia.NvidiaNode
//		tmpStore = make([]*nvidia.NvidiaNode, al.tree.Total())
//		sorter   = shareModeSort(nvidia.ByAllocatableCores, nvidia.ByAllocatableMemory, nvidia.ByPids, nvidia.ByMinorID)
//	)
//
//	for i := 0; i < al.tree.Total(); i++ {
//		tmpStore[i] = al.tree.Leaves()[i]
//	}
//
//	sorter.Sort(tmpStore)
//
//	for _, node := range tmpStore {
//		if node.AllocatableMeta.Cores >= cores && node.AllocatableMeta.Memory >= memory {
//			klog.V(2).Infof("Pick up %d mask %b, cores: %d, memory: %d", node.Meta.ID, node.Mask, node.AllocatableMeta.Cores, node.AllocatableMeta.Memory)
//			nodes = append(nodes, node)
//			break
//		}
//	}
//
//	return nodes
//}

func (al *shareMode) Evaluate(cores int64, memory int64) []*nvidia.NvidiaNode {
	var (
		nodes    []*nvidia.NvidiaNode
		tmpStore = make([]*nvidia.NvidiaNode, al.tree.Total())
		sorter   = shareModeSort(nvidia.ByAllocatableCores, nvidia.ByAllocatableMemory, nvidia.ByPids, nvidia.ByMinorID)
	)

	for i := 0; i < al.tree.Total(); i++ {
		tmpStore[i] = al.tree.Leaves()[i]
	}

	sorter.Sort(tmpStore)

	for _, node := range tmpStore {
		if node.AllocatableMeta.Cores >= cores && node.AllocatableMeta.Memory >= memory {
			klog.V(2).Infof("Pick up %d mask %b, cores: %d, memory: %d", node.Meta.ID, node.Mask, node.AllocatableMeta.Cores, node.AllocatableMeta.Memory)
			nodes = append(nodes, node)
			break
		}
	}

	return nodes
}

type shareModePriority struct {
	data []*nvidia.NvidiaNode
	less []nvidia.LessFunc
}

func shareModeSort(less ...nvidia.LessFunc) *shareModePriority {
	return &shareModePriority{
		less: less,
	}
}

func (smp *shareModePriority) Sort(data []*nvidia.NvidiaNode) {
	smp.data = data
	sort.Sort(smp)
}

func (smp *shareModePriority) Len() int {
	return len(smp.data)
}

func (smp *shareModePriority) Swap(i, j int) {
	smp.data[i], smp.data[j] = smp.data[j], smp.data[i]
}

func (smp *shareModePriority) Less(i, j int) bool {
	var k int

	for k = 0; k < len(smp.less)-1; k++ {
		less := smp.less[k]
		switch {
		case less(smp.data[i], smp.data[j]):
			return true
		case less(smp.data[j], smp.data[i]):
			return false
		}
	}

	return smp.less[k](smp.data[i], smp.data[j])
}
