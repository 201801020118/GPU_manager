package policy

import (
	"strconv"

	"github.com/spf13/cast"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"tkestack.io/gpu-manager/pkg/types"
	"tkestack.io/gpu-manager/pkg/utils"
)

// IsPolicyPod 检测该 Pod 是否是通过调度策略调度的 Pod
func IsPolicyPod(pod *corev1.Pod) bool {
	if pod.Annotations == nil {
		return false
	}

	if _, ok := pod.Annotations[types.PolicyNameAnnotation]; !ok {
		return false
	}

	return true
}

func GetRealGPU(pod *corev1.Pod, containerName string, needCores int64) int64 {
	containerIndex, err := utils.GetContainerIndexByName(pod, containerName)
	if err != nil {
		klog.Fatalf("get container index failed: %s", err.Error())
	}

	if coreStr, ok := pod.Annotations[types.RealVCoreAnnotationPrefix+strconv.Itoa(containerIndex)]; ok {
		needCores = cast.ToInt64(coreStr)
	}
	return needCores
}
