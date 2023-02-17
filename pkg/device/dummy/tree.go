package dummy

import (
	"tkestack.io/gpu-manager/pkg/config"
	"tkestack.io/gpu-manager/pkg/device"
)

func init() {
	device.Register("dummy", NewDummyTree)
}

//DummyTree represents dummy tree struct
type DummyTree struct {
}

var _ device.GPUTree = &DummyTree{}

//NewDummyTree creates a new DummyTree
func NewDummyTree(_ *config.Config) device.GPUTree {
	return &DummyTree{}
}

//Init a DummyTree
func (t *DummyTree) Init(_ string) {
}

//Update a DummyTree
func (t *DummyTree) Update() {

}
