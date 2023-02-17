package types

import (
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	VDeviceAnnotation       = "ideal.com/vcuda-device"
	VCoreAnnotation         = "ideal.com/vcuda-core"
	VCoreLimitAnnotation    = "ideal.com/vcuda-core-limit"
	VMemoryAnnotation       = "ideal.com/vcuda-memory"
	PredicateTimeAnnotation = "ideal.com/predicate-time"
	PredicateGPUIndexPrefix = "ideal.com/predicate-gpu-idx-"
	GPUAssigned             = "ideal.com/gpu-assigned"
	GPUMODELNAME            = "ideal.com/gpu-model"
	ClusterNameAnnotation   = "clusterName"

	// 记录在平均调度下实际分配的 gpu 和 memory 值
	// 后面接容器的索引
	RealVCoreAnnotationPrefix   = "ideal.com/real-vcuda-core-"
	RealVMemoryAnnotationPrefix = "ideal.com/real-vcuda-memory-"
	WeightVcoreAnnotationPrefix = "ideal.com/weight-"

	// 记录 Pod 应用的集群级调度策略
	PolicyNameAnnotation = "ideal.com/cluster-policy-name"

	VCUDA_MOUNTPOINT = "/etc/vcuda"

	/** 256MB */
	//MemoryBlockSize = 268435456
	/** 1MB */
	//MemoryBlockSize = 1048576

	/** 10MB */
	MemoryBlockSize = 10485760

	KubeletSocket                 = "kubelet.sock"
	VDeviceSocket                 = "vcuda.sock"
	CheckPointFileName            = "kubelet_internal_checkpoint"
	PreStartContainerCheckErrMsg  = "PreStartContainer check failed"
	PreStartContainerCheckErrType = "PreStartContainerCheckErr"
	UnexpectedAdmissionErrType    = "UnexpectedAdmissionError"
)

const (
	NvidiaCtlDevice    = "/dev/nvidiactl"
	NvidiaUVMDevice    = "/dev/nvidia-uvm"
	NvidiaFullpathRE   = `^/dev/nvidia([0-9]*)$`
	NvidiaDevicePrefix = "/dev/nvidia"
)

const (
	ManagerSocket = "/var/run/gpu-manager.sock"
)

const (
	CGROUP_BASE  = "/sys/fs/cgroup/memory"
	CGROUP_PROCS = "cgroup.procs"
)

type VCudaRequest struct {
	PodUID           string
	AllocateResponse *pluginapi.ContainerAllocateResponse
	ContainerName    string
	//Deprecated
	Cores int64
	//Deprecated
	Memory int64
	Done   chan error
}

type DevicesPerNUMA map[int64][]string

type PodDevicesEntry struct {
	PodUID        string
	ContainerName string
	ResourceName  string
	DeviceIDs     []string
	AllocResp     []byte
}

type PodDevicesEntryNUMA struct {
	PodUID        string
	ContainerName string
	ResourceName  string
	DeviceIDs     DevicesPerNUMA
	AllocResp     []byte
}

type CheckpointNUMA struct {
	PodDeviceEntries  []PodDevicesEntryNUMA
	RegisteredDevices map[string][]string
}

type Checkpoint struct {
	PodDeviceEntries  []PodDevicesEntry
	RegisteredDevices map[string][]string
}

type CheckpointDataNUMA struct {
	Data *CheckpointNUMA `json:"Data"`
}

type CheckpointData struct {
	Data *Checkpoint `json:"Data"`
}

var (
	DriverVersionMajor      int
	DriverVersionMinor      int
	DriverLibraryPath       string
	DriverOriginLibraryPath string
)

const (
	ContainerNameLabelKey = "io.kubernetes.container.name"
	PodNamespaceLabelKey  = "io.kubernetes.pod.namespace"
	PodNameLabelKey       = "io.kubernetes.pod.name"
	PodUIDLabelKey        = "io.kubernetes.pod.uid"
	PodCgroupNamePrefix   = "pod"
)
