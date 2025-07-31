package resctrl

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"k8s.io/klog/v2"
	"numaadj.huawei.com/pkg/typedef"
)

func TestSyncResCtrlGroupTasks(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		mockPodCache := typedef.PodCacheInstance()
		mockPod1 := &typedef.PodInfo{
			Name:      "pod1",
			UID:       "uid1",
			Namespace: "default",
			Labels: map[string]string{
				"rcgroup": "LS",
			},
		}
		mockPod2 := &typedef.PodInfo{
			Name:      "pod2",
			UID:       "uid2",
			Namespace: "default",
			Labels: map[string]string{
				"rcgroup": "BE",
			},
		}
		monkey.PatchInstanceMethod(reflect.TypeOf(mockPodCache), "ListPod", func(c *typedef.PodCache) map[string]*typedef.PodInfo {
			return map[string]*typedef.PodInfo{
				"uid":  mockPod1,
				"uid2": mockPod2}
		})

		mockAssignControlGroup := func(uid, rcgroup string) {}
		monkey.Patch(AssignControlGroup, mockAssignControlGroup)

		mockKlogInfof := func(format string, args ...interface{}) {}
		monkey.Patch(klog.Infof, mockKlogInfof)

		SyncResCtrlGroupTasks()

		monkey.UnpatchAll()
	})

	t.Run("empty pod list", func(t *testing.T) {
		mockPodCache := typedef.PodCacheInstance()
		monkey.PatchInstanceMethod(reflect.TypeOf(mockPodCache), "ListPod", func(c *typedef.PodCache) map[string]*typedef.PodInfo {
			return map[string]*typedef.PodInfo{}
		})

		mockAssignControlGroup := func(uid, rcgroup string) {}
		monkey.Patch(AssignControlGroup, mockAssignControlGroup)

		mockKlogInfof := func(format string, args ...interface{}) {}
		monkey.Patch(klog.Infof, mockKlogInfof)

		SyncResCtrlGroupTasks()

		monkey.UnpatchAll()
	})
}

func TestReadPids(t *testing.T) {
	t.Run("normal", func(t *testing.T) {

		mockFileContent := "123\n456\n"
		r, w, _ := os.Pipe()
		go func() {
			w.Write([]byte(mockFileContent))
			w.Close()
		}()
		patchOsOpen := monkey.Patch(os.OpenFile, func(path string, flag int, perm os.FileMode) (file *os.File, err error) {
			return r, nil
		})
		defer patchOsOpen.Unpatch()

		mockKlogErrorf := func(format string, args ...interface{}) {}
		monkey.Patch(klog.Errorf, mockKlogErrorf)

		pids, err := ReadPids("test")
		if err != nil {
			t.Fatalf("ReadPids failed: %v", err)
		}
		if len(pids) != 2 || pids[0] != "123" || pids[1] != "456" {
			t.Fatalf("ReadPids result unexpected: %v", pids)
		}

		monkey.UnpatchAll()
	})

	t.Run("open file failed", func(t *testing.T) {
		mockOsOpenFile := func(name string, flag int, perm os.FileMode) (*os.File, error) {
			return nil, fmt.Errorf("mock error")
		}
		monkey.Patch(os.OpenFile, mockOsOpenFile)

		mockKlogErrorf := func(format string, args ...interface{}) {}
		monkey.Patch(klog.Errorf, mockKlogErrorf)

		_, err := ReadPids("test")
		if err == nil {
			t.Fatal("ReadPids should return error")
		}

		monkey.UnpatchAll()
	})
}
