package cache

import corev1 "k8s.io/api/core/v1"

//Info contains infomations aboud GPU
type Info struct {
	Devices []string
	Cores   int64
	Memory  int64
}

type containerToInfo map[string]*Info

// PodCache represents a list of pod to GPU mappings.
type PodCache struct {
	PodGPUMapping map[string]containerToInfo
	UidPodMapping map[string]*corev1.Pod
}

//NewAllocateCache creates new PodCache
func NewAllocateCache() *PodCache {
	return &PodCache{
		PodGPUMapping: make(map[string]containerToInfo),
		UidPodMapping: make(map[string]*corev1.Pod),
	}
}

//Pods returns all pods in PodCache
func (pgpu *PodCache) Pods() []string {
	ret := make([]string, 0)
	for k := range pgpu.PodGPUMapping {
		ret = append(ret, k)
	}
	return ret
}

//Insert adds GPU info of pod into PodCache if not exist
func (pgpu *PodCache) Insert(podUID, contName string, pod *corev1.Pod, cache *Info) {
	if _, exists := pgpu.PodGPUMapping[podUID]; !exists {
		pgpu.PodGPUMapping[podUID] = make(containerToInfo)
	}
	pgpu.PodGPUMapping[podUID][contName] = cache

	if _, exist := pgpu.UidPodMapping[podUID]; !exist {
		pgpu.UidPodMapping[podUID] = pod
	}
}

//GetCache returns GPU of pod if exist
func (pgpu *PodCache) GetCache(podUID string) map[string]*Info {
	containers, exists := pgpu.PodGPUMapping[podUID]
	if !exists {
		return nil
	}

	return containers
}

//Delete removes GPU info in PodCache
func (pgpu *PodCache) Delete(uid string) {
	delete(pgpu.PodGPUMapping, uid)
	delete(pgpu.UidPodMapping, uid)
}
