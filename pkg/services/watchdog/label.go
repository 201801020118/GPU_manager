package watchdog

import (
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"os"
	"regexp"
	"strings"
	"time"

	"k8s.io/klog"
)

const (
	gpuModelLabel = "gaia.ideal.com/gpu-model"
)

type labelFunc interface {
	GetLabel() string
}

type nodeLabeler struct { //节点标签结构体
	hostName    string
	client      v1core.CoreV1Interface
	labelMapper map[string]labelFunc
	modelLabels string
}

type modelFunc struct{}
type stringFunc string

var modelFn = modelFunc{}

func (m modelFunc) GetLabel() (model string) {
	if err := nvml.Init(); err != nil {
		klog.Warningf("Can't initialize nvml library, %v", err)
		return
	}

	defer nvml.Shutdown()

	// Assume all devices on this node are the same model
	dev, err := nvml.DeviceGetHandleByIndex(0)
	if err != nil {
		klog.Warningf("Can't get device 0 information, %v", err)
		return
	}

	rawName, err := dev.DeviceGetName()
	if err != nil {
		klog.Warningf("Can't get device name, %v", err)
		return
	}

	klog.V(4).Infof("GPU name: %s", rawName)
	return strings.ReplaceAll(rawName, " ", "-")
}

func (s stringFunc) GetLabel() string {
	return string(s)
}

var modelNameSplitPattern = regexp.MustCompile("\\s+")

func getTypeName(name string) string {
	splits := modelNameSplitPattern.Split(name, -1)

	if len(splits) > 2 {
		return splits[1]
	}

	klog.V(4).Infof("GPU name splits: %v", splits)

	return ""
}

// NewNodeLabeler returns a new nodeLabeler
func NewNodeLabeler(client v1core.CoreV1Interface, hostname string, labels map[string]string) *nodeLabeler {
	if len(hostname) == 0 {
		hostname, _ = os.Hostname()
	}

	klog.V(2).Infof("Labeler for hostname %s", hostname)

	labelMapper := make(map[string]labelFunc)

	modelLabels := modelFn.GetLabel()
	for k, v := range labels {
		if k == gpuModelLabel {
			modelLabels = modelFn.GetLabel()
		} else {
			labelMapper[k] = stringFunc(v)
		}
	}

	return &nodeLabeler{
		hostName:    hostname,
		client:      client,
		labelMapper: labelMapper,
		modelLabels: modelLabels,
	}
}

func (nl *nodeLabeler) Run() error {
	err := wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		node, err := nl.client.Nodes().Get(nl.hostName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for k, fn := range nl.labelMapper {
			l := fn.GetLabel()
			if len(l) == 0 {
				klog.Warningf("Empty label for %s", k)
				continue
			}

			klog.V(2).Infof("Label %s %s=%s", nl.hostName, k, l)
			node.Labels[k] = l
		}

		if nl.modelLabels != "" {
			node.Labels["gpu-model"] = nl.modelLabels
		}

		_, updateErr := nl.client.Nodes().Update(node)
		if updateErr != nil {
			if errors.IsConflict(updateErr) {
				return false, nil
			}
			return true, updateErr
		}

		return true, nil
	})

	if err != nil {
		return err
	}

	klog.V(2).Infof("Auto label is running")

	return nil
}
