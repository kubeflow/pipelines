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

import "strings"

// PodFailureCategory classifies a pod lifecycle failure into one of the
// buckets described in https://github.com/kubeflow/pipelines/issues/12843,
// so that timeout enforcement and UI classification share a single source
// of truth instead of maintaining independent, potentially drifting logic.
type PodFailureCategory string

const (
	// PodFailureCategoryProvisioning covers failures where the pod cannot
	// be scheduled or its container image cannot be pulled.
	PodFailureCategoryProvisioning PodFailureCategory = "Provisioning"
	// PodFailureCategoryRuntime covers failures where the pod starts but
	// fails during execution.
	PodFailureCategoryRuntime PodFailureCategory = "Runtime"
	// PodFailureCategoryNode covers failures caused by the underlying node.
	PodFailureCategoryNode PodFailureCategory = "Node"
	// PodFailureCategoryNone means the message did not match any known pod
	// lifecycle failure pattern (e.g. an ordinary user pipeline-code error).
	PodFailureCategoryNone PodFailureCategory = ""
)

type podFailurePattern struct {
	substring string
	category  PodFailureCategory
}

// podFailurePatterns is ordered; the first matching substring wins.
var podFailurePatterns = []podFailurePattern{
	{"ImagePullBackOff", PodFailureCategoryProvisioning},
	{"ErrImagePull", PodFailureCategoryProvisioning},
	{"ErrImageNeverPull", PodFailureCategoryProvisioning},
	{"InvalidImageName", PodFailureCategoryProvisioning},
	{"Unschedulable", PodFailureCategoryProvisioning},
	{"CrashLoopBackOff", PodFailureCategoryRuntime},
	{"OOMKilled", PodFailureCategoryRuntime},
	{"DeadlineExceeded", PodFailureCategoryRuntime},
	{"ContainerCannotRun", PodFailureCategoryRuntime},
	{"CreateContainerConfigError", PodFailureCategoryRuntime},
	{"CreateContainerError", PodFailureCategoryRuntime},
	{"RunContainerError", PodFailureCategoryRuntime},
	{"NodeLost", PodFailureCategoryNode},
	{"Preempted", PodFailureCategoryNode},
	{"Evicted", PodFailureCategoryNode},
}

// ClassifyPodFailure inspects a raw Kubernetes/Argo failure reason or status
// message and classifies it into a PodFailureCategory, along with the
// specific substring that matched. It returns PodFailureCategoryNone and an
// empty reason if the message does not match any known pattern.
//
// All three categories, including Unschedulable (via the pod's PodScheduled
// condition) and Node-caused failures like preemption and eviction (via the
// pod's DisruptionTarget condition), are readable from the pod object
// itself, so callers do not need a separate Kubernetes Event watch to
// populate this.
func ClassifyPodFailure(reason string) (category PodFailureCategory, matchedReason string) {
	if reason == "" {
		return PodFailureCategoryNone, ""
	}
	for _, pattern := range podFailurePatterns {
		if strings.Contains(reason, pattern.substring) {
			return pattern.category, pattern.substring
		}
	}
	return PodFailureCategoryNone, ""
}
