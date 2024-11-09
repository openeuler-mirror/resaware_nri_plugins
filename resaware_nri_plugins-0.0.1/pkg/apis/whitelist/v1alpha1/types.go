package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen=true
// WhitelistSpec defines the desired state of Whitelist
type WhitelistSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS -- desired state of cluster
	Name          string      `json:"name,omitempty"`
	EffectiveFlag bool        `json:"effectiveflag,omitempty"`
	TargetPods    []TargetPod `json:"targetPods,omitempty"`
}

// +k8s:deepcopy-gen=true
type TargetPod struct {
	Namespace string   `json:"namespace,omitempty"`
	Names     []string `json:"names,omitempty"`
}

// WhitelistStatus defines the observed state of Whitelist.
// It should always be reconstructable from the state of the cluster and/or outside world.
type WhitelistStatus struct {
	// INSERT ADDITIONAL STATUS FIELDS -- observed state of cluster
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Whitelist is the Schema for the whitelists API
// +k8s:openapi-gen=true
type Whitelist struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WhitelistSpec   `json:"spec,omitempty"`
	Status WhitelistStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WhitelistList contains a list of Whitelist
type WhitelistList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Whitelist `json:"items"`
}
