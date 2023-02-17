package nvidia

import (
	"sort"

	"k8s.io/klog"

	"tkestack.io/gpu-manager/pkg/device/nvidia"
)

type fragmentMode struct {
	tree *nvidia.NvidiaTree
}

//NewFragmentMode returns a new fragmentMode struct.
//
//Evaluate() of fragmentMode returns nodes with minimum available cores
//which fullfil the request.
//
//Fragment mode means to allocate cores on fragmented nodes first, which
//helps link mode work better.
func NewFragmentMode(t *nvidia.NvidiaTree) *fragmentMode {
	return &fragmentMode{t}
}

func (al *fragmentMode) Evaluate(cores int64, _ int64) []*nvidia.NvidiaNode {
	var (
		candidate = al.tree.Root()
		next      *nvidia.NvidiaNode
		sorter    = fragmentSort(nvidia.ByAvailable, nvidia.ByAllocatableMemory, nvidia.ByPids, nvidia.ByMinorID)
		nodes     = make([]*nvidia.NvidiaNode, 0)
		num       = int(cores / nvidia.HundredCore)
	)

	for next != candidate {
		next = candidate

		sorter.Sort(candidate.Children)

		for _, node := range candidate.Children {
			if len(node.Children) == 0 || node.Available() < num {
				continue
			}

			candidate = node
			klog.V(2).Infof("Choose id %d, mask %b", candidate.Meta.ID, candidate.Mask)
			break
		}
	}

	for _, n := range candidate.GetAvailableLeaves() {
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

type fragmentPriority struct {
	data []*nvidia.NvidiaNode
	less []nvidia.LessFunc
}

func fragmentSort(less ...nvidia.LessFunc) *fragmentPriority {
	return &fragmentPriority{
		less: less,
	}
}

func (fp *fragmentPriority) Sort(data []*nvidia.NvidiaNode) {
	fp.data = data
	sort.Sort(fp)
}

func (fp *fragmentPriority) Len() int {
	return len(fp.data)
}

func (fp *fragmentPriority) Swap(i, j int) {
	fp.data[i], fp.data[j] = fp.data[j], fp.data[i]
}

func (fp *fragmentPriority) Less(i, j int) bool {
	var k int

	for k = 0; k < len(fp.less)-1; k++ {
		less := fp.less[k]
		switch {
		case less(fp.data[i], fp.data[j]):
			return true
		case less(fp.data[j], fp.data[i]):
			return false
		}
	}

	return fp.less[k](fp.data[i], fp.data[j])
}
