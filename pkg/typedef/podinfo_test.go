package typedef

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNewPodInfo(t *testing.T) {
	// Test case 1: RawPod is nil, should return nil
	nilPod := (*RawPod)(nil)
	nilPodInfo := NewPodInfo(nilPod)
	if nilPodInfo != nil {
		t.Errorf("Expected nil PodInfo when RawPod is nil, got %v", nilPodInfo)
	}

	// Test case 2: RawPod is not nil, should return PodInfo with correct fields
	testPod := &RawPod{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
			UID:       "test-uid",
			Labels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}
	testPodInfo := NewPodInfo(testPod)
	if testPodInfo == nil {
		t.Error("Expected PodInfo when RawPod is not nil, got nil")
	}
	if testPodInfo.Name != testPod.Name {
		t.Errorf("Expected PodInfo.Name to be %s, got %s", testPod.Name, testPodInfo.Name)
	}
	if testPodInfo.Namespace != testPod.Namespace {
		t.Errorf("Expected PodInfo.Namespace to be %s, got %s", testPod.Namespace, testPodInfo.Namespace)
	}
	if testPodInfo.UID != testPod.ID() {
		t.Errorf("Expected PodInfo.UID to be %s, got %s", testPod.ID(), testPodInfo.UID)
	}
	if testPodInfo.Labels == nil {
		t.Error("Expected PodInfo.Labels to be not nil")
	}
	for k, v := range testPod.Labels {
		if testPodInfo.Labels[k] != v {
			t.Errorf("Expected PodInfo.Labels[%s] to be %s, got %s", k, v, testPodInfo.Labels[k])
		}
	}
}

// TestPodInfo DeepCopy tests DeepCopy function of PodInfo
func TestPodInfoDeepCopy(t *testing.T) {
	// Test case 1: PodInfo is nil, should return nil
	nilPodInfo := (*PodInfo)(nil)
	nilCopy := nilPodInfo.DeepCopy()
	if nilCopy != nil {
		t.Errorf("Expected nil PodInfo copy when PodInfo is nil, got %v", nilCopy)
	}

	// Test case 2: PodInfo is not nil, should return a deep copy
	testPodInfo := &PodInfo{
		Name:      "test-pod",
		Namespace: "test-namespace",
		UID:       "test-uid",
		Labels: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}
	copyPodInfo := testPodInfo.DeepCopy()
	if copyPodInfo == nil {
		t.Error("Expected a copy of PodInfo, got nil")
	}
	if copyPodInfo == testPodInfo {
		t.Error("Expected a different instance of PodInfo, got the same instance")
	}
	if copyPodInfo.Name != testPodInfo.Name {
		t.Errorf("Expected copy.Name to be %s, got %s", testPodInfo.Name, copyPodInfo.Name)
	}
	if copyPodInfo.Namespace != testPodInfo.Namespace {
		t.Errorf("Expected copy.Namespace to be %s, got %s", testPodInfo.Namespace, copyPodInfo.Namespace)
	}
	if copyPodInfo.UID != testPodInfo.UID {
		t.Errorf("Expected copy.UID to be %s, got %s", testPodInfo.UID, copyPodInfo.UID)
	}
	if copyPodInfo.Labels == nil {
		t.Error("Expected copy.Labels to be not nil")
	}
	if &copyPodInfo.Labels == &testPodInfo.Labels {
		t.Error("Expected a different instance of Labels map, got the same instance")
	}
	for k, v := range testPodInfo.Labels {
		if copyPodInfo.Labels[k] != v {
			t.Errorf("Expected copy.Labels[%s] to be %s, got %s", k, v, copyPodInfo.Labels[k])
		}
	}
}

// TestPodInfo ResCtrlGroup tests ResCtrlGroup function of PodInfo
func TestPodInfoResCtrlGroup(t *testing.T) {
	// Test case 2: Labels is nil, should return empty string
	noLabelsPodInfo := &PodInfo{
		Name:      "test-pod",
		Namespace: "test-namespace",
		UID:       "test-uid",
		Labels:    nil,
	}
	if noLabelsPodInfo.ResCtrlGroup() != "" {
		t.Errorf("Expected empty string when Labels is nil, got %s", noLabelsPodInfo.ResCtrlGroup())
	}

	// Test case 3: Labels does not contain GroupLabel, should return empty string
	noGroupLabelPodInfo := &PodInfo{
		Name:      "test-pod",
		Namespace: "test-namespace",
		UID:       "test-uid",
		Labels: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}
	if noGroupLabelPodInfo.ResCtrlGroup() != "" {
		t.Errorf("Expected empty string when Labels does not contain GroupLabel, got %s", noGroupLabelPodInfo.ResCtrlGroup())
	}

	// Test case 4: Labels contains GroupLabel, should return the corresponding value
	groupLabelPodInfo := &PodInfo{
		Name:      "test-pod",
		Namespace: "test-namespace",
		UID:       "test-uid",
		Labels: map[string]string{
			"rcgroup": "test-group",
		},
	}
	expectedGroup := "test-group"
	if groupLabelPodInfo.ResCtrlGroup() != expectedGroup {
		t.Errorf("Expected ResCtrlGroup to be %s, got %s", expectedGroup, groupLabelPodInfo.ResCtrlGroup())
	}
}
