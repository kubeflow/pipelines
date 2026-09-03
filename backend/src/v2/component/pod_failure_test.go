// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_podFailureFields_waitingReason_ImagePullBackOff(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "main",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "ImagePullBackOff",
							Message: "Back-off pulling image",
						},
					},
				},
			},
		},
	}

	fields := podFailureFields(pod)
	require.NotNil(t, fields)
	require.Equal(t, "p", fields["podName"].GetStringValue())
	require.Equal(t, "ns", fields["namespace"].GetStringValue())
	require.Equal(t, "main", fields["containerName"].GetStringValue())
	require.Equal(t, "ImagePullBackOff", fields["reason"].GetStringValue())
	require.Equal(t, "Back-off pulling image", fields["kubernetesMessage"].GetStringValue())
}

func Test_podFailureFields_terminatedReason_OOMKilled(t *testing.T) {
	exitCode := int32(137)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "main",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "OOMKilled",
							Message:  "Container killed due to memory",
							ExitCode: exitCode,
						},
					},
				},
			},
		},
	}

	fields := podFailureFields(pod)
	require.NotNil(t, fields)
	require.Equal(t, "main", fields["containerName"].GetStringValue())
	require.Equal(t, "OOMKilled", fields["reason"].GetStringValue())
	require.Equal(t, float64(137), fields["containerExitCode"].GetNumberValue())
	require.Equal(t, "Container killed due to memory", fields["kubernetesMessage"].GetStringValue())
}

func Test_podFailureFields_crashLoopBackOff(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-123", Namespace: "kubeflow"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "main",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "CrashLoopBackOff",
							Message: "back-off 5m0s restarting failed container",
						},
					},
				},
			},
		},
	}

	fields := podFailureFields(pod)
	require.NotNil(t, fields)
	require.Equal(t, "pod-123", fields["podName"].GetStringValue())
	require.Equal(t, "kubeflow", fields["namespace"].GetStringValue())
	require.Equal(t, "CrashLoopBackOff", fields["reason"].GetStringValue())
}

func Test_podFailureFields_initContainer(t *testing.T) {
	exitCode := int32(1)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "init",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "Error",
							ExitCode: exitCode,
						},
					},
				},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "main",
					State: corev1.ContainerState{},
				},
			},
		},
	}

	fields := podFailureFields(pod)
	require.NotNil(t, fields)
	require.Equal(t, "init", fields["containerName"].GetStringValue())
	require.Equal(t, "Error", fields["reason"].GetStringValue())
	require.Equal(t, float64(1), fields["containerExitCode"].GetNumberValue())
}

func Test_podFailureFields_nilPod(t *testing.T) {
	fields := podFailureFields(nil)
	require.Nil(t, fields)
}

func Test_podFailureFields_noFailure(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "main",
					State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				},
			},
		},
	}

	fields := podFailureFields(pod)
	require.Nil(t, fields)
}

func Test_bestContainerFailure_waitingPriorityOverTerminated(t *testing.T) {
	exitCode := int32(137)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "main",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "OOMKilled",
							ExitCode: exitCode,
						},
					},
				},
				{
					Name: "sidecar",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "ImagePullBackOff",
							Message: "image not found",
						},
					},
				},
			},
		},
	}

	best := bestContainerFailure(pod)
	require.Equal(t, "sidecar", best.containerName)
	require.Equal(t, "ImagePullBackOff", best.reason)
}

func Test_bestContainerFailure_lastTerminationState(t *testing.T) {
	exitCode := int32(137)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "main",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "CrashLoopBackOff",
							Message: "restart",
						},
					},
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "OOMKilled",
							ExitCode: exitCode,
						},
					},
				},
			},
		},
	}

	best := bestContainerFailure(pod)
	require.Equal(t, "CrashLoopBackOff", best.reason)
	require.Nil(t, best.exitCode)
}

func Test_exitCodeFromError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantOk   bool
	}{
		{
			name:   "non-ExitError",
			err:    fmt.Errorf("some error"),
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, ok := exitCodeFromError(tt.err)
			require.Equal(t, tt.wantCode, code)
			require.Equal(t, tt.wantOk, ok)
		})
	}
}
