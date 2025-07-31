/*
Copyright (c) Huawei Technologies Co., Ltd. 2023-2023. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resctrl

import (
	"bufio"
	"fmt"
	"k8s.io/klog/v2"
	"numaadj.huawei.com/pkg/util"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var l3idCache []int

type ResCtrlGroup interface {
	SetGroupQos() error
}

type MpamResCtrlGroup struct {
	dir   string
	l3Pri int
	mbPri int
}

func NewMpamResCtrlGroup(groupName string, l3Pri, mbPri int) *MpamResCtrlGroup {
	return &MpamResCtrlGroup{
		l3Pri: l3Pri,
		mbPri: mbPri,
		dir:   filepath.Join(ResCtrlRootPath, groupName),
	}
}

func (cl *MpamResCtrlGroup) SetGroupQos() error {
	klog.Infof("starting to writeResctrlSchemata")
	if err := cl.createGroupDir(); err != nil {
		return err
	}
	schemetaFile := filepath.Join(cl.dir, ResCtrlSchemataFileName)
	l3Ids, err := cl.GetL3Id()
	if err != nil {
		klog.Errorf("Failed to get L3ID: %v", err)
		return err
	}
	var content string
	var l3List, mbList []string
	for i, l3Id := range l3Ids {
		mbList = append(mbList, fmt.Sprintf("%d=%d", i, cl.mbPri))
		l3List = append(l3List, fmt.Sprintf("%d=%d", l3Id, cl.l3Pri))
	}
	l3 := fmt.Sprintf("L3PRI:%s\n", strings.Join(l3List, ";"))
	mb := fmt.Sprintf("MBPRI:%s\n", strings.Join(mbList, ";"))
	klog.Infof("schemetaFile: %s, l3:%s, mb:%s", schemetaFile, l3, mb)
	content = l3 + mb
	if err := util.WriteFile(schemetaFile, content); err != nil {
		return fmt.Errorf("failed to write %s to file %s: %v", content, schemetaFile, err)
	}

	return nil
}

func (cl *MpamResCtrlGroup) GetL3Id() ([]int, error) {
	if l3idCache == nil || len(l3idCache) == 0 {
		l3IdFromFile, err := cl.GetL3IdFromFile()
		if err != nil {
			return nil, err
		}
		l3idCache = l3IdFromFile
	}
	return l3idCache, nil
}

func (cl *MpamResCtrlGroup) GetL3IdFromFile() ([]int, error) {
	schemetaFilePath := filepath.Join(cl.dir, ResCtrlSchemataFileName)
	file, err := os.Open(schemetaFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var l3priLine string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "L3PRI:") {
			l3priLine = line
			break
		}
	}

	if l3priLine == "" {
		return nil, fmt.Errorf("L3PRI line not found")
	}

	values := strings.SplitN(l3priLine, ":", 2)
	if len(values) < 2 {
		return nil, fmt.Errorf("invalid L3PRI format")
	}

	parts := strings.Split(values[1], ";")
	var ids []int

	for _, part := range parts {
		if part == "" {
			continue
		}
		kv := strings.Split(part, "=")
		if len(kv) < 2 {
			continue
		}

		id, err := strconv.Atoi(kv[0])
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (cl *MpamResCtrlGroup) createGroupDir() error {
	if err := os.Mkdir(cl.dir, 0700); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create cache limit directory: %v", err)
	}
	return nil
}
