// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License")
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/Licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific Language governing permissions and
// Limitations under the License.

package diagnostics

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
)

type DiagnosticCategory string

const (
	CategoryUnspecified DiagnosticCategory = "DIAGNOSTIC_CATEGORY_UNSPECIFIED"
	CategoryProvisioningFailure DiagnosticCategory = "PROVISIONING_FAILURE"
	CategorySchedulingFailure DiagnosticCategory = "SCHEDULING_FAILURE"
	CategoryRuntimeCrash DiagnosticCategory = "RUNTIME_CRASH"
	CategoryNodeEviction DiagnosticCategory = "NODE_EVICTION"
)

type PodLifecycleDiagnostics struct {
	Category   DiagnosticCategory  `json:"category"`
	ErrorCode   string          `json:"error_code"`
	ErrorMessage  string       `json:"error_message"`
}


const (
	DefaultDocsURL = "https://www.kubeflow.org/docs/components/pipelines/v2/troubleshooting"
)

// ClassifyPodStatus inspects a Kubernetes v1.PodStatus and returns a structured PodLifecycleDiagnostics pointer.
// If no Lifecycle failure is detected, it returns nil.

func ClassifyPodStatus(podStatus *v1.PodStatus) *PodLifecycleDiagnostics {
	if podStatus == nil {
		return nil
	}

	// 1. Check Pod Eviction / Node Preemption
	if strings.EqualFold(podStatus.Reason, "Evicted") || strings.EqualFold(podStatus.Reason, "Preempted") {
		return &PodLifecycleDiagnostics{
			Category: CategoryNodeEviction,
			ErrorCode: "NODE_EVICTED",
			ErrorMessage: fmt.Sprintf("The pod was evicted or preempted: %s", podStatus.Message),
		}
	}

	statusText := podStatus.Reason + " " + podStatus.Message

	if strings.Contains(statusText, "storageclass.storage.k8s.io") || strings.Contains(statusText, "StorageClass") && strings.Contains(statusText, "not found") {
		return &PodLifecycleDiagnostics{
			Category: CategoryProvisioningFailure,
			ErrorCode: "INVALID_STORAGE_CLASS",
			ErrorMessage: fmt.Sprintf("The pod could not be provisioned due to missing storage class: %s", podStatus.Message),
		}
	}


	if strings.Contains(statusText, "OOMKilled") {
		return &PodLifecycleDiagnostics{
			Category: CategoryRuntimeCrash,
			ErrorCode: "OOMKilled",
			ErrorMessage: fmt.Sprintf("The pod was evicted or preempted: %s", podStatus.Message),
		}
	}

	if strings.Contains(statusText, "ImagePullBackOff") || strings.Contains(statusText, "ErrImagePull") {
		return &PodLifecycleDiagnostics{
			Category: CategoryProvisioningFailure,
			ErrorCode: "IMAGE_PULL_BACKOFF",
			ErrorMessage: fmt.Sprintf("Container image could not be pulled: %s", podStatus.Message),
		}
	}

	if strings.Contains(statusText, "Unschedulable") {
		return &PodLifecycleDiagnostics{
			Category: CategorySchedulingFailure,
			ErrorCode: "UNSCHEDULABLE",
			ErrorMessage: fmt.Sprintf("The pod could not be scheduled on any node: %s", podStatus.Message),
		}
	}
	


	// 2. Check Container Level Statuses (Waiting or Terminated)
	containerStatuses := append(podStatus.InitContainerStatuses, podStatus.ContainerStatuses...)
	for _, cs := range containerStatuses {
		if cs.State.Waiting != nil {
			reason := cs.State.Waiting.Reason
			switch reason {
				case "ImagePullBackOff", "ErrImagePull":
					return &PodLifecycleDiagnostics{
						Category: CategoryProvisioningFailure,
						ErrorCode: "IMAGE_PULL_BACKOFF",
						ErrorMessage: fmt.Sprintf("Container image '%s' could not be pulled: %s", cs.Image, cs.State.Waiting.Message),
					}
				case "InvalidImageName":
					return &PodLifecycleDiagnostics{
						Category: CategoryProvisioningFailure,
						ErrorCode: "INVALID_IMAGE_NAME",
						ErrorMessage: fmt.Sprintf("Container image '%s' has an invalid name: %s", cs.Image, cs.State.Waiting.Message),
						

			}
			    case "CrashLoopBackOff":
					return &PodLifecycleDiagnostics{
						Category: CategoryRuntimeCrash,
						ErrorCode: "CRASH_LOOP_BACKOFF",
						ErrorMessage: fmt.Sprintf("Container '%s' is in a crash loop, failed repeatedly: %s", cs.Image, cs.State.Waiting.Message),
						
		}
	}


}

        if cs.State.Terminated != nil {
			term := cs.State.Terminated
			if term.Reason == "OOMKilled" || term.ExitCode == 137 {
                return &PodLifecycleDiagnostics{
					Category: CategoryRuntimeCrash,
					ErrorCode: "OOM_KILLED",
					ErrorMessage: fmt.Sprintf("Container '%s' was terminated due to out-of-memory (OOMKilled): %d", cs.Image, term.ExitCode),
					
				}
			}
			
			if term.ExitCode != 0 {
				return &PodLifecycleDiagnostics{
					Category: CategoryRuntimeCrash,
					ErrorCode: "RUNTIME_ERROR",
					ErrorMessage: fmt.Sprintf("Container '%s' failed at runtime: %d", cs.Image, term.ExitCode),
				}
			}
		}

	}


  // 3. Check Pod Level Conditions (Scheduling Failures)
  for _, condition := range podStatus.Conditions {
        if condition.Type == v1.PodScheduled && condition.Status == v1.ConditionFalse {
			if condition.Reason == v1.PodReasonUnschedulable || strings.Contains(condition.Message, "insufficient") {
				return &PodLifecycleDiagnostics{
					Category: CategorySchedulingFailure,
					ErrorCode: "POD_UNSCHEDULABLE",
					ErrorMessage: fmt.Sprintf("The pod could not be scheduled on any node: %s", condition.Message),
				}
			}
		}
  }

  return nil
}