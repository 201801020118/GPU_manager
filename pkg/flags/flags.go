package flags

import (
	goflag "flag"
	"strings"

	"github.com/spf13/pflag"
)

// WordSepNormalizeFunc changes all flags that contain "_" separators
func WordSepNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	if strings.Contains(name, "_") {
		return pflag.NormalizedName(strings.Replace(name, "_", "-", -1))
	}
	return pflag.NormalizedName(name)
}

// InitFlags normalizes and parses the command line flags
func InitFlags() {
	pflag.CommandLine.SetNormalizeFunc(WordSepNormalizeFunc)
	// Only klog flags will be added
	goflag.CommandLine.VisitAll(func(goflag *goflag.Flag) {
		switch goflag.Name {
		case "logtostderr", "alsologtostderr",
			"v", "stderrthreshold", "vmodule", "log_backtrace_at", "log_dir":
			pflag.CommandLine.AddGoFlag(goflag)
		}
	})

	pflag.Parse()
}
