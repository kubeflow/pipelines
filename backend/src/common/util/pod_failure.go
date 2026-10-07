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

// FailureCategory classifies the root cause of a pod lifecycle failure.
type FailureCategory string

const (
	// FailureCategoryImagePull covers ErrImagePull, ImagePullBackOff, "Back-off pulling image", and "Failed to pull image".
	FailureCategoryImagePull FailureCategory = "image-pull"
	// FailureCategoryScheduling covers Unschedulable, FailedScheduling, "Insufficient cpu", and "Insufficient memory".
	FailureCategoryScheduling FailureCategory = "scheduling"
	// FailureCategoryRuntime covers OOMKilled, CrashLoopBackOff, and a bare Error reason.
	FailureCategoryRuntime FailureCategory = "runtime"
	// FailureCategoryAdmission covers `pods "..." is forbidden`, Forbidden, admission webhook, PodSecurity, and Kyverno.
	FailureCategoryAdmission FailureCategory = "admission"
	// FailureCategoryUnknown is any message that does not match a known category.
	FailureCategoryUnknown FailureCategory = "unknown"
)

// FailureTier is derived only from the node phase. A Pending image-pull and a Failed image-pull share a category and differ in tier.
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

// imagePullReasonCodes map to FailureCategoryImagePull.
// Matched against the reason only, so a pod or webhook name cannot trigger them.
var imagePullReasonCodes = []string{
	"errimagepull",
	"imagepullbackoff",
}

// imagePullPhrases map to FailureCategoryImagePull.
var imagePullPhrases = []string{
	"back-off pulling image",
	"failed to pull image",
}

// admissionPhrases map to FailureCategoryAdmission.
var admissionPhrases = []string{
	"is forbidden",
	"admission webhook",
}

// admissionReasonTokens map to FailureCategoryAdmission.
// Matched against the reason only, so a pod name in the detail cannot trigger them.
var admissionReasonTokens = []string{
	"kyverno",
	"podsecurity",
}

// schedulingMarkers map to FailureCategoryScheduling.
var schedulingMarkers = []string{
	"unschedulable",
	"failedscheduling",
	"insufficient cpu",
	"insufficient memory",
}

// runtimeReasonCodes map to FailureCategoryRuntime.
var runtimeReasonCodes = []string{
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
		Category: getFailureCategoryForMessage(message),
		Tier:     getFailureTierForPhase(phase),
	}
}

func getFailureCategoryForMessage(message string) FailureCategory {
	lower := strings.ToLower(message)
	reason := strings.ToLower(lifecycleMessageReason(message))

	// image-pull is checked before runtime: ErrImagePull messages contain "error".
	if hasExactReason(reason, imagePullReasonCodes) || containsAny(lower, imagePullPhrases) {
		return FailureCategoryImagePull
	}
	if containsAny(lower, admissionPhrases) || reason == "forbidden" || containsAny(reason, admissionReasonTokens) {
		return FailureCategoryAdmission
	}
	if containsAny(lower, schedulingMarkers) {
		return FailureCategoryScheduling
	}
	// A bare Error reason is a runtime failure. "Error (exit code N)" has no
	// colon, so its reason is the whole string and does not match.
	if hasExactReason(reason, runtimeReasonCodes) || reason == "error" {
		return FailureCategoryRuntime
	}
	return FailureCategoryUnknown
}

func hasExactReason(reason string, codes []string) bool {
	for _, code := range codes {
		if reason == code {
			return true
		}
	}
	return false
}

func containsAny(s string, parts []string) bool {
	for _, part := range parts {
		if strings.Contains(s, part) {
			return true
		}
	}
	return false
}

func getFailureTierForPhase(phase string) FailureTier {
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
