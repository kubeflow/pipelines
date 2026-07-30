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
		source         PodFailureSignalSource
		expectCategory PodFailureCategory
		expectReason   string
	}{
		{
			name:           "empty message",
			reason:         "",
			source:         PodFailureSignalSourcePodStatus,
			expectCategory: PodFailureCategoryNone,
			expectReason:   "",
		},
		{
			name:           "ordinary user pipeline error is not classified",
			reason:         "exit status 1: division by zero in user script",
			source:         PodFailureSignalSourcePodStatus,
			expectCategory: PodFailureCategoryNone,
			expectReason:   "",
		},
		{
			name:           "ImagePullBackOff is Provisioning",
			reason:         `Failed to pull image "bad-tag:latest": ImagePullBackOff`,
			source:         PodFailureSignalSourcePodStatus,
			expectCategory: PodFailureCategoryProvisioning,
			expectReason:   "ImagePullBackOff",
		},
		{
			name:           "Unschedulable is Provisioning",
			reason:         "0/1 nodes are available: 1 Insufficient cpu. Unschedulable",
			source:         PodFailureSignalSourcePodStatus,
			expectCategory: PodFailureCategoryProvisioning,
			expectReason:   "Unschedulable",
		},
		{
			name:           "OOMKilled is Runtime",
			reason:         "container terminated with reason OOMKilled, exit code 137",
			source:         PodFailureSignalSourcePodStatus,
			expectCategory: PodFailureCategoryRuntime,
			expectReason:   "OOMKilled",
		},
		{
			name:           "CrashLoopBackOff is Runtime",
			reason:         "back-off restarting failed container: CrashLoopBackOff",
			source:         PodFailureSignalSourcePodStatus,
			expectCategory: PodFailureCategoryRuntime,
			expectReason:   "CrashLoopBackOff",
		},
		{
			name:           "NodeLost is Node",
			reason:         "node has been marked NodeLost by the controller",
			source:         PodFailureSignalSourcePodStatus,
			expectCategory: PodFailureCategoryNode,
			expectReason:   "NodeLost",
		},
		{
			name:           "Preempted is Node",
			reason:         "pod was Preempted to make room for a higher priority pod",
			source:         PodFailureSignalSourcePodStatus,
			expectCategory: PodFailureCategoryNode,
			expectReason:   "Preempted",
		},
		{
			name:           "Evicted is Node",
			reason:         "pod Evicted due to node memory pressure",
			source:         PodFailureSignalSourcePodStatus,
			expectCategory: PodFailureCategoryNode,
			expectReason:   "Evicted",
		},
		{
			name:           "first matching pattern wins when message contains multiple substrings",
			reason:         "ImagePullBackOff eventually led to OOMKilled during retry",
			source:         PodFailureSignalSourcePodStatus,
			expectCategory: PodFailureCategoryProvisioning,
			expectReason:   "ImagePullBackOff",
		},
		{
			// Documents the current, known gap: nothing watches pod Events yet,
			// so a FailedScheduling reason doesn't match any pattern even when
			// explicitly sourced from a PodEvent. See #12843 and #13401.
			name:           "FailedScheduling from a pod event is not yet classified",
			reason:         "0/3 nodes are available: 3 Insufficient cpu.",
			source:         PodFailureSignalSourcePodEvent,
			expectCategory: PodFailureCategoryNone,
			expectReason:   "",
		},
		{
			// Source doesn't change matching today, only Reason does; a
			// PodEvent-sourced reason that happens to contain a known
			// substring still classifies the same way a PodStatus-sourced
			// one would.
			name:           "matching is independent of source",
			reason:         "CrashLoopBackOff",
			source:         PodFailureSignalSourcePodEvent,
			expectCategory: PodFailureCategoryRuntime,
			expectReason:   "CrashLoopBackOff",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			category, reason := ClassifyPodFailure(PodFailureSignal{
				Reason: testCase.reason,
				Source: testCase.source,
			})
			assert.Equal(t, testCase.expectCategory, category)
			assert.Equal(t, testCase.expectReason, reason)
		})
	}
}
