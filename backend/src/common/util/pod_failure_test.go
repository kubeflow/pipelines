// Copyright 2026 The Kubeflow Authors
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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyPodFailure_Categories(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    FailureCategory
	}{
		{"image pull backoff detail", `Back-off pulling image "ghcr.io/example/missing:v1"`, FailureCategoryImagePull},
		{"err image pull reason", "ErrImagePull: rpc error: failed to pull image", FailureCategoryImagePull},
		{"image pull backoff reason", "ImagePullBackOff: Back-off pulling image \"repo/app:v1\"", FailureCategoryImagePull},
		{"failed to pull image", "Failed to pull image \"repo/app:v1\"", FailureCategoryImagePull},
		{"image pull case insensitive", "imagepullbackoff", FailureCategoryImagePull},

		{"unschedulable", "Unschedulable: 0/1 nodes are available: 1 Insufficient cpu.", FailureCategoryScheduling},
		{"failed scheduling", "FailedScheduling: 0/3 nodes are available: 3 Insufficient memory.", FailureCategoryScheduling},
		{"insufficient cpu", "Insufficient cpu", FailureCategoryScheduling},
		{"insufficient memory", "Insufficient memory", FailureCategoryScheduling},

		{"oom killed", "OOMKilled", FailureCategoryRuntime},
		{"crash loop", "CrashLoopBackOff: back-off 5m0s restarting failed container", FailureCategoryRuntime},
		{"bare error reason", "Error", FailureCategoryRuntime},
		{"error reason with detail", "Error: container failed to start", FailureCategoryRuntime},

		{"forbidden pods", `pods "train-abc" is forbidden: violates PodSecurity "restricted"`, FailureCategoryAdmission},
		{"forbidden reason", `Forbidden: pods "train-abc" is forbidden`, FailureCategoryAdmission},
		{"admission webhook", "admission webhook \"kyverno\" denied the request", FailureCategoryAdmission},
		{"kyverno", "Kyverno policy require-labels failed", FailureCategoryAdmission},

		{"empty", "", FailureCategoryUnknown},
		{"whitespace", "   ", FailureCategoryUnknown},
		{"user exit code 1", "Error (exit code 1)", FailureCategoryUnknown},
		{"user exit code 137", "Error (exit code 137)", FailureCategoryUnknown},
		{"unrecognized", "something unexpected happened", FailureCategoryUnknown},
		{"unbound volume stays unknown", "0/1 nodes are available: 1 pod has unbound immediate PersistentVolumeClaims.", FailureCategoryUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyPodFailure(tt.message, "Pending")
			assert.Equal(t, tt.want, got.Category)
		})
	}
}

func TestClassifyPodFailure_PhaseTier(t *testing.T) {
	const message = "ImagePullBackOff"
	tests := []struct {
		phase string
		want  FailureTier
	}{
		{"Pending", FailureTierWarning},
		{"Running", FailureTierWarning},
		{"pending", FailureTierWarning},
		{"Failed", FailureTierTerminal},
		{"Error", FailureTierTerminal},
		{"Succeeded", FailureTierNone},
		{"Skipped", FailureTierNone},
		{"Omitted", FailureTierNone},
		{"", FailureTierWarning},
		{"Unknown", FailureTierWarning},
	}
	for _, tt := range tests {
		t.Run(tt.phase, func(t *testing.T) {
			got := ClassifyPodFailure(message, tt.phase)
			assert.Equal(t, FailureCategoryImagePull, got.Category)
			assert.Equal(t, tt.want, got.Tier)
		})
	}
}

func TestClassifyPodFailure_TierFollowsPhaseForEveryCategory(t *testing.T) {
	tests := []struct {
		name    string
		message string
		phase   string
		cat     FailureCategory
		tier    FailureTier
	}{
		{"image pull pending", "ErrImagePull", "Pending", FailureCategoryImagePull, FailureTierWarning},
		{"image pull failed", "ImagePullBackOff", "Failed", FailureCategoryImagePull, FailureTierTerminal},
		{"scheduling running", "Unschedulable", "Running", FailureCategoryScheduling, FailureTierWarning},
		{"scheduling failed", "FailedScheduling", "Failed", FailureCategoryScheduling, FailureTierTerminal},
		{"runtime running", "OOMKilled", "Running", FailureCategoryRuntime, FailureTierWarning},
		{"runtime error phase", "CrashLoopBackOff", "Error", FailureCategoryRuntime, FailureTierTerminal},
		{"admission pending", `pods "x" is forbidden`, "Pending", FailureCategoryAdmission, FailureTierWarning},
		{"admission failed", "Forbidden", "Failed", FailureCategoryAdmission, FailureTierTerminal},
		{"unknown pending", "something unexpected", "Pending", FailureCategoryUnknown, FailureTierWarning},
		{"unknown failed", "something unexpected", "Failed", FailureCategoryUnknown, FailureTierTerminal},
		{"unknown succeeded", "something unexpected", "Succeeded", FailureCategoryUnknown, FailureTierNone},
		{"empty failed is none", "", "Failed", FailureCategoryUnknown, FailureTierNone},
		{"user exit stays unknown", "Error (exit code 1)", "Failed", FailureCategoryUnknown, FailureTierTerminal},
		{"succeeded image pull does not escalate", "ImagePullBackOff", "Succeeded", FailureCategoryImagePull, FailureTierNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyPodFailure(tt.message, tt.phase)
			assert.Equal(t, tt.cat, got.Category)
			assert.Equal(t, tt.tier, got.Tier)
		})
	}
}

func TestClassifyPodFailure_ImagePullNotClassifiedAsRuntime(t *testing.T) {
	got := ClassifyPodFailure("ErrImagePull: error looking up image", "Pending")
	assert.Equal(t, FailureCategoryImagePull, got.Category)
	assert.Equal(t, FailureTierWarning, got.Tier)
}
