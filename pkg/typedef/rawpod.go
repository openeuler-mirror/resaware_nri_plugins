// Copyright (c) Huawei Technologies Co., Ltd. 2023. All rights reserved.
// rubik licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Jiaqi Yang
// Create: 2023-01-05
// Description: This file defines RawPod which encapsulate kubernetes pods

// Package typedef defines core struct and methods for rubik
package typedef

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	// RUNNING means the Pod is in the running phase
	RUNNING = corev1.PodRunning
)

type (
	// RawPod represents kubernetes pod structure
	RawPod corev1.Pod
)

// ExtractPodInfo returns podInfo from RawPod
func (pod *RawPod) ExtractPodInfo() *PodInfo {
	if pod == nil {
		return nil
	}
	return NewPodInfo(pod)
}

// ID returns the unique identity of pod
func (pod *RawPod) ID() string {
	if pod == nil {
		return ""
	}
	return string(pod.UID)
}
