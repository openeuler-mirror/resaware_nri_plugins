package typedef

import (
	"sync"
)

type PodCacheInf interface {
	DelPod(podID string)
	UpdatePod(pod *PodInfo)
	ListPod() map[string]*PodInfo
}

// PodCache is used to store PodInfo
type PodCache struct {
	lock sync.RWMutex
	pods map[string]*PodInfo
}

var podCache *PodCache
var once sync.Once

func PodCacheInstance() *PodCache {
	once.Do(func() {
		podCache = &PodCache{
			pods: make(map[string]*PodInfo),
		}
	})
	return podCache
}

// podExist returns true if there is a pod whose key is podID in the pods
func (c *PodCache) podExist(podID string) bool {
	c.lock.RLock()
	_, ok := c.pods[podID]
	c.lock.RUnlock()
	return ok
}

// DelPod deletes pod information
func (c *PodCache) DelPod(podID string) {
	if ok := c.podExist(podID); !ok {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.pods, podID)
}

// UpdatePod updates pod information
func (c *PodCache) UpdatePod(pod *PodInfo) {
	if pod == nil || pod.UID == "" {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.pods[pod.UID] = pod
}

// ListPod returns the deepcopy object of all pod
func (c *PodCache) ListPod() map[string]*PodInfo {
	res := make(map[string]*PodInfo, len(c.pods))
	c.lock.RLock()
	for id, pi := range c.pods {
		res[id] = pi.DeepCopy()
	}
	c.lock.RUnlock()
	return res
}
