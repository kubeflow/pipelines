// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	k8score "k8s.io/api/core/v1"
)

// PatchPVCSpec applies fields from `patch` into `base` where patch has non-zero values.
// It mutates and returns the `base` spec.
func PatchPVCSpec(base, patch *k8score.PersistentVolumeClaimSpec) *k8score.PersistentVolumeClaimSpec {
	if base == nil || patch == nil {
		return base
	}
	if len(patch.AccessModes) > 0 {
		base.AccessModes = patch.AccessModes
	}
	if patch.Selector != nil {
		base.Selector = patch.Selector.DeepCopy()
	}
	if patch.Resources.Requests != nil {
		if base.Resources.Requests == nil {
			base.Resources.Requests = make(k8score.ResourceList)
		}
		for k, v := range patch.Resources.Requests {
			base.Resources.Requests[k] = v
		}
	}
	if patch.VolumeName != "" {
		base.VolumeName = patch.VolumeName
	}
	if patch.StorageClassName != nil {
		base.StorageClassName = patch.StorageClassName
	}
	if patch.VolumeMode != nil {
		base.VolumeMode = patch.VolumeMode
	}
	if patch.DataSource != nil {
		base.DataSource = patch.DataSource.DeepCopy()
	}
	if patch.DataSourceRef != nil {
		base.DataSourceRef = patch.DataSourceRef.DeepCopy()
	}
	return base
}
