package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	displayapi "tkestack.io/gpu-manager/pkg/api/runtime/display"
	virtualmanagerapi "tkestack.io/gpu-manager/pkg/api/runtime/virtual-manager"
	"tkestack.io/gpu-manager/pkg/config"
	deviceFactory "tkestack.io/gpu-manager/pkg/device"
	containerRuntime "tkestack.io/gpu-manager/pkg/runtime"
	allocFactory "tkestack.io/gpu-manager/pkg/services/allocator"
	"tkestack.io/gpu-manager/pkg/services/response"

	// Register allocator controller
	_ "tkestack.io/gpu-manager/pkg/services/allocator/register"
	"tkestack.io/gpu-manager/pkg/services/display"
	vitrual_manager "tkestack.io/gpu-manager/pkg/services/virtual-manager"
	"tkestack.io/gpu-manager/pkg/services/volume"
	"tkestack.io/gpu-manager/pkg/services/watchdog"
	"tkestack.io/gpu-manager/pkg/types"
	"tkestack.io/gpu-manager/pkg/utils"

	systemd "github.com/coreos/go-systemd/daemon"
	google_protobuf1 "github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type managerImpl struct {
	config *config.Config

	allocator      allocFactory.GPUTopoService     //gpu容器调度分配
	displayer      *display.Display                //gpu使用情况可视化服务
	virtualManager *vitrual_manager.VirtualManager //vgpu manager

	bundleServer map[string]ResourceServer //资源服务器(暂时不知道怎么使用的)
	srv          *grpc.Server              //将gpu display server注册到此grpc server中，也通过srv像k8s注册设备插件
	tcpServer    *grpc.Server              // 监听指定端口的 grpc 服务器, 用于对外通过端口暴露 grpc 服务
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

// NewManager creates and returns a new managerImpl struct
// NewManager 建造并返回了一个manager的结构体接口
func NewManager(cfg *config.Config) Manager {
	manager := &managerImpl{
		config:       cfg,
		bundleServer: make(map[string]ResourceServer),
		srv:          grpc.NewServer(),
		tcpServer:    grpc.NewServer(),
	}

	return manager
}

// Ready tells the manager whether all bundle servers are truely running
// Ready 告诉manager所有的包服务器是否真的运行
func (m *managerImpl) Ready() bool {
	readyServers := 0

	for _, ins := range m.bundleServer {
		if err := utils.WaitForServer(ins.SocketName()); err == nil {
			readyServers++
			klog.V(2).Infof("Server %s is ready, readyServers: %d", ins.SocketName(), readyServers)
			continue
		}

		return false
	}

	return readyServers > 0 && readyServers == len(m.bundleServer)
}

// #lizard forgives
func (m *managerImpl) Run() error {
	if err := m.validExtraConfig(m.config.ExtraConfigPath); err != nil { //校验配置，额外配置路径
		klog.Errorf("Can not load extra config, err %s", err)

		return err
	}

	if m.config.Driver == "" { //判断配置，driver是否为空
		return fmt.Errorf("you should define a driver")
	}

	if len(m.config.VolumeConfigPath) > 0 {
		//如果配置了配置。VolumeConfigPath，就初始化VolumeManager对象，执行VolumeManager.Run函数（解析volume.conf，创建对应的目录/硬链接，拷贝对应的文件）
		volumeManager, err := volume.NewVolumeManager(m.config.VolumeConfigPath, m.config.EnableShare)
		if err != nil {
			klog.Errorf("Can not create volume managerImpl, err %s", err)
			return err
		}

		if err := volumeManager.Run(); err != nil { //运行卷管理器
			klog.Errorf("Can not start volume managerImpl, err %s", err)
			return err
		}
	}

	sent, err := systemd.SdNotify(true, "READY=1\n") //向守护进程发送消息，systemd发送通知信息给daemonREADY=1
	//相关知识：https://cloud.tencent.com/developer/article/1516125
	if err != nil {
		klog.Errorf("Unable to send systemd daemon successful start message: %v\n", err)
	}

	if !sent {
		klog.Errorf("Unable to set Type=notify in systemd service file?")
	}

	var ( //初始化clientSet， 使用sharedInformer创建pod cache（当前主机的所有pod）,在185行执行
		client    *kubernetes.Clientset
		clientCfg *rest.Config
	)

	clientCfg, err = clientcmd.BuildConfigFromFlags("", m.config.KubeConfig) //加载配置文件（加载kube的配置文件）
	if err != nil {
		return fmt.Errorf("invalid client config: err(%v)", err)
	}

	client, err = kubernetes.NewForConfig(clientCfg) //生成客户端配置
	if err != nil {
		return fmt.Errorf("can not generate client from config: error(%v)", err)
	}

	/*
		一些知识：
		什么是容器运行时：掌控容器运行的整个生命周期，主要提供的功能如下：
		制定容器镜像格式
		构建容器镜像 docker build
		管理容器镜像 docker images
		管理容器实例 docker ps
		运行容器 docker run
		实现容器镜像共享 docker pull/push
		相关链接：
		https://zhuanlan.zhihu.com/p/102171749
		https://www.cnblogs.com/lianngkyle/p/15086099.html
	*/

	containerRuntimeManager, err := containerRuntime.NewContainerRuntimeManager( //初始化ContainerRuntimeManager对象， 和容器运行时交互
		m.config.CgroupDriver, m.config.ContainerRuntimeEndpoint, m.config.RequestTimeout)
	if err != nil {
		klog.Errorf("can't create container runtime manager: %v", err)
		return err
	}
	klog.V(2).Infof("Container runtime manager is running")

	klog.V(2).Infof("Load container response data")
	responseManager := response.NewResponseManager()                                //创建返回数据的结构体
	if err := responseManager.LoadFromFile(m.config.DevicePluginPath); err != nil { //获取设备插件（device-plugin）的容器的返回数据
		klog.Errorf("can't load container response data, %+#v", err)
		return err
	}

	// 先初始化 VirtualManager[1]
	m.virtualManager = vitrual_manager.NewVirtualManager(m.config, containerRuntimeManager, responseManager)

	// 然后初始化并启动 PodInformer, 同时在 PodInformer 中注册 VirtualManager.UpdateGPU handler
	watchdog.NewPodCache(client, m.config.Hostname, m.virtualManager.UpdateGPU)
	klog.V(2).Infof("Watchdog is running")

	labeler := watchdog.NewNodeLabeler(client.CoreV1(), m.config.Hostname, m.config.NodeLabels) //初始化nodeLabeler对象， 快速更新节点标签
	if err := labeler.Run(); err != nil {
		return err
	}

	// 执行VirtualManager的Run函数[2]
	m.virtualManager.Run()

	treeInitFn := deviceFactory.NewFuncForName(m.config.Driver)
	//从deviceFactory设备工厂函数中根据config.Driver类型返回一个有名函数，
	//用于获取实现了接口的具体实例对象
	tree := treeInitFn(m.config)

	tree.Init("") //执行了接口的Init和Update函数（gpu拓扑结构感知）
	tree.Update() //更新树节点显存使用情况，还有pid列表

	initAllocator := allocFactory.NewFuncForName(m.config.Driver)
	if initAllocator == nil {
		return fmt.Errorf("can not find allocator for %s", m.config.Driver)
	}

	m.allocator = initAllocator(m.config, tree, client, responseManager)
	m.displayer = display.NewDisplay(m.config, tree, containerRuntimeManager)

	klog.V(2).Infof("Starting the GRPC server, driver %s, queryPort %d", m.config.Driver, m.config.QueryPort)
	m.setupGRPCService()
	m.setupTCPGRPCService()
	mux, err := m.setupGRPCGatewayService()
	if err != nil {
		return err
	}
	m.setupMetricsService(mux)

	go func() {
		displayListenHandler := net.JoinHostPort(m.config.QueryAddr, strconv.Itoa(m.config.QueryPort))
		if err := http.ListenAndServe(displayListenHandler, mux); err != nil {
			klog.Fatalf("failed to serve connections: %v", err)
		}
	}()

	return m.runServer()
}

func (m *managerImpl) setupGRPCService() {
	vcoreServer := newVcoreServer(m)
	vmemoryServer := newVmemoryServer(m)

	m.bundleServer[types.VCoreAnnotation] = vcoreServer
	m.bundleServer[types.VMemoryAnnotation] = vmemoryServer

	displayapi.RegisterGPUDisplayServer(m.srv, m)
}

func (m *managerImpl) setupTCPGRPCService() {
	virtualmanagerapi.RegisterVirtualManagerServer(m.tcpServer, m)
}

func (m *managerImpl) setupGRPCGatewayService() (*http.ServeMux, error) {
	mux := http.NewServeMux()
	displayMux := runtime.NewServeMux()

	mux.Handle("/", displayMux)
	mux.HandleFunc("/debug/pprof/", pprof.Index)

	go func() {
		if err := displayapi.RegisterGPUDisplayHandlerFromEndpoint(context.Background(), displayMux, types.ManagerSocket, utils.DefaultDialOptions); err != nil {
			klog.Fatalf("Register display service failed, error %s", err)
		}
	}()

	return mux, nil
}

func (m *managerImpl) setupMetricsService(mux *http.ServeMux) {
	r := prometheus.NewRegistry()

	r.MustRegister(m.displayer)

	mux.Handle("/metric", promhttp.HandlerFor(r, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError}))
}

func (m *managerImpl) runServer() error {
	for name, srv := range m.bundleServer {
		klog.V(2).Infof("Server %s is running", name)
		go srv.Run()
	}

	tcpListen, err := net.Listen("tcp", fmt.Sprintf(":%d", m.config.Port))
	if err != nil {
		return err
	}
	go func() {
		klog.Infof("gRPC server starting on %s", tcpListen.Addr())
		if err := m.tcpServer.Serve(tcpListen); err != nil {
			klog.Fatal(err)
		}
	}()

	err = syscall.Unlink(types.ManagerSocket)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	l, err := net.Listen("unix", types.ManagerSocket)
	if err != nil {
		return err
	}

	klog.V(2).Infof("Server is ready at %s", types.ManagerSocket)

	return m.srv.Serve(l)
}

func (m *managerImpl) Stop() {
	for name, srv := range m.bundleServer {
		klog.V(2).Infof("Server %s is stopping", name)
		srv.Stop()
	}
	m.srv.Stop()
	klog.Fatal("Stop server")
}

func (m *managerImpl) validExtraConfig(path string) error {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}

		cfg := make(map[string]*config.ExtraConfig)
		if err := json.NewDecoder(file).Decode(&cfg); err != nil {
			return err
		}
	}

	return nil
}

/** device plugin interface */
func (m *managerImpl) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	return m.allocator.Allocate(ctx, reqs)
}

func (m *managerImpl) ListAndWatchWithResourceName(resourceName string, e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	return m.allocator.ListAndWatchWithResourceName(resourceName, e, s)
}

func (m *managerImpl) GetDevicePluginOptions(ctx context.Context, e *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return m.allocator.GetDevicePluginOptions(ctx, e)
}

func (m *managerImpl) PreStartContainer(ctx context.Context, req *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return m.allocator.PreStartContainer(ctx, req)
}

/** statistics interface */
func (m *managerImpl) PrintGraph(ctx context.Context, req *google_protobuf1.Empty) (*displayapi.GraphResponse, error) {
	return m.displayer.PrintGraph(ctx, req)
}

/** statistics interface */
func (m *managerImpl) PrintGpuinfo(ctx context.Context, req *google_protobuf1.Empty) (*displayapi.GraphResponse, error) {

	resp := &displayapi.GraphResponse{}
	data, err := m.displayer.PrintGpuinfo(ctx, req)
	if err != nil {
		return resp, err
	}

	infos := []*VgpuNodeInfo{}
	if err := json.Unmarshal([]byte(data.Graph), &infos); err != nil {
		return nil, err
	}

	for _, info := range infos {

		vgpuinfo, err := m.allocator.Vgpuinfo(ctx, &virtualmanagerapi.VgpuInfoRequest{strconv.Itoa(info.Gpuid)})
		if err != nil {
			klog.Infof("allocator.Vgpuinfo failed: %v", err.Error())
			continue
		}
		//running pod

		klog.Infof("gpuid %d taskNumS: %d", info.Gpuid, vgpuinfo.TaskNums)
		info.Pids = int(vgpuinfo.TaskNums)
	}

	jdata, err := json.Marshal(infos)
	if err != nil {
		return resp, err
	}

	resp.Graph = fmt.Sprintf("%s", jdata)
	return resp, nil

}

func (m *managerImpl) PrintUsages(ctx context.Context, req *google_protobuf1.Empty) (*displayapi.UsageResponse, error) {
	return m.displayer.PrintUsages(ctx, req)
}

func (m *managerImpl) PrintGpuDetails(ctx context.Context, req *displayapi.ContUsageRequest) (*displayapi.ContUsageResponse, error) {
	return m.displayer.PrintGpuDetails(ctx, req)
}

func (m *managerImpl) Version(ctx context.Context, req *google_protobuf1.Empty) (*displayapi.VersionResponse, error) {
	return m.displayer.Version(ctx, req)
}

func (m *managerImpl) Vgpuinfo(ctx context.Context, req *virtualmanagerapi.VgpuInfoRequest) (*virtualmanagerapi.VgpuInfoResponse, error) {
	return m.allocator.Vgpuinfo(ctx, req)
}

func (m *managerImpl) RegisterToKubelet() error {
	socketFile := filepath.Join(m.config.DevicePluginPath, types.KubeletSocket)
	dialOptions := []grpc.DialOption{grpc.WithInsecure(), grpc.WithDialer(utils.UnixDial), grpc.WithBlock(), grpc.WithTimeout(time.Second * 5)}

	conn, err := grpc.Dial(socketFile, dialOptions...)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)

	for _, srv := range m.bundleServer {
		req := &pluginapi.RegisterRequest{
			Version:      pluginapi.Version,
			Endpoint:     path.Base(srv.SocketName()),
			ResourceName: srv.ResourceName(),
			Options:      &pluginapi.DevicePluginOptions{PreStartRequired: true},
		}

		klog.V(2).Infof("Register to kubelet with endpoint %s", req.Endpoint)
		_, err = client.Register(context.Background(), req)
		if err != nil {
			return err
		}
	}

	return nil
}
