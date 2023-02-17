package display

type containerToCgroup map[string]string

// podGPUs represents a list of pod to GPU mappings.
type podGPUs struct {
	podGPUMapping map[string]containerToCgroup
}

func newPodGPUs() *podGPUs {
	return &podGPUs{
		podGPUMapping: make(map[string]containerToCgroup),
	}
}

func (pgpu *podGPUs) pods() []string {
	ret := make([]string, 0)
	for k := range pgpu.podGPUMapping {
		ret = append(ret, k)
	}
	return ret
}

func (pgpu *podGPUs) insert(podUID, contName string, cgroup string) {
	if _, exists := pgpu.podGPUMapping[podUID]; !exists {
		pgpu.podGPUMapping[podUID] = make(containerToCgroup)
	}
	pgpu.podGPUMapping[podUID][contName] = cgroup
}

func (pgpu *podGPUs) getCgroup(podUID, contName string) string {
	containers, exists := pgpu.podGPUMapping[podUID]
	if !exists {
		return ""
	}
	cgroup, exists := containers[contName]
	if !exists {
		return ""
	}
	return cgroup
}

func (pgpu *podGPUs) delete(uid string) []string {
	var cgroups []string

	for _, cont := range pgpu.podGPUMapping[uid] {
		cgroups = append(cgroups, cont)
	}

	delete(pgpu.podGPUMapping, uid)

	return cgroups
}
