package cambricon

import (
	"tkestack.io/gpu-manager/pkg/config"
	"tkestack.io/gpu-manager/pkg/device"
)

func init() {
	device.Register("cambeicon", NewCambeiconTree)
}

// CambeiconTree represents cambeicon tree struct
type CambeiconTree struct {
}

var _ device.GPUTree = &CambeiconTree{}

// NewCambeiconTree creates a new DummyTree
func NewCambeiconTree(_ *config.Config) device.GPUTree {
	return &CambeiconTree{}
}

// Init a CambeiconTree
func (t *CambeiconTree) Init(_ string) {
}

// Update a CambeiconTree
func (t *CambeiconTree) Update() {

}
