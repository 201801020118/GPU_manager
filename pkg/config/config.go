package config

import (
	"time"

	"tkestack.io/gpu-manager/pkg/types"
)

// Config contains the necessary options for the plugin.
// Config 包含了对于plugin（插件）来说必要的选项
type Config struct {
	Driver                   string
	ExtraConfigPath          string
	QueryPort                int
	QueryAddr                string
	KubeConfig               string
	SamplePeriod             time.Duration
	Hostname                 string
	NodeLabels               map[string]string //map[string]string 则表示一个字符串到字符串的映射
	VirtualManagerPath       string
	DevicePluginPath         string
	VolumeConfigPath         string
	EnableShare              bool
	AllocationCheckPeriod    time.Duration
	CheckpointPath           string
	ContainerRuntimeEndpoint string
	CgroupDriver             string
	RequestTimeout           time.Duration
	Port                     int
	//存放cuda请求的channel
	VCudaRequestsQueue chan *types.VCudaRequest
}

// ExtraConfig contains extra options other than Config
type ExtraConfig struct {
	Devices []string `json:"devices,omitempty"`
}
