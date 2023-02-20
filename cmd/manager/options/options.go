package options

import (
	"time"

	"github.com/spf13/pflag"
)

const (
	DefaultDriver                   = "nvidia" //将其修改为寒武纪的驱动
	DefaultQueryPort                = 5678
	DefaultSamplePeriod             = 1
	DefaultVirtualManagerPath       = "/etc/gpu-manager/vm"
	DefaultAllocationCheckPeriod    = 30
	DefaultCheckpointPath           = "/etc/gpu-manager/checkpoint"
	DefaultContainerRuntimeEndpoint = "/var/run/dockershim.sock"
	DefaultCgroupDriver             = "cgroupfs"
	DefaultPort                     = 5679
)

// Options contains plugin information
// 一些device-plugin与k8s关联所需的配置项
type Options struct {
	Driver           string //gpu-manger的驱动，默认为英伟达（nvidia）
	ExtraPath        string //额外配置文件加载路径
	VolumeConfigPath string //volume配置文件路径，volum.conf记录了具体路径关于使用到的cuda命令和nvidia cuda库
	QueryPort        int    //使用prometheus查询metric监听的端口
	//相关知识（prometheus）：https://zhuanlan.zhihu.com/p/267966193
	//相关知识（metric）：https://zhuanlan.zhihu.com/p/107213754
	QueryAddr                string //让prometheus查询metric监听的地址
	KubeConfigFile           string //kubeConfig路劲，获取集群资源信息的凭证
	SamplePeriod             int    //每张GPU执行的时间段，默认单位为秒
	NodeLabels               string //节点自动打标签
	HostnameOverride         string //节点主机名表示，不是实际主机名
	VirtualManagerPath       string //virtual manager配置文件路径
	DevicePluginPath         string //device-plugin注册插件的路径
	EnableShare              bool   //是否启用GPU共享分配
	AllocationCheckPeriod    int    //检查已分配的GPU的间隔，单位为秒
	CheckpointPath           string //checkpoint配置存储的路径，默认值为/etc/gpu-manager（没看懂）
	ContainerRuntimeEndpoint string //容器运行时，默认值为/var/run/dockershim.sock
	CgroupDriver             string //cgroup驱动，默认cgroupfs，还有systemd
	//相关知识：https://blog.csdn.net/chen_haoren/article/details/108773459
	//相关知识：https://zhuanlan.zhihu.com/p/544464235
	RequestTimeout time.Duration //请求容器运行时的超时时间
	WaitTimeout    time.Duration
	Port           int // grpc 服务器监听端口
}

// NewOptions gives a default options template.
// 创建一个默认模板选项，里面包含一些配置项
func NewOptions() *Options {
	return &Options{
		Driver:                   DefaultDriver,
		QueryPort:                DefaultQueryPort,
		QueryAddr:                "localhost",
		SamplePeriod:             DefaultSamplePeriod,
		VirtualManagerPath:       DefaultVirtualManagerPath,
		AllocationCheckPeriod:    DefaultAllocationCheckPeriod,
		CheckpointPath:           DefaultCheckpointPath,
		ContainerRuntimeEndpoint: DefaultContainerRuntimeEndpoint,
		CgroupDriver:             DefaultCgroupDriver,
		RequestTimeout:           time.Second * 5,
		WaitTimeout:              time.Minute,
		Port:                     DefaultPort,
	}
}

// AddFlags add some commandline flags.
func (opt *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&opt.Driver, "driver", opt.Driver, "The driver name for manager")
	fs.StringVar(&opt.ExtraPath, "extra-config", opt.ExtraPath, "The extra config file location")
	fs.StringVar(&opt.VolumeConfigPath, "volume-config", opt.VolumeConfigPath, "The volume config file location")
	fs.IntVar(&opt.QueryPort, "query-port", opt.QueryPort, "port for query statistics information")
	fs.StringVar(&opt.QueryAddr, "query-addr", opt.QueryAddr, "address for query statistics information")
	fs.StringVar(&opt.KubeConfigFile, "kubeconfig", opt.KubeConfigFile, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	fs.IntVar(&opt.SamplePeriod, "sample-period", opt.SamplePeriod, "Sample period for each card, unit second")
	fs.StringVar(&opt.NodeLabels, "node-labels", opt.NodeLabels, "automated label for this node, if empty, node will be only labeled by gpu model")
	fs.StringVar(&opt.HostnameOverride, "hostname-override", opt.HostnameOverride, "If non-empty, will use this string as identification instead of the actual hostname.")
	fs.StringVar(&opt.VirtualManagerPath, "virtual-manager-path", opt.VirtualManagerPath, "configuration path for virtual manager store files")
	fs.StringVar(&opt.DevicePluginPath, "device-plugin-path", opt.DevicePluginPath, "the path for kubelet receive device plugin registration")
	fs.StringVar(&opt.CheckpointPath, "checkpoint-path", opt.CheckpointPath, "configuration path for checkpoint store file")
	fs.BoolVar(&opt.EnableShare, "share-mode", opt.EnableShare, "enable share mode allocation")
	fs.IntVar(&opt.AllocationCheckPeriod, "allocation-check-period", opt.AllocationCheckPeriod, "allocation check period, unit second")
	fs.StringVar(&opt.ContainerRuntimeEndpoint, "container-runtime-endpoint", opt.ContainerRuntimeEndpoint, "container runtime endpoint")
	fs.StringVar(&opt.CgroupDriver, "cgroup-driver", opt.CgroupDriver, "Driver that the kubelet uses to manipulate cgroups on the host.  "+
		"Possible values: 'cgroupfs', 'systemd'")
	fs.DurationVar(&opt.RequestTimeout, "runtime-request-timeout", opt.RequestTimeout,
		"request timeout for communicating with container runtime endpoint")
	fs.DurationVar(&opt.WaitTimeout, "wait-timeout", opt.WaitTimeout, "wait timeout for resource server ready")
	fs.IntVar(&opt.Port, "port", opt.Port, "port for query gpu statistics information")
}
