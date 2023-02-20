package main

import (
	goflag "flag"
	"fmt"
	"os"

	"k8s.io/klog"

	"tkestack.io/gpu-manager/cmd/manager/app"
	"tkestack.io/gpu-manager/cmd/manager/options"
	"tkestack.io/gpu-manager/pkg/flags"
	"tkestack.io/gpu-manager/pkg/logs"
	"tkestack.io/gpu-manager/pkg/version"

	"github.com/spf13/pflag"
)

func main() { //启动函数，主函数
	klog.InitFlags(nil)
	opt := options.NewOptions()
	opt.AddFlags(pflag.CommandLine)

	flags.InitFlags()
	goflag.CommandLine.Parse([]string{})
	logs.InitLogs()
	defer logs.FlushLogs()

	version.PrintAndExitIfRequested()

	if err := app.Run(opt); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
