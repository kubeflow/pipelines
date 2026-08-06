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

func TestClassifyPodFailure(t *testing.T) {
	testCases := []struct {
		name           string
		reason         string
		expectCategory PodFailureCategory
		expectReason   string
	}{
		{
			name:           "empty message",
			reason:         "",
			expectCategory: PodFailureCategoryNone,
			expectReason:   "",
		},
		{
			name:           "ordinary user pipeline error is not classified",
			reason:         "exit status 1: division by zero in user script",
			expectCategory: PodFailureCategoryNone,
			expectReason:   "",
		},
		{
			name:           "ImagePullBackOff is Provisioning",
			reason:         `Failed to pull image "bad-tag:latest": ImagePullBackOff`,
			expectCategory: PodFailureCategoryProvisioning,
			expectReason:   "ImagePullBackOff",
		},
		{
			name:           "Unschedulable is Provisioning",
			reason:         "0/1 nodes are available: 1 Insufficient cpu. Unschedulable",
			expectCategory: PodFailureCategoryProvisioning,
			expectReason:   "Unschedulable",
		},
		{
			name:           "OOMKilled is Runtime",
			reason:         "container terminated with reason OOMKilled, exit code 137",
			expectCategory: PodFailureCategoryRuntime,
			expectReason:   "OOMKilled",
		},
		{
			name:           "CrashLoopBackOff is Runtime",
			reason:         "back-off restarting failed container: CrashLoopBackOff",
			expectCategory: PodFailureCategoryRuntime,
			expectReason:   "CrashLoopBackOff",
		},
		{
			name:           "NodeLost is Node",
			reason:         "node has been marked NodeLost by the controller",
			expectCategory: PodFailureCategoryNode,
			expectReason:   "NodeLost",
		},
		{
			name:           "Preempted is Node",
			reason:         "pod was Preempted to make room for a higher priority pod",
			expectCategory: PodFailureCategoryNode,
			expectReason:   "Preempted",
		},
		{
			name:           "Evicted is Node",
			reason:         "pod Evicted due to node memory pressure",
			expectCategory: PodFailureCategoryNode,
			expectReason:   "Evicted",
		},
		{
			name:           "PreemptionByScheduler condition reason is Node",
			reason:         "PreemptionByScheduler",
			expectCategory: PodFailureCategoryNode,
			expectReason:   "PreemptionByScheduler",
		},
		{
			name:           "TerminationByKubelet condition reason is Node",
			reason:         "TerminationByKubelet",
			expectCategory: PodFailureCategoryNode,
			expectReason:   "TerminationByKubelet",
		},
		{
			name:           "first matching pattern wins when message contains multiple substrings",
			reason:         "ImagePullBackOff eventually led to OOMKilled during retry",
			expectCategory: PodFailureCategoryProvisioning,
			expectReason:   "ImagePullBackOff",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			category, reason := ClassifyPodFailure(testCase.reason)
			assert.Equal(t, testCase.expectCategory, category)
			assert.Equal(t, testCase.expectReason, reason)
		})
	}
}
