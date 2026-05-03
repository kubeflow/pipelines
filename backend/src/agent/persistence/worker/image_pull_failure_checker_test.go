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

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// fakeExecutionInterface records Patch calls for testing.
type fakeExecutionInterface struct {
	util.ExecutionInterface
	patchCalled bool
	patchedName string
	patchData   []byte
}

func (f *fakeExecutionInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (util.ExecutionSpec, error) {
	f.patchCalled = true
	f.patchedName = name
	f.patchData = data
	return nil, nil
}

// fakeExecutionClient returns a fakeExecutionInterface for testing.
type fakeExecutionClient struct {
	executionInterface *fakeExecutionInterface
	namespace          string
}

func (f *fakeExecutionClient) Execution(namespace string) util.ExecutionInterface {
	f.namespace = namespace
	return f.executionInterface
}

func (f *fakeExecutionClient) Compare(old, new interface{}) bool {
	return true
}

// newTestPodLister creates a pod lister backed by a fake clientset and shared
// informer. The provided pods are pre-created in the fake clientset before the
// informer cache is synced.
func newTestPodLister(pods ...*corev1.Pod) corelisters.PodLister {
	ctx := context.Background()
	kubeClient := fake.NewClientset()
	for _, pod := range pods {
		kubeClient.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	}

	factory := informers.NewSharedInformerFactory(kubeClient, 0)
	podInformer := factory.Core().V1().Pods()
	podLister := podInformer.Lister()

	stopChannel := make(chan struct{})
	defer close(stopChannel)
	factory.Start(stopChannel)
	cache.WaitForCacheSync(stopChannel, podInformer.Informer().HasSynced)

	return podLister
}

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
	podLister := newTestPodLister()
	checker := NewImagePullFailureChecker(podLister, nil, 5*time.Minute)

	err := checker.CheckAndTerminate("default", "my-workflow")
	assert.NoError(t, err)
}

func TestCheckAndTerminate_HealthyPods(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "healthy-pod",
			Namespace: "default",
			Labels:    map[string]string{ArgoWorkflowLabelKey: "my-workflow"},
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

	podLister := newTestPodLister(pod)
	checker := NewImagePullFailureChecker(podLister, nil, 5*time.Minute)
	err := checker.CheckAndTerminate("default", "my-workflow")
	assert.NoError(t, err)
}

func TestCheckAndTerminate_ImagePullFailureWithinGracePeriod(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "failing-pod",
			Namespace:         "default",
			Labels:            map[string]string{ArgoWorkflowLabelKey: "my-workflow"},
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

	podLister := newTestPodLister(pod)
	// Grace period is 1 hour, pod was just created -- should not terminate.
	checker := NewImagePullFailureChecker(podLister, nil, 1*time.Hour)
	err := checker.CheckAndTerminate("default", "my-workflow")
	assert.NoError(t, err)
}

func TestCheckAndTerminate_ImagePullFailureExceedsGracePeriod(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "failing-pod",
			Namespace:         "default",
			Labels:            map[string]string{ArgoWorkflowLabelKey: "my-workflow"},
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

	podLister := newTestPodLister(pod)
	// Grace period is 5 min, pod is 10 min old -- should attempt to terminate.
	// executionClient is nil so the termination will return an error.
	checker := NewImagePullFailureChecker(podLister, nil, 5*time.Minute)
	err := checker.CheckAndTerminate("default", "my-workflow")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execution client not configured")
}

func TestCheckAndTerminate_MixedPods(t *testing.T) {
	// Healthy pod
	healthyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "healthy-pod",
			Namespace:         "default",
			Labels:            map[string]string{ArgoWorkflowLabelKey: "my-workflow"},
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
			Labels:            map[string]string{ArgoWorkflowLabelKey: "my-workflow"},
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

	podLister := newTestPodLister(healthyPod, newFailingPod)
	// Grace period is 5 min, only the new failing pod has issues but is within grace -- should not terminate.
	checker := NewImagePullFailureChecker(podLister, nil, 5*time.Minute)
	err := checker.CheckAndTerminate("default", "my-workflow")
	assert.NoError(t, err)
}

func TestCheckAndTerminate_OnlyListsPodsForWorkflow(t *testing.T) {
	// Pod belonging to a different workflow, old and failing.
	otherPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "other-pod",
			Namespace:         "default",
			Labels:            map[string]string{ArgoWorkflowLabelKey: "other-workflow"},
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

	podLister := newTestPodLister(otherPod)
	// Checking "my-workflow" should not see "other-workflow" pods.
	checker := NewImagePullFailureChecker(podLister, nil, 5*time.Minute)
	err := checker.CheckAndTerminate("default", "my-workflow")
	assert.NoError(t, err)
}

func TestCheckAndTerminate_SuccessfulTermination(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "failing-pod",
			Namespace:         "default",
			Labels:            map[string]string{ArgoWorkflowLabelKey: "my-workflow"},
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

	podLister := newTestPodLister(pod)
	fakeExecInterface := &fakeExecutionInterface{}
	fakeExecClient := &fakeExecutionClient{executionInterface: fakeExecInterface}

	checker := NewImagePullFailureChecker(podLister, fakeExecClient, 5*time.Minute)
	err := checker.CheckAndTerminate("default", "my-workflow")

	assert.NoError(t, err)
	assert.True(t, fakeExecInterface.patchCalled, "Patch should have been called to terminate the workflow")
	assert.Equal(t, "my-workflow", fakeExecInterface.patchedName)
	assert.Equal(t, "default", fakeExecClient.namespace)
	assert.Contains(t, string(fakeExecInterface.patchData), "activeDeadlineSeconds")
}
