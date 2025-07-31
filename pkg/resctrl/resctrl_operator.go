package resctrl

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"k8s.io/klog/v2"
	"numaadj.huawei.com/pkg/typedef"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	ResCtrlSchemataFileName = "schemata"
	CgroupCpusetRootPath    = "/sys/fs/cgroup/cpuset/"
	TasksFileName           = "tasks"
)


var (
	wg sync.WaitGroup
	ResCtrlRootPath 		= os.Getenv("RESCTRL_PATH")
)

func init() {
	if ResCtrlRootPath == "" {
		ResCtrlRootPath = "/sys/fs/resctrl"
	}
}


func SyncResCtrlGroupTasks() {
	pods := typedef.PodCacheInstance().ListPod()
	startTime := time.Now()
	defer klog.Infof("Sync resctrl group tasks for %d pods took %v", len(pods), time.Since(startTime))
	wg.Add(len(pods))
	for _, p := range pods {
		go func(p *typedef.PodInfo) {
			defer wg.Done()
			AssignControlGroup(p.UID, p.ResCtrlGroup())
		}(p)
	}
	wg.Wait()
}

// ReadPids reads pids from a cgroup's tasks file
func ReadPids(tasksFile string) ([]string, error) {
	var pids []string

	f, err := os.OpenFile(tasksFile, os.O_RDONLY, 0644)
	if err != nil {
		klog.Errorf("Failed to open %q: %v", tasksFile, err)
		return nil, fmt.Errorf("failed to open %q: %v", tasksFile, err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		pids = append(pids, s.Text())
	}
	if s.Err() != nil {
		klog.Errorf("Failed to read %q: %v", tasksFile, err)
		return nil, fmt.Errorf("failed to read %q: %v", tasksFile, err)
	}

	return pids, nil
}

// WritePids writes pids to a restctrl tasks file
func WritePids(tasksFile string, pids []string) error {
	f, err := os.OpenFile(tasksFile, os.O_WRONLY, 0644)
	if err != nil {
		klog.Errorf("Failed to write pids to %q: %v", tasksFile, err)
		return err
	}
	defer f.Close()

	for _, pid := range pids {
		if _, err := f.Write([]byte(pid)); err != nil {
			if !errors.Is(err, syscall.ESRCH) {
				klog.Errorf("Failed to write pid %s to %q: %v", pid, tasksFile, err)
				return err
			}
		}
	}
	return nil
}

func assignMPAMControlGroup(dir, rcgroup string) {
	if fis, err := ioutil.ReadDir(dir); err == nil {
		path := filepath.Join(dir, TasksFileName)
		if _, err := os.Lstat(path); err == nil || os.IsExist(err) {
			if pids, err := ReadPids(path); err == nil {
				WritePids(filepath.Join(ResCtrlRootPath, rcgroup, TasksFileName), pids)
			}
		}

		for _, fi := range fis {
			if fi.IsDir() {
				path := filepath.Join(dir, fi.Name())
				assignMPAMControlGroup(path, rcgroup)
			}
		}
	}
}

func findPodAndAssign(dir, uid, rcgroup string) {
	if fis, err := ioutil.ReadDir(dir); err == nil {
		for _, fi := range fis {
			if fi.IsDir() {
				path := filepath.Join(dir, fi.Name())

				if strings.Contains(fi.Name(), uid) {
					assignMPAMControlGroup(path, rcgroup)
					continue
				}

				findPodAndAssign(path, uid, rcgroup)
			}
		}
	}
}

func AssignControlGroup(uid, rcgroup string) {
	id := strings.Replace(uid, "-", "_", -1)
	findPodAndAssign(CgroupCpusetRootPath, id, rcgroup)

	findPodAndAssign(CgroupCpusetRootPath, uid, rcgroup)
}
