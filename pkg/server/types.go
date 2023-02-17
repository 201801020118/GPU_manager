package server

import (
	"google.golang.org/grpc"
)

//Manager api
type Manager interface {
	Ready() bool
	Run() error
	RegisterToKubelet() error
}

//ResourceServer api for manager
type ResourceServer interface {
	Run() error
	Stop()
	SocketName() string
	ResourceName() string
}

type resourceServerImpl struct {
	srv        *grpc.Server
	socketFile string

	mgr *managerImpl
}
