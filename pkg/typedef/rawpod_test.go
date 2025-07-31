// Package typedef defines core struct and methods for rubik
package typedef

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestExtractPodInfo(t *testing.T) {
	// Test case 1: RawPod is nil, should return nil
	nilPod := (*RawPod)(nil)
	nilPodInfo := nilPod.ExtractPodInfo()
	if nilPodInfo != nil {
		t.Errorf("Expected nil PodInfo when RawPod is nil, got %v", nilPodInfo)
	}

	// Test case 2: RawPod is not nil, should return PodInfo
	testPod := &RawPod{
		ObjectMeta: v1.ObjectMeta{
			UID: "test-uid",
		},
	}
	testPodInfo := testPod.ExtractPodInfo()
	if testPodInfo == nil {
		t.Errorf("Expected PodInfo when RawPod is not nil, got nil")
	}
}

// TestID tests ID function
func TestID(t *testing.T) {
	// Test case 1: RawPod is nil, should return empty string
	nilPod := (*RawPod)(nil)
	if nilPod.ID() != "" {
		t.Errorf("Expected empty string when RawPod is nil, got %s", nilPod.ID())
	}

	// Test case 2: RawPod is not nil, should return UID string
	testPod := &RawPod{
		ObjectMeta: v1.ObjectMeta{
			UID: "test-uid",
		},
	}
	if testPod.ID() != "test-uid" {
		t.Errorf("Expected UID string 'test-uid', got %s", testPod.ID())
	}
}
