package logs

import (
	"log"
	"time"

	"google.golang.org/grpc/grpclog"
	"k8s.io/klog"
)

// klogWriter serves as a bridge between the standard log package and the klog package.
type klogWriter struct{}

// Write implements the io.Writer interface.
func (gw klogWriter) Write(data []byte) (n int, err error) {
	klog.Info(string(data))
	return len(data), nil
}

// InitLogs initializes logs the way we want for kubernetes.
func InitLogs() {
	logger := klogWriter{}
	log.SetOutput(logger)
	log.SetFlags(0)

	grpclog.SetLogger(logger)
	// The default klog flush interval is 30 seconds, which is frighteningly long.
	go func() {
		for range time.Tick(time.Second) {
			klog.Flush()
		}
	}()
}

//FlushLogs calls klog.Flush to flush all pending log I/O
func FlushLogs() {
	klog.Flush()
}

//Fatal wraps klog.FatalDepth
func (gw klogWriter) Fatal(args ...interface{}) {
	klog.FatalDepth(1, args...)
}

//Fatalf wraps klog.Fatalf
func (gw klogWriter) Fatalf(format string, args ...interface{}) {
	klog.Fatalf(format, args...)
}

//Fatalln wraps klog.Fatalln
func (gw klogWriter) Fatalln(args ...interface{}) {
	klog.Fatalln(args...)
}

//Print wraps klog.InfoDepth
func (gw klogWriter) Print(args ...interface{}) {
	klog.InfoDepth(1, args...)
}

//Printf wraps klog.V(2).Infof
func (gw klogWriter) Printf(format string, args ...interface{}) {
	klog.V(2).Infof(format, args...)
}

//Println wraps klog.Info
func (gw klogWriter) Println(args ...interface{}) {
	klog.Info(args...)
}
