package cambricon

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	"time"
	virtualmanager "tkestack.io/gpu-manager/pkg/api/runtime/virtual-manager"
	"tkestack.io/gpu-manager/pkg/config"
	"tkestack.io/gpu-manager/pkg/device"
	"tkestack.io/gpu-manager/pkg/services/allocator"
	"tkestack.io/gpu-manager/pkg/services/response"
)

func init() {
	allocator.Register("dummy", NewCambriconAllocator)
}

// CambriconAllocator is a struct{}
type CambriconAllocator struct {
}

func (ta *CambriconAllocator) Vgpuinfo(c context.Context, request *virtualmanager.VgpuInfoRequest) (*virtualmanager.VgpuInfoResponse, error) {
	panic("implement me")
}

var _ allocator.GPUTopoService = &CambriconAllocator{}

// NewCambriconAllocator returns a new CambriconAllocator
func NewCambriconAllocator(_ *config.Config, _ device.GPUTree, _ kubernetes.Interface, _ response.Manager) allocator.GPUTopoService {
	return &CambriconAllocator{}
}

// Allocate returns /dev/fuse for Cambricon device
func (ta *CambriconAllocator) Allocate(_ context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resps := &pluginapi.AllocateResponse{}
	for range reqs.ContainerRequests {
		resps.ContainerResponses = append(resps.ContainerResponses, &pluginapi.ContainerAllocateResponse{
			Devices: []*pluginapi.DeviceSpec{
				{
					// We use /dev/fuse for dummy device
					ContainerPath: "/dev/fuse",
					HostPath:      "/dev/fuse",
					Permissions:   "mrw",
				},
			},
		})
	}

	return resps, nil
}

// ListAndWatch not implement
func (ta *CambriconAllocator) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	return fmt.Errorf("not implement")
}

// ListAndWatchWithResourceName sends dummy device back to server
func (ta *CambriconAllocator) ListAndWatchWithResourceName(resourceName string, e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	devs := []*pluginapi.Device{
		{
			ID:     fmt.Sprintf("dummy-%s-0", resourceName),
			Health: pluginapi.Healthy,
		},
	}

	s.Send(&pluginapi.ListAndWatchResponse{Devices: devs})

	// We don't send unhealthy state
	for {
		time.Sleep(time.Second)
	}

	klog.V(2).Infof("ListAndWatch %s exit", resourceName)

	return nil
}

// GetDevicePluginOptions returns empty DevicePluginOptions
func (ta *CambriconAllocator) GetDevicePluginOptions(ctx context.Context, e *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

// PreStartContainer returns empty PreStartContainerResponse
func (ta *CambriconAllocator) PreStartContainer(ctx context.Context, req *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}
