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
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
)

func TestClassifyPodStatus_NilStatus_ReturnNil(t *testing.T) {
	diag := ClassifyPodStatus(nil)
	assert.Nil(t, diag)
}

func TestClassifyPodStatus_TableDriven(t *testing.T){
	tests := []struct {
		name  string
		podStatus *v1.PodStatus
		expectedCategory DiagnosticCategory
		expectedReason string
		expectedCode  string
		expectedExitCode int32
	}{
		{
			name: "ImagePullBackOff failure",
			podStatus: &v1.PodStatus{
				ContainerStatuses: []v1.ContainerStatus{
					{
					Image: "gcr.io/invalid/missing-image:latest",
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "ImagePullBackOff",
							Message: fmt.Sprintf("Back-off pulling image 'gcr.io/invalid/missing-image:latest'"),
							
						},
					},
				},
			},
		},
		expectedCategory: CategoryProvisioningFailure,
		expectedReason: "ImagePullBackOff",
		expectedCode: "IMAGE_PULL_BACKOFF",
		expectedExitCode: -1,
	},
	{
		name: "OOMKilled runtime crash",
		podStatus: &v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					Image: "python:3.11-slim",
					State: v1.ContainerState{
						Terminated: &v1.ContainerStateTerminated{
							Reason: "OOMKilled",
							ExitCode: 137,
							Message: fmt.Sprintf("Container limit 8Gi exceeded"),
						},
					},
				},
			},
		},
		expectedCategory: CategoryRuntimeCrash,
		expectedReason: "OOMKilled",
		expectedCode: "OOM_KILLED",
		expectedExitCode: 137,
	},

	{
		name: "Unscedulable cluster resource quota",
		podStatus: &v1.PodStatus{
			Conditions: []v1.PodCondition{
				{
				Type: v1.PodScheduled,
				Status: v1.ConditionFalse,
			    Reason: v1.PodReasonUnschedulable,
				Message: fmt.Sprintf("0/5 nodes are available: 5 Insufficient nvidia.com/gpu."),
				},
			},
		},
		expectedCategory: CategorySchedulingFailure,
	    expectedReason: "Unschedulable",
		expectedCode: "UNSCHEDULABLE",
	    expectedExitCode: -1,
    },
	  {
		name: "Node Evicted failure",
		podStatus: &v1.PodStatus{
			Reason: "Evicted",
			Message: fmt.Sprintf("The node was low on resource: ephemeral-storage."),
		},
		expectedCategory: CategoryNodeEviction,
		expectedReason: "Evicted",
		expectedCode: "NODE_EVICTED",
		expectedExitCode: -1,
	  },
	  {
		name: "Invalid StorageClass failure",
		podStatus: &v1.PodStatus{
			Reason: "FailedBinding",
			Message: fmt.Sprintf("StorageClass 'invalid-storage-class' not found."),
		},
		expectedCategory: CategoryProvisioningFailure,
		expectedReason: "FailedBinding",
		expectedCode: "INVALID_STORAGE_CLASS",
		expectedExitCode: -1,
	  },
	  {
		name: "CrashLoopBackOff failure",
		podStatus: &v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					Image: "python:3.11-slim",
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
							Message: fmt.Sprintf("Back-off 5m0s restarting failed container"),
						},
					},
				},
			},
		},
		expectedCategory: CategoryRuntimeCrash,
        expectedReason: "CrashLoopBackOff",
		expectedCode: "CRASH_LOOP_BACKOFF",
		expectedExitCode: -1,
	  },
	}


	for _, tt := range tests{
		t.Run(tt.name, func(t *testing.T){
			diag := ClassifyPodStatus(tt.podStatus)
			assert.NotNil(t, diag)
			assert.Equal(t, tt.expectedCategory, diag.Category)
			assert.NotEmpty(t, diag.ErrorCode)
			assert.NotEmpty(t, diag.ErrorMessage)
		})
	}
}

