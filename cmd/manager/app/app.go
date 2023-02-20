package app

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tkestack.io/gpu-manager/cmd/manager/options"
	"tkestack.io/gpu-manager/pkg/config"
	"tkestack.io/gpu-manager/pkg/server"
	"tkestack.io/gpu-manager/pkg/types"
	"tkestack.io/gpu-manager/pkg/utils"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// #lizard forgives
func Run(opt *options.Options) error { //run的主代码，在app.go中
	cfg := &config.Config{ //流程1：解析Options对象（配置参数），用来填充Config对象
		Driver:                   opt.Driver,
		QueryPort:                opt.QueryPort,
		QueryAddr:                opt.QueryAddr,
		KubeConfig:               opt.KubeConfigFile,
		SamplePeriod:             time.Duration(opt.SamplePeriod) * time.Second,
		VCudaRequestsQueue:       make(chan *types.VCudaRequest, 10),
		DevicePluginPath:         pluginapi.DevicePluginPath,
		VirtualManagerPath:       opt.VirtualManagerPath,
		VolumeConfigPath:         opt.VolumeConfigPath,
		EnableShare:              opt.EnableShare,
		AllocationCheckPeriod:    time.Duration(opt.AllocationCheckPeriod) * time.Second,
		CheckpointPath:           opt.CheckpointPath,
		ContainerRuntimeEndpoint: opt.ContainerRuntimeEndpoint,
		CgroupDriver:             opt.CgroupDriver,
		RequestTimeout:           opt.RequestTimeout,
		Port:                     opt.Port,
	}

	if len(opt.HostnameOverride) > 0 { //获取主机名
		cfg.Hostname = opt.HostnameOverride
	}

	if len(opt.ExtraPath) > 0 { //额外配置文件路径
		cfg.ExtraConfigPath = opt.ExtraPath
	}

	if len(opt.DevicePluginPath) > 0 { //配置device-plugin路径
		cfg.DevicePluginPath = opt.DevicePluginPath
	}

	cfg.NodeLabels = make(map[string]string) //获取节点标签
	for _, item := range strings.Split(opt.NodeLabels, ",") {
		if len(item) > 0 {
			kvs := strings.SplitN(item, "=", 2)
			if len(kvs) == 2 {
				cfg.NodeLabels[kvs[0]] = kvs[1]
			} else {
				klog.Warningf("malformed node labels: %v", kvs)
			}
		}
	}

	srv := server.NewManager(cfg) //流程2：初始化managerImpl对象（实现Manager接口），执行接口中定义的Run函数
	go srv.Run()                  //流程3：执行入口

	waitTimer := time.NewTimer(opt.WaitTimeout)
	for !srv.Ready() {
		select {
		case <-waitTimer.C:
			klog.Warningf("Wait too long for server ready, restarting")
			os.Exit(1)
		default:
			klog.Infof("Wait for internal server ready")
		}
		time.Sleep(time.Second)
	}
	waitTimer.Stop()

	if err := srv.RegisterToKubelet(); err != nil {
		return err
	}

	devicePluginSocket := filepath.Join(cfg.DevicePluginPath, types.KubeletSocket) //设置sock文件的位置，通过config中的配置项设置
	watcher, err := utils.NewFSWatcher(cfg.DevicePluginPath)
	if err != nil {
		log.Println("Failed to created FS watcher.")
		os.Exit(1)
	}
	defer watcher.Close()

	for {
		select {
		case event := <-watcher.Events:
			if event.Name == devicePluginSocket && event.Op&fsnotify.Create == fsnotify.Create {
				time.Sleep(time.Second)
				klog.Fatalf("inotify: %s created, restarting.", devicePluginSocket)
			}
		case err := <-watcher.Errors:
			klog.Fatalf("inotify: %s", err)
		}
	}
	return nil
}
