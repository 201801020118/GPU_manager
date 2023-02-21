package nvidia

import (
	"fmt"
	"github.com/mxpv/nvml-go"
	"math/bits"

	"k8s.io/klog"
)

// SchedulerCache contains allocatable resource of GPU
type SchedulerCache struct {
	Cores  int64
	Memory int64
}

// DeviceMeta contains metadata of GPU device
type DeviceMeta struct {
	ID          int
	ModelName   string
	MinorID     int
	UsedMemory  uint64
	TotalMemory uint64
	Pids        []uint
	BusId       string
	Utilization uint
	UUID        string
}

// NvidiaNode represents a node of Nvidia GPU
type NvidiaNode struct {
	Meta            DeviceMeta
	AllocatableMeta SchedulerCache

	Parent   *NvidiaNode
	Children []*NvidiaNode
	Mask     uint32

	pendingReset bool
	vchildren    map[int]*NvidiaNode
	ntype        nvml.GPUTopologyLevel
	tree         *NvidiaTree
}

var (
	/** test only */
	nodeIndex = 0
)

// NewNvidiaNode returns a new NvidiaNode
func NewNvidiaNode(t *NvidiaTree) *NvidiaNode {
	node := &NvidiaNode{
		vchildren: make(map[int]*NvidiaNode),
		ntype:     nvml.TOPOLOGY_UNKNOWN,
		tree:      t,
		Meta: DeviceMeta{
			ID: nodeIndex,
		},
	}

	nodeIndex++

	return node
}

func (n *NvidiaNode) setParent(p *NvidiaNode) {
	n.Parent = p
	p.vchildren[n.Meta.ID] = n
}

// MinorName returns MinorID of this NvidiaNode
func (n *NvidiaNode) MinorName() string {
	return fmt.Sprintf(NamePattern, n.Meta.MinorID)
}

// Type returns GpuTopologyLevel of this NvidiaNode
func (n *NvidiaNode) Type() int {
	return int(n.ntype)
}

// GetAvailableLeaves returns leaves of this NvidiaNode
// which available for allocating.
func (n *NvidiaNode) GetAvailableLeaves() []*NvidiaNode {
	var leaves []*NvidiaNode

	mask := n.Mask

	for mask != 0 {
		id := uint32(bits.TrailingZeros32(mask))
		klog.V(2).Infof("Pick up %d mask %b", id, n.tree.leaves[id].Mask)
		leaves = append(leaves, n.tree.leaves[id])
		mask ^= one << id
	}

	return leaves
}

// Available returns conut of available leaves
// of this NvidiaNode.
func (n *NvidiaNode) Available() int {
	return bits.OnesCount32(n.Mask)
}

func (n *NvidiaNode) String() string {
	switch n.ntype {
	case nvml.TopologyInternal:
		return fmt.Sprintf("GPU%d", n.Meta.ID)
	case nvml.TopologySingle:
		return "PIX"
	case nvml.TopologyMultiple:
		return "PXB"
	case nvml.TopologyHostbridge:
		return "PHB"
	case nvml.TopologyNode:
		return "CPU"
	case nvml.TopologySystem:
		return "SYS"
	}

	return "ROOT"
}
