package nvidia

import (
	"flag"
	"testing"

	"tkestack.io/gpu-manager/pkg/device/nvidia"
)

func init() {
	flag.Set("v", "4")
	flag.Set("logtostderr", "true")
}

func TestLink(t *testing.T) {
	flag.Parse()
	obj := nvidia.NewNvidiaTree(nil)
	tree, _ := obj.(*nvidia.NvidiaTree)

	testCase1 :=
		`    GPU0    GPU1    GPU2    GPU3    GPU4    GPU5
GPU0      X      PIX     PHB     PHB     SOC     SOC
GPU1     PIX      X      PHB     PHB     SOC     SOC
GPU2     PHB     PHB      X      PIX     SOC     SOC
GPU3     PHB     PHB     PIX      X      SOC     SOC
GPU4     SOC     SOC     SOC     SOC      X      PIX
GPU5     SOC     SOC     SOC     SOC     PIX      X
`
	tree.Init(testCase1)
	algo := NewLinkMode(tree)

	expectCase1 := []string{
		"/dev/nvidia0",
		"/dev/nvidia1",
		"/dev/nvidia2",
	}

	cores := int64(3 * nvidia.HundredCore)
	pass, should, but := examining(expectCase1, algo.Evaluate(cores, 0))
	if !pass {
		t.Fatalf("Evaluate function got wrong, should be %s, but %s", should, but)
	}

	tree.MarkOccupied(&nvidia.NvidiaNode{
		Meta: nvidia.DeviceMeta{
			MinorID: 2,
		},
	}, cores, 0)

	expectCase2 := []string{
		"/dev/nvidia0",
		"/dev/nvidia1",
	}

	cores = int64(2 * nvidia.HundredCore)
	pass, should, but = examining(expectCase2, algo.Evaluate(cores, 0))
	if !pass {
		t.Fatalf("Evaluate function got wrong, should be %s, but %s", should, but)
	}
}
