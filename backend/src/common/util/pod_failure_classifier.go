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

// PodFailureSignalSource identifies where a PodFailureSignal's Reason came
// from.
//
// ImagePullBackOff, CrashLoopBackOff, OOMKilled and the other reasons below
// are all readable from the pod's own status (containerStatuses[].state).
// Unschedulable is not: there is no container yet, so Argo leaves the pod
// Pending with no message, and the only signal is a FailedScheduling Event
// recorded against the pod. Carrying Source alongside Reason means a future
// caller that watches pod Events can classify through this same function
// without another signature change (see #12843).
type PodFailureSignalSource string

const (
	// PodFailureSignalSourcePodStatus means Reason was read from the pod's
	// own status (e.g. a container waiting/terminated reason).
	PodFailureSignalSourcePodStatus PodFailureSignalSource = "PodStatus"
	// PodFailureSignalSourcePodEvent means Reason was read from a
	// Kubernetes Event recorded against the pod (e.g. FailedScheduling).
	PodFailureSignalSourcePodEvent PodFailureSignalSource = "PodEvent"
)

// PodFailureSignal is the input to ClassifyPodFailure: a raw failure reason
// or status message, and where it was read from.
type PodFailureSignal struct {
	Reason string
	Source PodFailureSignalSource
}

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

// ClassifyPodFailure inspects a pod lifecycle failure signal and classifies
// it into a PodFailureCategory, along with the specific substring that
// matched. It returns PodFailureCategoryNone and an empty reason if the
// signal does not match any known pattern.
//
// Matching is currently substring-based against signal.Reason regardless of
// signal.Source; no pattern here is sourced from a pod Event yet, since
// nothing in this codebase watches pod Events today. Source is carried so
// that whoever adds that watch (see #12843) can pass a PodEvent-sourced
// signal, such as a FailedScheduling reason, through this same function.
func ClassifyPodFailure(signal PodFailureSignal) (category PodFailureCategory, matchedReason string) {
	if signal.Reason == "" {
		return PodFailureCategoryNone, ""
	}
	for _, pattern := range podFailurePatterns {
		if strings.Contains(signal.Reason, pattern.substring) {
			return pattern.category, pattern.substring
		}
	}
	return PodFailureCategoryNone, ""
}
