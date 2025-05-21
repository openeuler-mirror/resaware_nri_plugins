package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen=true
// OenumaSpec defines the desired state of Oenuma
type OenumaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS -- desired state of cluster
	Name         string `json:"name,omitempty"`
	Replicas     int32  `json:"replicas,omitempty"`
	Node         []Node `json:"node,omitempty"`
	UpdateEnable string `json:"updateenable,omitempty"`
}

// +k8s:deepcopy-gen=true
type Node struct {
	Name        string        `json:"name,omitempty"`
	Numa        []Numa        `json:"numa,omitempty"`
	PodAffinity []PodAffinity `json:"podAffinity,omitempty"`
}

// +k8s:deepcopy-gen=true
type Numa struct {
	NumaNum int32  `json:"numaNum"`
	Cpuset  string `json:"cpuset,omitempty"`
	Memset  string `json:"memset,omitempty"`
}

// +k8s:deepcopy-gen=true
type PodAffinity struct {
	NumaNum    int32       `json:"numaNum"`
	PodName    string      `json:"podName,omitempty"`
	Namespace  string      `json:"namespace,omitempty"`
	Containers []Container `json:"containers,omitempty"`
}

// +k8s:deepcopy-gen=true
type Container struct {
	ContainerId string `json:"containerId"`
	Name        string `json:"name"`
	Cpuset      string `json:"cpuset"`
	Memset      string `json:"memset"`
}

// OenumaStatus defines the observed state of Oenuma.
// It should always be reconstructable from the state of the cluster and/or outside world.
type OenumaStatus struct {
	// INSERT ADDITIONAL STATUS FIELDS -- observed state of cluster
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Oenuma is the Schema for the oenumas API
// +k8s:openapi-gen=true
type Oenuma struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OenumaSpec   `json:"spec,omitempty"`
	Status OenumaStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OenumaList contains a list of Oenuma
type OenumaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Oenuma `json:"items"`
}
