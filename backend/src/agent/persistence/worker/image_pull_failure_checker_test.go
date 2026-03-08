// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetImagePullFailure_NoFailure(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Image: "myimage:latest",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
		},
	}
	assert.Equal(t, "", getImagePullFailure(pod))
}

func TestGetImagePullFailure_ImagePullBackOff(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Image: "nonexistent-registry.io/myimage:latest",
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
	assert.Equal(t, "nonexistent-registry.io/myimage:latest", getImagePullFailure(pod))
}

func TestGetImagePullFailure_ErrImagePull(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Image: "myregistry.io/badimage:v1",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "ErrImagePull",
							Message: "pull access denied",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, "myregistry.io/badimage:v1", getImagePullFailure(pod))
}

func TestGetImagePullFailure_InitContainerFailure(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Image: "init-image:v1",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "ImagePullBackOff",
						},
					},
				},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Image: "main-image:v1",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "PodInitializing",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, "init-image:v1", getImagePullFailure(pod))
}

func TestGetImagePullFailure_OtherWaitingReason(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Image: "myimage:latest",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "ContainerCreating",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, "", getImagePullFailure(pod))
}

func TestCheckAndTerminate_NoPods(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	checker := NewImagePullFailureChecker(kubeClient, nil, 5*time.Minute)

	err := checker.CheckAndTerminate("default", "my-workflow")
	assert.NoError(t, err)
}

func TestCheckAndTerminate_HealthyPods(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	ctx := context.Background()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "healthy-pod",
			Namespace: "default",
			Labels:    map[string]string{argoWorkflowLabelKey: "my-workflow"},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Image: "good-image:latest",
					State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				},
			},
		},
	}
	_, err := kubeClient.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
	assert.NoError(t, err)

	checker := NewImagePullFailureChecker(kubeClient, nil, 5*time.Minute)
	err = checker.CheckAndTerminate("default", "my-workflow")
	assert.NoError(t, err)
}

func TestCheckAndTerminate_ImagePullFailureWithinGracePeriod(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	ctx := context.Background()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "failing-pod",
			Namespace:         "default",
			Labels:            map[string]string{argoWorkflowLabelKey: "my-workflow"},
			CreationTimestamp: metav1.Now(), // just created
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Image: "bad-image:latest",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"},
					},
				},
			},
		},
	}
	_, err := kubeClient.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Grace period is 1 hour, pod was just created -- should not terminate.
	checker := NewImagePullFailureChecker(kubeClient, nil, 1*time.Hour)
	err = checker.CheckAndTerminate("default", "my-workflow")
	assert.NoError(t, err)
}

func TestCheckAndTerminate_ImagePullFailureExceedsGracePeriod(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	ctx := context.Background()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "failing-pod",
			Namespace:         "default",
			Labels:            map[string]string{argoWorkflowLabelKey: "my-workflow"},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Minute)), // created 10 min ago
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Image: "bad-image:latest",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"},
					},
				},
			},
		},
	}
	_, err := kubeClient.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Grace period is 5 min, pod is 10 min old -- should attempt to terminate.
	// executionClient is nil so the termination will return an error.
	checker := NewImagePullFailureChecker(kubeClient, nil, 5*time.Minute)
	err = checker.CheckAndTerminate("default", "my-workflow")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execution client not configured")
}

func TestCheckAndTerminate_MixedPods(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	ctx := context.Background()

	// Healthy pod
	healthyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "healthy-pod",
			Namespace:         "default",
			Labels:            map[string]string{argoWorkflowLabelKey: "my-workflow"},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Minute)),
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Image: "good-image:latest",
					State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				},
			},
		},
	}

	// Failing pod within grace period
	newFailingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "new-failing-pod",
			Namespace:         "default",
			Labels:            map[string]string{argoWorkflowLabelKey: "my-workflow"},
			CreationTimestamp: metav1.Now(),
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Image: "bad-image:latest",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "ErrImagePull"},
					},
				},
			},
		},
	}

	_, err := kubeClient.CoreV1().Pods("default").Create(ctx, healthyPod, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = kubeClient.CoreV1().Pods("default").Create(ctx, newFailingPod, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Grace period is 5 min, only the new failing pod has issues but is within grace -- should not terminate.
	checker := NewImagePullFailureChecker(kubeClient, nil, 5*time.Minute)
	err = checker.CheckAndTerminate("default", "my-workflow")
	assert.NoError(t, err)
}

func TestCheckAndTerminate_OnlyListsPodsForWorkflow(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	ctx := context.Background()

	// Pod belonging to a different workflow, old and failing.
	otherPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "other-pod",
			Namespace:         "default",
			Labels:            map[string]string{argoWorkflowLabelKey: "other-workflow"},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Minute)),
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Image: "bad-image:latest",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"},
					},
				},
			},
		},
	}

	_, err := kubeClient.CoreV1().Pods("default").Create(ctx, otherPod, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Checking "my-workflow" should not see "other-workflow" pods.
	checker := NewImagePullFailureChecker(kubeClient, nil, 5*time.Minute)
	err = checker.CheckAndTerminate("default", "my-workflow")
	assert.NoError(t, err)
}
