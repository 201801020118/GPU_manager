package policy

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"tkestack.io/gpu-manager/pkg/types"
)

func TestGetRealGPU(t *testing.T) {
	testPod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"ideal.com/real-vcuda-core-0": "10",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "test",
				},
			},
		},
	}

	testPod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"ideal.com/real-vcuda-core-0": "10",
				"ideal.com/real-vcuda-core-1": "30",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "test",
				},
				{
					Name: "test2",
				},
			},
		},
	}

	type args struct {
		pod           *corev1.Pod
		containerName string
		needCores     int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "one",
			args: args{
				pod:           testPod1,
				containerName: "test",
				needCores:     0,
			},
			want: 10,
		},
		{
			name: "two",
			args: args{
				pod:           testPod2,
				containerName: "test2",
				needCores:     0,
			},
			want: 30,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRealGPU(tt.args.pod, tt.args.containerName, tt.args.needCores); got != tt.want {
				t.Errorf("GetRealGPU() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPolicyPod(t *testing.T) {
	testPod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				types.PolicyNameAnnotation: "test",
			},
		},
	}

	testPod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{},
	}

	tests := []struct {
		name string
		args *corev1.Pod
		want bool
	}{
		{
			name: "yes",
			args: testPod1,
			want: true,
		},
		{
			name: "no",
			args: testPod2,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPolicyPod(tt.args); got != tt.want {
				t.Errorf("IsPolicyPod() = %v, want %v", got, tt.want)
			}
		})
	}
}
