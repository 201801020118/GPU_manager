module tkestack.io/gpu-manager

go 1.14

replace tkestack.io/nvml => github.com/tkestack/go-nvml v0.0.0-20191217064248-7363e630a33e

require (
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e
	github.com/docker/go-units v0.4.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gkarthiks/cndev v0.0.0-20220825024840-68fc0486df12
	github.com/godbus/dbus v0.0.0-20181101234600-2ff6f7ffd60f // indirect
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/mxpv/nvml-go v0.0.0-20180227003457-e07f8c26812d
	github.com/opencontainers/runc v1.0.0-rc9
	github.com/opencontainers/runtime-spec v1.0.2 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/spf13/cast v1.3.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.2
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b
	google.golang.org/genproto v0.0.0-20210924002016-3dee208752a0
	google.golang.org/grpc v1.40.0
	k8s.io/api v0.25.0
	k8s.io/apimachinery v0.25.0
	k8s.io/client-go v0.25.0
	k8s.io/cri-api v0.17.4
	k8s.io/klog v1.0.0
	k8s.io/kubectl v0.17.4
	k8s.io/kubelet v0.19.0
)
