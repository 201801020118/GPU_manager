package nvidia

import (
	"sort"

	"k8s.io/klog"

	"tkestack.io/gpu-manager/pkg/device/nvidia"
)

type linkMode struct {
	tree *nvidia.NvidiaTree
}

//NewLinkMode returns a new linkMode struct.
//
//Evaluate() of linkMode returns nodes with minimum connection overhead
//of each other.
func NewLinkMode(t *nvidia.NvidiaTree) *linkMode {
	return &linkMode{t}
}

func (al *linkMode) Evaluate(cores int64, memory int64) []*nvidia.NvidiaNode {
	var (
		sorter   = linkSort(nvidia.ByType, nvidia.ByAvailable, nvidia.ByAllocatableMemory, nvidia.ByPids, nvidia.ByMinorID)
		tmpStore = make(map[int]*nvidia.NvidiaNode)
		root     = al.tree.Root()
		nodes    = make([]*nvidia.NvidiaNode, 0)
		num      = int(cores / nvidia.HundredCore)
	)

	for _, node := range al.tree.Leaves() {
		for node != root {
			klog.V(2).Infof("Test %d mask %b", node.Meta.ID, node.Mask)
			if node.Available() < num {
				node = node.Parent
				continue
			}

			tmpStore[node.Meta.ID] = node
			klog.V(2).Infof("Choose %d mask %b", node.Meta.ID, node.Mask)
			break
		}
	}

	if len(tmpStore) == 0 {
		tmpStore[-1] = root
	}

	candidates := make([]*nvidia.NvidiaNode, 0)
	for _, n := range tmpStore {
		candidates = append(candidates, n)
	}

	sorter.Sort(candidates)

	for _, n := range candidates[0].GetAvailableLeaves() {
		if num == 0 {
			break
		}

		klog.V(2).Infof("Pick up %d mask %b", n.Meta.ID, n.Mask)
		nodes = append(nodes, n)
		num--
	}

	if num > 0 {
		return nil
	}

	return nodes
}

type linkPriority struct {
	data []*nvidia.NvidiaNode
	less []nvidia.LessFunc
}

func linkSort(less ...nvidia.LessFunc) *linkPriority {
	return &linkPriority{
		less: less,
	}
}

func (lp *linkPriority) Sort(data []*nvidia.NvidiaNode) {
	lp.data = data
	sort.Sort(lp)
}

func (lp *linkPriority) Len() int {
	return len(lp.data)
}

func (lp *linkPriority) Swap(i, j int) {
	lp.data[i], lp.data[j] = lp.data[j], lp.data[i]
}

func (lp *linkPriority) Less(i, j int) bool {
	var k int

	for k = 0; k < len(lp.less)-1; k++ {
		less := lp.less[k]
		switch {
		case less(lp.data[i], lp.data[j]):
			return true
		case less(lp.data[j], lp.data[i]):
			return false
		}
	}

	return lp.less[k](lp.data[i], lp.data[j])
}
