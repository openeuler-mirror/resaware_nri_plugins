package typedef

const (
	GroupLabel = "rcgroup"
)

type PodInfo struct {
	Name      string
	UID       string
	Namespace string
	Labels    map[string]string
}
type ContainerInfo struct {
	Name string
	ID   string
}

// NewPodInfo creates the PodInfo instance
func NewPodInfo(pod *RawPod) *PodInfo {
	if pod == nil {
		return nil
	}
	return &PodInfo{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		UID:       pod.ID(),
		Labels:    pod.DeepCopy().Labels,
	}
}

// DeepCopy returns deepcopy object
func (pod *PodInfo) DeepCopy() *PodInfo {
	if pod == nil {
		return nil
	}

	var podInfoCopy = *pod
	if pod.Labels != nil {
		annoMap := make(map[string]string)
		for k, v := range pod.Labels {
			annoMap[k] = v
		}
		podInfoCopy.Labels = annoMap
	}

	return &podInfoCopy
}

// ResCtrlGroup get res ctrl group
func (pod *PodInfo) ResCtrlGroup() string {
	if pod.Labels != nil {
		if rcgroup, ok := pod.Labels[GroupLabel]; ok {
			return rcgroup
		}
	}
	return ""
}
