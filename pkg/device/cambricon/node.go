package cambricon

import (
	"C"
	"fmt"
	"github.com/mxpv/nvml-go"
	"math/bits"
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

// CambriconNode represents a node of Nvidia GPU
type CambriconNode struct {
	Meta            DeviceMeta
	AllocatableMeta SchedulerCache

	Parent   *CambriconNode
	Children []*CambriconNode
	Mask     uint32

	pendingReset bool
	vchildren    map[int]*CambriconNode
	ntype        C.cndevTopologyGetRelationship
	tree         *CambriconTree
}

var (
	/** test only */
	nodeIndex = 0
)

// NewCambriconNode returns a new CambriconNode
func NewCambriconNode(t *CambriconTree) *CambriconNode {
	node := &CambriconNode{
		vchildren: make(map[int]*CambriconNode),
		ntype:     nvml.TOPOLOGY_UNKNOWN,
		tree:      t,
		Meta: DeviceMeta{
			ID: nodeIndex,
		},
	}

	nodeIndex++

	return node
}

func (n *CambriconNode) setParent(p *CambriconNode) {
	n.Parent = p
	p.vchildren[n.Meta.ID] = n
}

// MinorName returns MinorID of this CambriconNode
func (n *CambriconNode) MinorName() string {
	return fmt.Sprintf(NamePattern, n.Meta.MinorID)
}

// Type returns GpuTopologyLevel of this CambriconNode
func (n *CambriconNode) Type() int {
	return int(n.ntype)
}

// GetAvailableLeaves returns leaves of this CambriconNode
// which available for allocating.
func (n *CambriconNode) GetAvailableLeaves() []*CambriconNode {
	var leaves []*CambriconNode

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
// of this CambriconNode.
func (n *CambriconNode) Available() int {
	return bits.OnesCount32(n.Mask)
}

func (n *CambriconNode) String() string {
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
