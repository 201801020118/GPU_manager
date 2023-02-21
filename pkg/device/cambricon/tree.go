package cambricon

import (
	"github.com/mxpv/nvml-go"
	"k8s.io/client-go/kubernetes"
	"sync"
	"time"
	"tkestack.io/gpu-manager/pkg/config"
	"tkestack.io/gpu-manager/pkg/device"
)

const (
	NamePattern = "/dev/cambeicon%d"
	one         = uint32(1)
)

type CambriconTree struct {
	sync.Mutex
	root         *CambriconNode
	leaves       []*CambriconNode
	k8sclient    kubernetes.Interface
	realMode     bool
	query        map[string]*CambriconNode
	index        int
	samplePeriod time.Duration
}

func init() {
	device.Register("cambeicon", NewCambriconTree)
}

var _ device.GPUTree = &CambriconTree{}

// NewCambriconTree creates a new DummyTree
func NewCambriconTree(_ *config.Config) device.GPUTree {
	return &CambriconTree{}
}

func (t *CambriconTree) allocateNode(index int) *CambriconNode {
	node := NewCambriconNode(t)

	node.ntype = nvml.TopologyInternal
	node.Meta.ID = index
	node.Mask = one << uint(index)

	return node
}

// Init a CambriconTree
func (t *CambriconTree) Init(_ string) {
}

// Update a CambriconTree
func (t *CambriconTree) Update() {

}
