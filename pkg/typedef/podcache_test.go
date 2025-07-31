package typedef

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	PodCacheInstance()
}

func TestPodCache_DelPod(t *testing.T) {
	podCache.DelPod("non-existent-pod")

	pod := &PodInfo{
		UID:  "test-pod-uid",
		Name: "test-pod-name",
	}
	podCache.pods[pod.UID] = pod
	podCache.DelPod(pod.UID)
	if podCache.podExist(pod.UID) {
		t.Errorf("DelPod failed to delete pod: %v", pod.UID)
	}
}

func TestPodCache_UpdatePod(t *testing.T) {

	podCache.UpdatePod(nil)
	if len(podCache.pods) > 0 {
		t.Error("UpdatePod should not add nil Pod")
	}

	noUIDPod := &PodInfo{
		Name: "no-uid-pod",
	}
	podCache.UpdatePod(noUIDPod)
	if len(podCache.pods) > 0 {
		t.Error("UpdatePod should not add Pod without UID")
	}

	pod := &PodInfo{
		UID:  "test-pod-uid",
		Name: "test-pod-name",
	}
	podCache.UpdatePod(pod)
	if !podCache.podExist(pod.UID) {
		t.Errorf("UpdatePod failed to add pod: %v", pod.UID)
	}

	updatedPod := &PodInfo{
		UID:  pod.UID,
		Name: "updated-pod-name",
	}

	podCache.UpdatePod(updatedPod)

	listPod := podCache.ListPod()
	newPod := listPod["test-pod-uid"]
	if newPod.Name != updatedPod.Name {
		t.Errorf("UpdatePod failed to update pod: %v", newPod.UID)
	}
}

func TestPodCache_ListPod(t *testing.T) {
	podCache = &PodCache{
		pods: make(map[string]*PodInfo),
	}

	pod1 := &PodInfo{
		UID:  "pod1-uid",
		Name: "pod1",
	}
	pod2 := &PodInfo{
		UID:  "pod2-uid",
		Name: "pod2",
	}
	podCache.UpdatePod(pod1)
	podCache.UpdatePod(pod2)

	listedPods := podCache.ListPod()
	assert.Equal(t, 2, len(listedPods), "ListPod returned unexpected number of Pods")
}
