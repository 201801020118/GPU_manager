package watchdog

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"tkestack.io/gpu-manager/pkg/types"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
)

func init() {
	flag.Set("v", "4")
	flag.Set("logtostderr", "true")
}

func TestWatchdog(t *testing.T) {
	flag.Parse()
	podName := "testpod"
	podUID := "testuid"
	ns := "test-ns"
	containerName := "test-container"
	// create pod with fake client
	k8sclient := fake.NewSimpleClientset()
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			UID:  k8stypes.UID(podUID),
		},
		Spec: v1.PodSpec{Containers: []v1.Container{
			{
				Name: containerName,
				Resources: v1.ResourceRequirements{
					Limits: v1.ResourceList{
						types.VCoreAnnotation:   resource.MustParse(fmt.Sprintf("%d", 1)),
						types.VMemoryAnnotation: resource.MustParse(fmt.Sprintf("%d", 1)),
					},
				},
			},
		}},
		Status: v1.PodStatus{Phase: v1.PodRunning},
	}
	k8sclient.CoreV1().Pods(ns).Create(pod)

	// create watchdog and run
	NewPodCacheForTest(k8sclient)

	// check if watchdog work well
	err := wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		activepods := GetActivePods()
		if v, ok := activepods[podUID]; !ok || v.Name != podName {
			t.Logf("can't find pod %s", podName)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("test failed: %s", err.Error())
	}
}
