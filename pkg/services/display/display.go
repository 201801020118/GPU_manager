package display

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cast"
	displayapi "tkestack.io/gpu-manager/pkg/api/runtime/display"
	"tkestack.io/gpu-manager/pkg/config"
	"tkestack.io/gpu-manager/pkg/device"
	nvtree "tkestack.io/gpu-manager/pkg/device/nvidia"
	"tkestack.io/gpu-manager/pkg/runtime"
	"tkestack.io/gpu-manager/pkg/services/watchdog"
	"tkestack.io/gpu-manager/pkg/types"
	"tkestack.io/gpu-manager/pkg/utils"
	"tkestack.io/gpu-manager/pkg/version"

	google_protobuf1 "github.com/golang/protobuf/ptypes/empty"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

const (
	// 权重调度
	WeightSchedulerName = "WEIGHT-SCHEDULING"
	// 抢占调度
	PreemptSchedulerName = "PREEMPT-SCHEDULING"
	// 平均调度
	AverageSchedulerName = "AVERAGE-SCHEDULING"
)

// Display is used to show GPU device usage
type Display struct {
	sync.Mutex

	config                  *config.Config
	tree                    *nvtree.NvidiaTree                //nvidia GPU资源树
	containerRuntimeManager runtime.ContainerRuntimeInterface //容器运行时manager
}

type VgpuNodeInfo struct {
	NodeName          string `json:"nodeName"`
	Gpuid             int    `json:"gpuid"`
	ModelName         string `json:"modelName"`
	Available         int    `json:"available"`
	Pids              int    `json:"pids"`
	UsedMemory        uint64 `json:"usedMemory"`
	AllocatableCores  int64  `json:"allocatableCores"`
	AllocatableMemory int64  `json:"allocatableMemory"`
}

var _ displayapi.GPUDisplayServer = &Display{}
var _ prometheus.Collector = &Display{}

// NewDisplay returns a new Display
func NewDisplay(config *config.Config, tree device.GPUTree, runtimeManager runtime.ContainerRuntimeInterface) *Display {
	_tree, _ := tree.(*nvtree.NvidiaTree)
	return &Display{
		tree:                    _tree,
		config:                  config,
		containerRuntimeManager: runtimeManager,
	}
}

// PrintGraph updates the tree and returns the result of tree.PrintGraph
func (disp *Display) PrintGraph(context.Context, *google_protobuf1.Empty) (*displayapi.GraphResponse, error) {
	disp.tree.Update()

	return &displayapi.GraphResponse{
		Graph: disp.tree.PrintGraph(),
	}, nil
}

func (disp *Display) PrintGpuinfo(context.Context, *google_protobuf1.Empty) (*displayapi.GraphResponse, error) {
	disp.tree.Update()

	return &displayapi.GraphResponse{
		Graph: disp.tree.PrintGpuinfo(),
	}, nil
}

// PrintUsages returns usage info getting from docker and watchdog
func (disp *Display) PrintUsages(context.Context, *google_protobuf1.Empty) (*displayapi.UsageResponse, error) {
	disp.Lock()
	defer disp.Unlock()

	activePods := watchdog.GetActivePods()
	displayResp := &displayapi.UsageResponse{
		Usage: make(map[string]*displayapi.ContainerStat),
	}

	for _, pod := range activePods {
		podUID := string(pod.UID)

		podUsage := disp.getPodUsage(pod)
		if len(podUsage) > 0 {
			displayResp.Usage[podUID] = &displayapi.ContainerStat{
				Cluster: pod.Annotations[types.ClusterNameAnnotation],
				Project: pod.Namespace,
				User:    getUserName(pod),
				Stat:    podUsage,
				Spec:    disp.getPodSpec(pod, podUsage),
			}
		}
	}

	if len(displayResp.Usage) > 0 {
		return displayResp, nil
	}

	return &displayapi.UsageResponse{}, nil
}

func (disp *Display) PrintGpuDetails(ctx context.Context, req *displayapi.ContUsageRequest) (*displayapi.ContUsageResponse, error) {
	disp.Lock()
	defer disp.Unlock()
	klog.Infof("PrintGpuDetails get GpuId %s gpu details.", req.GpuId)

	nvml.Init()
	defer nvml.Shutdown()

	activePods := watchdog.GetActivePods()
	displayResp := &displayapi.ContUsageResponse{
		Info: make([]*displayapi.ContGpuinfo, 0),
	}

	if req.GpuId == "" {
		return nil, errors.New("req.Gpuid is null")
	}

	for _, pod := range activePods {

		for contIdx, cont := range pod.Spec.Containers {
			//如果跟cardid 相等
			if cardId, ok := pod.GetAnnotations()[types.PredicateGPUIndexPrefix+strconv.Itoa(contIdx)]; ok && cardId == req.GpuId {

				stat := v1.ContainerStatus{}
				for _, contStatus := range pod.Status.ContainerStatuses {
					if contStatus.Name == cont.Name {
						stat = contStatus
					}
				}
				contID := strings.TrimPrefix(stat.ContainerID, fmt.Sprintf("%s://", disp.containerRuntimeManager.RuntimeName()))
				gpuid, _ := strconv.Atoi(req.GpuId)
				resp, err := disp.PrintGpuinfo(ctx, &google_protobuf1.Empty{})
				if err != nil {
					return nil, err
				}
				infos := []*VgpuNodeInfo{}
				if err := json.Unmarshal([]byte(resp.Graph), &infos); err != nil {
					return nil, err
				}

				freemb := int64(0)
				for _, info := range infos {
					if info.Gpuid == gpuid {
						freemb = info.AllocatableMemory
					}
				}

				podUsage := disp.getPodUsage(pod)
				usage, exist := disp.getPodSpec(pod, podUsage)[stat.Name]
				if exist && usage != nil {
					if len(podUsage) > 0 {
						gpuinfo := &displayapi.ContGpuinfo{
							Namespace:   pod.Namespace,
							ProjectName: pod.Annotations["gsp-name"],
							PodName:     pod.Name,
							ContIdx:     strconv.Itoa(contIdx),
							ContName:    stat.Name,
							Contid:      contID,
							Policy:      pod.Annotations["ideal.com/cluster-policy-name"],
							GpuId:       req.GpuId,
							Gpu:         usage.Gpu,
							Mem:         usage.Mem,
							Priority:    pod.Annotations["ideal.com/preempt-priority-"+strconv.Itoa(contIdx)],
							FreeMem:     (freemb / types.MemoryBlockSize),
							//FreeGpu:,
						}
						displayResp.Info = append(displayResp.Info, gpuinfo)
					}
				}

			}

		}

	}

	if len(displayResp.Info) > 0 {
		return displayResp, nil
	}

	return displayResp, nil
}

func (disp *Display) FilterPodByGpuid(pod *v1.Pod, gpuid string) (bool, int, *v1.ContainerStatus) {
	for key, idx := range pod.GetAnnotations() {
		if strings.HasPrefix(key, types.PredicateGPUIndexPrefix) {
			if gpuid == idx {
				strs := strings.Split(key, types.PredicateGPUIndexPrefix)
				containerIndex, _ := strconv.Atoi(strs[len(strs)-1])

				return true, containerIndex, &pod.Status.ContainerStatuses[containerIndex]
			}
		}
	}

	return false, -1, nil
}

//
//func (disp *Display) FilterPodBycontId(activePods map[string]*v1.Pod,contId string) {
//
//	for _, pod := range activePods {
//
//		for _, stat := range pod.Status.ContainerStatuses {
//			contID := strings.TrimPrefix(stat.ContainerID, fmt.Sprintf("%s://", disp.containerRuntimeManager.RuntimeName()))
//			if len(contID) == 0 {
//				continue
//			}
//			if contID == contId {
//				for key,gpuid:=range pod.Annotations{
//					if strings.HasPrefix(key, types.PredicateGPUIndexPrefix) {
//						if gpuid==idx{
//							return true,pod.
//						}
//					}
//				}
//
//
//			}
//		}
//
//	}
//
//}

func (disp *Display) getPodSpec(pod *v1.Pod, devicesInfo map[string]*displayapi.Devices) map[string]*displayapi.Spec {
	podSpec := make(map[string]*displayapi.Spec)
	var needCores int64 = 0
	for containerIndex, ctnt := range pod.Spec.Containers {

		if coreStr, ok := pod.Annotations[types.RealVCoreAnnotationPrefix+strconv.Itoa(containerIndex)]; ok {
			needCores = cast.ToInt64(coreStr)
		}
		//else {
		//	vcore := ctnt.Resources.Requests[types.VCoreAnnotation]
		//	needCores = vcore.Value()
		//}

		vmemory := ctnt.Resources.Requests[types.VMemoryAnnotation]
		memBytes := vmemory.Value()
		//前端已经乘以10mb
		//* types.MemoryBlockSize

		spec := &displayapi.Spec{
			Gpu: needCores,
			Mem: memBytes,
		}

		if memBytes == 0 {
			var deviceMem int64
			if dev, ok := devicesInfo[ctnt.Name]; ok {
				for _, dev := range dev.Dev {
					deviceMem += int64(dev.DeviceMem)
				}
			}
			spec.Mem = deviceMem
		}

		podSpec[ctnt.Name] = spec
	}

	return podSpec
}

func (disp *Display) getPodUsage(pod *v1.Pod) map[string]*displayapi.Devices {
	podUsage := make(map[string]*displayapi.Devices)

	for _, stat := range pod.Status.ContainerStatuses {
		contName := stat.Name
		contID := strings.TrimPrefix(stat.ContainerID, fmt.Sprintf("%s://", disp.containerRuntimeManager.RuntimeName()))
		if len(contID) == 0 {
			continue
		}
		klog.V(4).Infof("Get container %s usage", contID)

		containerInfo, err := disp.containerRuntimeManager.InspectContainer(contID)
		if err != nil {
			klog.Warningf("can't find %s from docker", contID)
			continue
		}

		pidsInContainer, err := disp.containerRuntimeManager.GetPidsInContainers(contID)
		if err != nil {
			klog.Errorf("can't get pids form container %s, %v", contID, err)
			continue
		}
		_, _, deviceNames := utils.GetGPUData(containerInfo.Annotations)
		devicesUsage := make([]*displayapi.DeviceInfo, 0)
		for _, deviceName := range deviceNames {
			if utils.IsValidGPUPath(deviceName) {
				node := disp.tree.Query(deviceName)
				if usage := disp.getDeviceUsage(pidsInContainer, node.Meta.ID); usage != nil {
					usage.DeviceMem = float32(node.Meta.TotalMemory >> 20)
					devicesUsage = append(devicesUsage, usage)
				}
			}
		}

		if len(devicesUsage) > 0 {
			podUsage[contName] = &displayapi.Devices{
				Dev: devicesUsage,
			}
		}
	}

	return podUsage
}

// Version returns version of GPU manager
func (disp *Display) Version(context.Context, *google_protobuf1.Empty) (*displayapi.VersionResponse, error) {
	resp := &displayapi.VersionResponse{
		Version: version.Get().String(),
	}

	return resp, nil
}

func (disp *Display) getDeviceUsage(pidsInCont []int, deviceIdx int) *displayapi.DeviceInfo {
	nvml.Init()
	defer nvml.Shutdown()

	dev, err := nvml.DeviceGetHandleByIndex(uint(deviceIdx))
	if err != nil {
		klog.Warningf("can't find device %d, error %s", deviceIdx, err)
		return nil
	}

	processSamples, err := dev.DeviceGetProcessUtilization(1024, time.Second)
	if err != nil {
		klog.Warningf("can't get processes utilization from device %d, error %s", deviceIdx, err)
		return nil
	}

	// 在该 GPU 卡设备上的进程数
	processOnDevices, err := dev.DeviceGetComputeRunningProcesses(1024)
	if err != nil {
		klog.Warningf("can't get processes info from device %d, error %s", deviceIdx, err)
		return nil
	}

	busID, err := dev.DeviceGetPciInfo()
	if err != nil {
		klog.Warningf("can't get pci info from device %d, error %s", deviceIdx, err)
		return nil
	}

	sort.Slice(pidsInCont, func(i, j int) bool {
		return pidsInCont[i] < pidsInCont[j]
	})

	usedMemory := uint64(0)
	usedPids := make([]int32, 0)
	usedGPU := uint(0)
	for _, info := range processOnDevices {
		idx := sort.Search(len(pidsInCont), func(pivot int) bool {
			return pidsInCont[pivot] >= int(info.Pid)
		})

		if idx < len(pidsInCont) && pidsInCont[idx] == int(info.Pid) {
			usedPids = append(usedPids, int32(pidsInCont[idx]))
			usedMemory += info.UsedGPUMemory
		}
	}

	for _, sample := range processSamples {
		idx := sort.Search(len(pidsInCont), func(pivot int) bool {
			return pidsInCont[pivot] >= int(sample.Pid)
		})

		if idx < len(pidsInCont) && pidsInCont[idx] == int(sample.Pid) {
			usedGPU += sample.SmUtil
		}
	}

	return &displayapi.DeviceInfo{
		Id:      busID.BusID,
		CardIdx: fmt.Sprintf("%d", deviceIdx),
		Gpu:     float32(usedGPU),
		Mem:     float32(usedMemory >> 20),
		Pids:    usedPids,
	}
}

func getUserName(pod *v1.Pod) string {
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "SUBMITTER" {
			return env.Value
		}
	}

	return ""
}

type gpuUtilDesc struct{}
type gpuUtilSpecDesc struct{}
type gpuMemoryDesc struct{}
type gpuMemorySpecDesc struct{}

var (
	defaultMetricLabels   = []string{"pod_name", "namespace", "node", "container_name"}
	utilDescBuilder       = gpuUtilDesc{}
	utilSpecDescBuilder   = gpuUtilSpecDesc{}
	memoryDescBuilder     = gpuMemoryDesc{}
	memorySpecDescBuilder = gpuMemorySpecDesc{}
)

const (
	metricPodName = iota
	metricNamespace
	metricNodeName
	metricContainerName
)

func (gpuUtilDesc) getDescribeDesc() *prometheus.Desc {
	return prometheus.NewDesc("container_gpu_utilization", "gpu utilization", []string{"gpu"}, nil)
}

func (gpuUtilSpecDesc) getDescribeDesc() *prometheus.Desc {
	return prometheus.NewDesc("container_request_gpu_utilization", "request of gpu utilization", []string{"req_of_gpu"}, nil)
}

func (gpuUtilDesc) getMetricDesc() *prometheus.Desc {
	return prometheus.NewDesc("container_gpu_utilization", "gpu utilization", append(defaultMetricLabels, "gpu"), nil)
}

func (gpuUtilSpecDesc) getMetricDesc() *prometheus.Desc {
	return prometheus.NewDesc("container_request_gpu_utilization", "request of gpu utilization", append(defaultMetricLabels, "req_of_gpu"), nil)
}

func (gpuMemoryDesc) getDescribeDesc() *prometheus.Desc {
	return prometheus.NewDesc("container_gpu_memory_total", "gpu memory usage in MiB", []string{"gpu_memory"}, nil)
}

func (gpuMemorySpecDesc) getDescribeDesc() *prometheus.Desc {
	return prometheus.NewDesc("container_request_gpu_memory", "request of gpu memory in MiB", []string{"req_of_gpu_memory"}, nil)
}

func (gpuMemoryDesc) getMetricDesc() *prometheus.Desc {
	return prometheus.NewDesc("container_gpu_memory_total", "gpu memory usage in MiB", append(defaultMetricLabels, "gpu_memory"), nil)
}

func (gpuMemorySpecDesc) getMetricDesc() *prometheus.Desc {
	return prometheus.NewDesc("container_request_gpu_memory", "request of gpu memory in MiB", append(defaultMetricLabels, "req_of_gpu_memory"), nil)
}

// Describe implements prometheus Collector interface
func (disp *Display) Describe(ch chan<- *prometheus.Desc) {
	ch <- utilDescBuilder.getDescribeDesc()
	ch <- utilSpecDescBuilder.getDescribeDesc()
	ch <- memoryDescBuilder.getDescribeDesc()
	ch <- memorySpecDescBuilder.getDescribeDesc()
}

// Collect implements prometheus Collector interface
func (disp *Display) Collect(ch chan<- prometheus.Metric) {
	for _, pod := range watchdog.GetActivePods() {
		valueLabels := make([]string, len(defaultMetricLabels))
		valueLabels[metricPodName] = pod.Name
		valueLabels[metricNamespace] = pod.Namespace
		valueLabels[metricNodeName] = pod.Spec.NodeName

		podUsage := disp.getPodUsage(pod)
		podSpec := disp.getPodSpec(pod, podUsage)
		// Usage of container
		for contName, devicesStat := range podUsage {
			valueLabels[metricContainerName] = contName

			var totalUtils, totalMemory float32
			for _, perDeviceStat := range devicesStat.Dev {
				totalUtils += perDeviceStat.Gpu
				totalMemory += perDeviceStat.Mem

				gpuID := fmt.Sprintf("gpu%s", perDeviceStat.CardIdx)
				if perDeviceStat.Gpu >= 0 {
					ch <- prometheus.MustNewConstMetric(utilDescBuilder.getMetricDesc(),
						prometheus.GaugeValue, float64(perDeviceStat.Gpu), append(valueLabels, gpuID)...)
				}

				if perDeviceStat.Mem >= 0 {
					ch <- prometheus.MustNewConstMetric(memoryDescBuilder.getMetricDesc(),
						prometheus.GaugeValue, float64(perDeviceStat.Mem), append(valueLabels, gpuID)...)
				}
			}

			if totalUtils >= 0 {
				ch <- prometheus.MustNewConstMetric(utilDescBuilder.getMetricDesc(),
					prometheus.GaugeValue, float64(totalUtils), append(valueLabels, "total")...)
			}

			if totalMemory >= 0 {
				ch <- prometheus.MustNewConstMetric(memoryDescBuilder.getMetricDesc(),
					prometheus.GaugeValue, float64(totalMemory), append(valueLabels, "total")...)
			}
		}
		// Spec of container
		for contName, spec := range podSpec {
			valueLabels[metricContainerName] = contName

			ch <- prometheus.MustNewConstMetric(utilSpecDescBuilder.getMetricDesc(),
				prometheus.GaugeValue, float64(spec.Gpu), append(valueLabels, "total")...)
			ch <- prometheus.MustNewConstMetric(memorySpecDescBuilder.getMetricDesc(),
				prometheus.GaugeValue, float64(spec.Mem), append(valueLabels, "total")...)
		}
	}
}
