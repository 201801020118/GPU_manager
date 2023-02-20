package allocator

import (
	virtualmanager "tkestack.io/gpu-manager/pkg/api/runtime/virtual-manager"
	"tkestack.io/gpu-manager/pkg/config"
	"tkestack.io/gpu-manager/pkg/device"
	"tkestack.io/gpu-manager/pkg/services/response"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// GPUTopoService is server api for GPU topology service
type GPUTopoService interface {
	pluginapi.DevicePluginServer
	ListAndWatchWithResourceName(string, *pluginapi.Empty, pluginapi.DevicePlugin_ListAndWatchServer) error
	virtualmanager.VirtualManagerServer
}

// NewFunc represents function for creating new GPUTopoService
type NewFunc func(cfg *config.Config,
	tree device.GPUTree,
	k8sClient kubernetes.Interface,
	responseManager response.Manager) GPUTopoService

var (
	factory = make(map[string]NewFunc)
)

// Register stores NewFunc in factory
func Register(name string, item NewFunc) {
	if _, ok := factory[name]; ok {
		return
	}

	klog.V(2).Infof("Register NewFunc with name %s", name)

	factory[name] = item
}

// NewFuncForName tries to find NewFunc by name, return nil if not found
// 试图找到NewFunc的名字,如果没有找到返回nil
func NewFuncForName(name string) NewFunc {
	if item, ok := factory[name]; ok {
		return item
	}

	klog.V(2).Infof("Can not find NewFunc with name %s", name)

	return nil
}
