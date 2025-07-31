package agent

import (
	"bou.ke/monkey"
	"context"
	"fmt"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"numaadj.huawei.com/pkg/util"
	"os"
	"path/filepath"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetWatcher(t *testing.T) {
	tests := []struct {
		name     string
		k8sCli   kubernetes.Interface
		expected *watcher
	}{
		{
			name:   "test_get_watcher",
			k8sCli: fake.NewSimpleClientset(),
			expected: &watcher{
				k8sCli: fake.NewSimpleClientset(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			singletonWatcher = nil
			got := getWatcher(test.k8sCli)
			if got.k8sCli != test.k8sCli {
				t.Errorf("getWatcher() = %v, expected %v", got.k8sCli, test.k8sCli)
			}
		})
	}
}

func TestWatcher_WatchGroupConfigMap(t *testing.T) {
	type args struct {
		k8sCli kubernetes.Interface
	}
	tests := []struct {
		name    string
		w       *watcher
		args    args
		wantErr bool
	}{
		{
			name: "test_watch_group_config_map",
			w: &watcher{
				k8sCli: fake.NewSimpleClientset(),
			},
			args: args{
				k8sCli: fake.NewSimpleClientset(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试用的ConfigMap
			testCm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mpam-qos-config",
					Namespace: configMapNamespace,
				},
				Data: map[string]string{
					"rc.conf": "create",
				},
			}

			var realTimeData *ConfigData
			applyConfigPatch := monkey.Patch(ApplyConfig, func(data *ConfigData) {
				realTimeData = data
			})
			defer applyConfigPatch.Unpatch()
			// 启动watchGroupConfigMap
			tt.w.watchGroupConfigMap()
			// 等待一段时间，让watcher开始工作
			time.Sleep(2 * time.Second)

			tt.w.k8sCli.CoreV1().ConfigMaps(configMapNamespace).Create(context.TODO(), testCm, metav1.CreateOptions{})

			time.Sleep(2 * time.Second)
			if fmt.Sprintf("%v", realTimeData) != "&map[rc.conf:create]" {
				t.Fatalf("verifyData: %s should be %s", fmt.Sprintf("%v", realTimeData), "&map[rc.conf:create]")
			}
			// 修改ConfigMap，触发更新事件
			updatedCm := testCm.DeepCopy()
			updatedCm.Data["rc.conf"] = "updated"
			tt.w.k8sCli.CoreV1().ConfigMaps(configMapNamespace).Update(context.TODO(), updatedCm, metav1.UpdateOptions{})

			time.Sleep(2 * time.Second)

			if fmt.Sprintf("%v", realTimeData) != "&map[rc.conf:updated]" {
				t.Fatalf("verifyData: %s should be %s", fmt.Sprintf("%v", realTimeData), "&map[rc.conf:updated]")
			}
			// 删除ConfigMap，触发删除事件
			tt.w.k8sCli.CoreV1().ConfigMaps(configMapNamespace).Delete(context.TODO(), testCm.Name, metav1.DeleteOptions{})

			// 等待一段时间，让watcher处理删除事件
			time.Sleep(2 * time.Second)
			if realTimeData != nil {
				t.Fatalf("verifyData: %s should be nil", fmt.Sprintf("%v", realTimeData))
			}
		})
	}
}

func TestApplyConfig(t *testing.T) {
	tests := []struct {
		name   string
		data   ConfigData
		expect string
	}{
		{
			name: "test_apply_config_with_data",
			data: map[string]string{
				"config.yaml": "mpam:\n  group1:\n    L3PRI: 1\n    MBPRI: 2\n",
			},
			expect: "L3PRI:1=1;98=1;195=1;292=1\nMBPRI:0=2;1=2;2=2;3=2\n",
		},
		{
			name:   "test_apply_config_without_data",
			data:   nil,
			expect: "",
		},
	}

	patch := monkey.Patch(os.Mkdir, func(path string, perm os.FileMode) error {
		return nil
	})
	defer patch.Unpatch()

	mockFileContent := "L3PRI:1=0000003;98=0000003;195=0000003;292=0000003"
	r, w, _ := os.Pipe()
	go func() {
		w.Write([]byte(mockFileContent))
		w.Close()
	}()
	patchOsOpen := monkey.Patch(os.OpenFile, func(path string, flag int, perm os.FileMode) (file *os.File, err error) {
		return r, nil
	})
	defer patchOsOpen.Unpatch()

	verifyData := ""
	writeFilePatch := monkey.Patch(util.WriteFile, func(schemetaFile string, content string) error {
		verifyData = content
		return nil
	})
	defer writeFilePatch.Unpatch()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifyData = ""
			ApplyConfig(&tt.data)
			if verifyData != tt.expect {
				t.Fatalf("verifyData: %s should be %s", verifyData, tt.expect)
			}
		})
	}
}

func TestCleanResctrlGroup(t *testing.T) {
	tests := []struct {
		name    string
		groups  []string
		wantErr bool
	}{
		{
			name:    "test_clean_resctrl_group",
			groups:  []string{"group1", "group2"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(resCtrlRoot, "test_group")
			err := os.MkdirAll(testDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}
			defer os.RemoveAll(resCtrlRoot)

			schemataPath := filepath.Join(testDir, resCtrlSchemataFile)
			err = ioutil.WriteFile(schemataPath, []byte("test schemata"), 0644)
			if err != nil {
				t.Fatalf("Failed to create schemata file: %v", err)
			}

			cleanResCtrlGroup(tt.groups)

			entries, err := ioutil.ReadDir(resCtrlRoot)
			if err != nil {
				t.Fatalf("Failed to read directory: %v", err)
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				groupName := entry.Name()
				found := false
				for _, group := range tt.groups {
					if group == groupName {
						found = true
						break
					}
				}
				if found {
					t.Errorf("Directory %s should have been cleaned", groupName)
				}
			}
		})
	}
}