package config

import (
	"time"

	"tkestack.io/gpu-manager/pkg/types"
)

// Config contains the necessary options for the plugin.
type Config struct {
	Driver                   string
	ExtraConfigPath          string
	QueryPort                int
	QueryAddr                string
	KubeConfig               string
	SamplePeriod             time.Duration
	Hostname                 string
	NodeLabels               map[string]string
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

	VCudaRequestsQueue chan *types.VCudaRequest
}

//ExtraConfig contains extra options other than Config
type ExtraConfig struct {
	Devices []string `json:"devices,omitempty"`
}
