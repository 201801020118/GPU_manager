package device

import (
	"tkestack.io/gpu-manager/pkg/config"

	"k8s.io/klog"
)

//GPUTree is an interface for GPU tree structure
type GPUTree interface {
	Init(input string)
	Update()
}

//NewFunc is a function to create GPUTree
type NewFunc func(cfg *config.Config) GPUTree

var (
	factory = make(map[string]NewFunc)
)

//Register NewFunc with name, which can be get
//by calling NewFuncForName() later.
func Register(name string, item NewFunc) {
	if _, ok := factory[name]; ok {
		return
	}

	klog.V(2).Infof("Register NewFunc with name %s", name)

	factory[name] = item
}

//NewFuncForName tries to find functions with specific name
//from factory, return nil if not found.
func NewFuncForName(name string) NewFunc {
	if item, ok := factory[name]; ok {
		return item
	}

	klog.V(2).Infof("Can not find NewFunc with name %s", name)

	return nil
}
