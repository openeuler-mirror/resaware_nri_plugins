package resctrl

import (
	"numaadj.huawei.com/pkg/util"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"bou.ke/monkey"
)

func TestMpamResCtrlGroup_SetGroupQos(t *testing.T) {
	testDir := filepath.Join(ResCtrlRootPath, "testGroup")
	defer os.RemoveAll(testDir)

	patch := monkey.Patch(util.WriteFile, func(schemetaFile string, content string) error {
		return nil
	})
	defer patch.Unpatch()

	group := NewMpamResCtrlGroup("testGroup", 1, 1)
	monkey.PatchInstanceMethod(reflect.TypeOf(group), "GetL3Id", func(cl *MpamResCtrlGroup) ([]int, error) {
		return []int{1, 2, 3}, nil
	})

	monkey.Patch(util.WriteFile, func(schemetaFile string, content string) error {
		return nil
	})

	monkey.Patch(os.Mkdir, func(name string, perm os.FileMode) error {
		return nil
	})
	defer monkey.UnpatchAll()
	err := group.SetGroupQos()

	if err != nil {
		t.Errorf("SetGroupQos() error = %v, want nil", err)
	}
}

func TestMpamResCtrlGroup_getL3Id(t *testing.T) {
	testDir := filepath.Join(ResCtrlRootPath, "testGroup")
	group := &MpamResCtrlGroup{
		dir: testDir,
	}

	schemetaFilePath := filepath.Join(testDir, ResCtrlSchemataFileName)
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Errorf("MkdirAll() error = %v, want nil", err)
		return
	}
	err = os.WriteFile(schemetaFilePath, []byte("L3PRI:1=0000003;98=0000003;195=0000003;292=0000003"), 0600)
	if err != nil {
		t.Errorf("WriteFile() error = %v, want nil", err)
		return
	}
	defer os.RemoveAll(testDir)

	l3Ids, err := group.GetL3Id()
	if err != nil {
		t.Errorf("GetL3Id() error = %v, want nil", err)
		return
	}
	if !equalIntSlices(l3Ids, []int{1, 98, 195, 292}) {
		t.Errorf("GetL3Id() = %v, want [1, 98, 195, 292]", l3Ids)
	}
}

func TestMpamResCtrlGroup_createGroupDir(t *testing.T) {
	testDir := filepath.Join(ResCtrlRootPath, "testGroup")
	group := &MpamResCtrlGroup{
		dir: testDir,
	}

	patch := monkey.Patch(os.Mkdir, func(path string, perm os.FileMode) error {
		return nil
	})
	defer patch.Unpatch()

	err := group.createGroupDir()
	if err != nil {
		t.Errorf("createGroupDir() error = %v, want nil", err)
	}

	os.MkdirAll(testDir, 0700)
	err = group.createGroupDir()
	if err != nil {
		t.Errorf("createGroupDir() error = %v, want nil", err)
	}
	os.RemoveAll(testDir)
}

func equalIntSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
