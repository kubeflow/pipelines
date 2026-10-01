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

// FailureCategory is an engine-neutral class for a pod lifecycle failure.
// Values match KEP-12843 so later persistence and UI code can share them.
type FailureCategory string

const (
	FailureCategoryImagePull  FailureCategory = "image-pull"
	FailureCategoryScheduling FailureCategory = "scheduling"
	FailureCategoryRuntime    FailureCategory = "runtime"
	FailureCategoryAdmission  FailureCategory = "admission"
	FailureCategoryUnknown    FailureCategory = "unknown"
)

// FailureTier is derived only from the node phase.
// A Pending image-pull and a Failed image-pull share a category and differ in tier.
type FailureTier string

const (
	// FailureTierNone means there is no lifecycle failure to surface.
	FailureTierNone FailureTier = "none"
	// FailureTierWarning means the pod is still non-terminal.
	FailureTierWarning FailureTier = "warning"
	// FailureTierTerminal means the node phase is Failed or Error.
	FailureTierTerminal FailureTier = "terminal"
)

// PodFailure is the classification of one pod status message.
type PodFailure struct {
	Category FailureCategory
	Tier     FailureTier
}

var imagePullMarkers = []string{
	"errimagepull",
	"imagepullbackoff",
	"back-off pulling image",
	"failed to pull image",
}

var schedulingMarkers = []string{
	"unschedulable",
	"failedscheduling",
	"insufficient cpu",
	"insufficient memory",
}

var admissionMarkers = []string{
	"is forbidden",
	"admission webhook",
	"podsecurity",
	"kyverno",
}

var runtimeMarkers = []string{
	"oomkilled",
	"crashloopbackoff",
}

// ClassifyPodFailure maps a pod status message and phase to a category and tier.
// Category comes from known Kubernetes failure text. Tier comes from phase.
// An empty message is unknown with tier none. "Error (exit code N)" stays unknown:
// that is a user-script failure, not an infrastructure lifecycle failure.
func ClassifyPodFailure(message, phase string) PodFailure {
	message = strings.TrimSpace(message)
	if message == "" {
		return PodFailure{Category: FailureCategoryUnknown, Tier: FailureTierNone}
	}
	return PodFailure{
		Category: categoryForMessage(message),
		Tier:     tierForPhase(phase),
	}
}

func categoryForMessage(message string) FailureCategory {
	lower := strings.ToLower(message)
	reason := strings.ToLower(lifecycleMessageReason(message))
	if matchesMarker(lower, reason, imagePullMarkers) {
		return FailureCategoryImagePull
	}
	if matchesMarker(lower, reason, admissionMarkers) || reason == "forbidden" {
		return FailureCategoryAdmission
	}
	if matchesMarker(lower, reason, schedulingMarkers) {
		return FailureCategoryScheduling
	}
	if matchesMarker(lower, reason, runtimeMarkers) {
		return FailureCategoryRuntime
	}
	// A bare Error reason is a runtime failure. "Error (exit code N)" has no
	// colon, so its reason is the whole string and does not match.
	if reason == "error" {
		return FailureCategoryRuntime
	}
	return FailureCategoryUnknown
}

func matchesMarker(message, reason string, markers []string) bool {
	for _, marker := range markers {
		if reason == marker || strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func tierForPhase(phase string) FailureTier {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "failed", "error":
		return FailureTierTerminal
	case "succeeded", "skipped", "omitted":
		return FailureTierNone
	case "pending", "running":
		return FailureTierWarning
	default:
		// An unrecognized phase is not treated as terminal.
		return FailureTierWarning
	}
}
