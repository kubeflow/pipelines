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
	"encoding/json"
	"fmt"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argocommon "github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"
)

// fakeExecutionInterface records Patch calls for testing and can emulate the
// API server's precondition checks and a stalled request.
type fakeExecutionInterface struct {
	util.ExecutionInterface
	patchCount  int
	patchedName string
	patchData   []byte
	// liveUID and liveResourceVersion, when set, emulate the API server
	// rejecting a patch whose metadata.resourceVersion (Conflict) or
	// metadata.uid (Invalid, immutable field) does not match the live object.
	liveUID             types.UID
	liveResourceVersion string
	// blockUntilCanceled makes Patch hang until the context is done.
	blockUntilCanceled bool
}

func (f *fakeExecutionInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (util.ExecutionSpec, error) {
	f.patchCount++
	f.patchedName = name
	f.patchData = data
	if f.blockUntilCanceled {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if f.liveUID != "" || f.liveResourceVersion != "" {
		var patch struct {
			Metadata metav1.ObjectMeta `json:"metadata"`
		}
		if err := json.Unmarshal(data, &patch); err != nil {
			return nil, err
		}
		if patch.Metadata.ResourceVersion != f.liveResourceVersion {
			return nil, apierrors.NewConflict(workflowapi.Resource("workflows"), name, fmt.Errorf("the object has been modified"))
		}
		if patch.Metadata.UID != f.liveUID {
			return nil, apierrors.NewInvalid(workflowapi.SchemeGroupVersion.WithKind("Workflow").GroupKind(), name,
				field.ErrorList{field.Invalid(field.NewPath("metadata", "uid"), patch.Metadata.UID, "field is immutable")})
		}
	}
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

// newTestPodIndexer creates an in-memory pod indexer, with the workflow index
// the checker relies on, holding the provided pods. Tests can update or delete
// pods on it between checks.
func newTestPodIndexer(pods ...*corev1.Pod) cache.Indexer {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{WorkflowPodIndexName: WorkflowPodIndexFunc})
	for _, pod := range pods {
		if err := indexer.Add(pod); err != nil {
			panic(err)
		}
	}
	return indexer
}

// fakeClock is a manually advanced clock for exercising the grace period.
type fakeClock struct {
	current time.Time
}

func (c *fakeClock) now() time.Time {
	return c.current
}

func (c *fakeClock) advance(d time.Duration) {
	c.current = c.current.Add(d)
}

// newTestChecker builds a checker on top of the given indexer with a fake clock.
func newTestChecker(podIndexer cache.Indexer, executionClient util.ExecutionClient, gracePeriod time.Duration) (*imagePullFailureChecker, *fakeClock) {
	clock := &fakeClock{current: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	checker := NewImagePullFailureChecker(podIndexer, executionClient, gracePeriod).(*imagePullFailureChecker)
	checker.now = clock.now
	return checker, clock
}

// testWorkflowUID returns the UID used for the workflow with the given name in
// these tests.
func testWorkflowUID(workflowName string) types.UID {
	return types.UID(workflowName + "-uid")
}

// testWorkflowMeta returns the metadata of the workflow as the saver would
// pass it from the informer cache.
func testWorkflowMeta(workflowName string) *metav1.ObjectMeta {
	return &metav1.ObjectMeta{
		Namespace:       "default",
		Name:            workflowName,
		UID:             testWorkflowUID(workflowName),
		ResourceVersion: "100",
	}
}

// newWorkflowPod builds a pod owned by workflowName (both labeled and with a
// controller owner reference, as Argo creates them) whose single main
// container is in the given waiting state. An empty reason yields a running
// container.
func newWorkflowPod(name, workflowName, image, waitingReason string) *corev1.Pod {
	state := corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}
	if waitingReason != "" {
		state = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: waitingReason}}
	}
	controller := true
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       types.UID(name + "-uid"),
			Labels:    map[string]string{ArgoWorkflowLabelKey: workflowName},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "argoproj.io/v1alpha1",
				Kind:       "Workflow",
				Name:       workflowName,
				UID:        testWorkflowUID(workflowName),
				Controller: &controller,
			}},
		},
		Status: corev1.PodStatus{
			Phase:             corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{{Name: "main", Image: image, State: state}},
		},
	}
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
	assert.Empty(t, getImagePullFailures(pod))
}

func TestGetImagePullFailure_ImagePullBackOff(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "main",
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
	assert.Equal(t, []imagePullFailure{{container: "main", image: "nonexistent-registry.io/myimage:latest"}}, getImagePullFailures(pod))
}

func TestGetImagePullFailure_ErrImagePull(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "main",
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
	assert.Equal(t, []imagePullFailure{{container: "main", image: "myregistry.io/badimage:v1"}}, getImagePullFailures(pod))
}

func TestGetImagePullFailure_InitContainerFailure(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "init",
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
					Name:  "main",
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
	assert.Equal(t, []imagePullFailure{{container: "init", image: "init-image:v1"}}, getImagePullFailures(pod))
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
	assert.Empty(t, getImagePullFailures(pod))
}

func TestIsPodTerminal(t *testing.T) {
	now := metav1.Now()
	tests := []struct {
		name     string
		pod      *corev1.Pod
		terminal bool
	}{
		{"pending", &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}}, false},
		{"running", &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning}}, false},
		{"succeeded", &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodSucceeded}}, true},
		{"failed", &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}}, true},
		{"deleting", &corev1.Pod{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now}, Status: corev1.PodStatus{Phase: corev1.PodRunning}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.terminal, isPodTerminal(tc.pod))
		})
	}
}

func TestCheckAndTerminate_NoPods(t *testing.T) {
	podIndexer := newTestPodIndexer()
	checker, _ := newTestChecker(podIndexer, nil, 5*time.Minute)

	err := checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow"))
	assert.NoError(t, err)
}

func TestCheckAndTerminate_HealthyPods(t *testing.T) {
	podIndexer := newTestPodIndexer(newWorkflowPod("healthy-pod", "my-workflow", "good-image:latest", ""))
	checker, _ := newTestChecker(podIndexer, nil, 5*time.Minute)

	err := checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow"))
	assert.NoError(t, err)
}

func TestCheckAndTerminate_ImagePullFailureWithinGracePeriod(t *testing.T) {
	podIndexer := newTestPodIndexer(newWorkflowPod("failing-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff"))
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	// First observation starts the clock; nothing should be terminated yet.
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(4 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 0, fakeExecInterface.patchCount)
}

func TestCheckAndTerminate_ImagePullFailureExceedsGracePeriod(t *testing.T) {
	podIndexer := newTestPodIndexer(newWorkflowPod("failing-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff"))
	// executionClient is nil so the termination will return an error.
	checker, clock := newTestChecker(podIndexer, nil, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(5 * time.Minute)

	err := checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execution client not configured")
}

func TestCheckAndTerminate_GracePeriodStartsAtFailureNotPodCreation(t *testing.T) {
	// The pod was created long ago (e.g. spent a long time Pending), but the
	// image pull failure has only just appeared. The grace period must start
	// from the first observed failure, not from pod creation.
	pod := newWorkflowPod("failing-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff")
	pod.CreationTimestamp = metav1.NewTime(time.Now().Add(-time.Hour))
	podIndexer := newTestPodIndexer(pod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 0, fakeExecInterface.patchCount, "old pod with a fresh failure must get the full grace period")

	clock.advance(5 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 1, fakeExecInterface.patchCount)
}

func TestCheckAndTerminate_RecoveryResetsGracePeriod(t *testing.T) {
	pod := newWorkflowPod("flaky-pod", "my-workflow", "flaky-image:latest", "ErrImagePull")
	indexer := newTestPodIndexer(pod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(indexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	// Failing for 4 minutes, then the pull recovers.
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(4 * time.Minute)
	require.NoError(t, indexer.Update(newWorkflowPod("flaky-pod", "my-workflow", "flaky-image:latest", "")))
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))

	// It fails again 2 minutes later: the clock restarts, so 4 more minutes is
	// still inside the grace period.
	clock.advance(2 * time.Minute)
	require.NoError(t, indexer.Update(newWorkflowPod("flaky-pod", "my-workflow", "flaky-image:latest", "ImagePullBackOff")))
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(4 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 0, fakeExecInterface.patchCount)

	clock.advance(1 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 1, fakeExecInterface.patchCount)
}

func TestCheckAndTerminate_IgnoresTerminalPods(t *testing.T) {
	// A retained failed pod (e.g. killed by a task deadline while an init
	// container was in ImagePullBackOff) must not fail a workflow whose retry
	// is healthy.
	oldPod := newWorkflowPod("task-attempt-1", "my-workflow", "main-image:v1", "PodInitializing")
	oldPod.Status.Phase = corev1.PodFailed
	oldPod.Status.InitContainerStatuses = []corev1.ContainerStatus{{
		Image: "init-image:v1",
		State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}},
	}}
	retryPod := newWorkflowPod("task-attempt-2", "my-workflow", "main-image:v1", "")
	retryPod.Status.Phase = corev1.PodRunning

	podIndexer := newTestPodIndexer(oldPod, retryPod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(time.Hour)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 0, fakeExecInterface.patchCount, "terminal pods must not trigger termination")
}

func TestCheckAndTerminate_MixedPods(t *testing.T) {
	healthyPod := newWorkflowPod("healthy-pod", "my-workflow", "good-image:latest", "")
	newFailingPod := newWorkflowPod("new-failing-pod", "my-workflow", "bad-image:latest", "ErrImagePull")

	podIndexer := newTestPodIndexer(healthyPod, newFailingPod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	// Only the failing pod has issues but it is within grace -- should not terminate.
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 0, fakeExecInterface.patchCount)
}

func TestCheckAndTerminate_OnlyListsPodsForWorkflow(t *testing.T) {
	// Pod belonging to a different workflow, failing.
	podIndexer := newTestPodIndexer(newWorkflowPod("other-pod", "other-workflow", "bad-image:latest", "ImagePullBackOff"))
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	// Checking "my-workflow" should not see "other-workflow" pods.
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(time.Hour)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 0, fakeExecInterface.patchCount)
}

func TestCheckAndTerminate_SuccessfulTermination(t *testing.T) {
	podIndexer := newTestPodIndexer(newWorkflowPod("failing-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff"))
	fakeExecInterface := &fakeExecutionInterface{}
	fakeExecClient := &fakeExecutionClient{executionInterface: fakeExecInterface}
	checker, clock := newTestChecker(podIndexer, fakeExecClient, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(5 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))

	require.Equal(t, 1, fakeExecInterface.patchCount, "Patch should have been called to terminate the workflow")
	assert.Equal(t, "my-workflow", fakeExecInterface.patchedName)
	assert.Equal(t, "default", fakeExecClient.namespace)

	var patch struct {
		Metadata struct {
			UID             string            `json:"uid"`
			ResourceVersion string            `json:"resourceVersion"`
			Annotations     map[string]string `json:"annotations"`
		} `json:"metadata"`
		Spec struct {
			ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds"`
			Shutdown              string `json:"shutdown"`
		} `json:"spec"`
	}
	require.NoError(t, json.Unmarshal(fakeExecInterface.patchData, &patch))
	// The inspected identity is carried as a precondition so a retried or
	// recreated run under the same name is never terminated by a stale decision.
	assert.Equal(t, string(testWorkflowUID("my-workflow")), patch.Metadata.UID)
	assert.Equal(t, "100", patch.Metadata.ResourceVersion)
	assert.Equal(t, "ImagePullFailure", patch.Metadata.Annotations["pipelines.kubeflow.org/termination-reason"])
	assert.Equal(t, "bad-image:latest", patch.Metadata.Annotations["pipelines.kubeflow.org/failed-image"])
	require.NotNil(t, patch.Spec.ActiveDeadlineSeconds)
	assert.Equal(t, int64(0), *patch.Spec.ActiveDeadlineSeconds, "KFP marks a run terminated via activeDeadlineSeconds=0")
	// An ordinary task failure must use the deadline only, so that healthy
	// exit handlers (which Argo exempts from the deadline) still run.
	assert.Empty(t, patch.Spec.Shutdown)

	// Tracking state is dropped after a successful termination.
	assert.Empty(t, checker.failureStart)
}

func TestCheckAndTerminate_TerminatesExitHandlerPod(t *testing.T) {
	// Argo labels exit-handler pods with the workflow name just like any other
	// task pod. Once the main task has completed and only the exit handler is
	// stuck pulling its image, the workflow must still be terminated.
	mainPod := newWorkflowPod("my-workflow-main", "my-workflow", "good-image:latest", "")
	mainPod.Status.Phase = corev1.PodSucceeded
	exitHandlerPod := newWorkflowPod("my-workflow-onexit", "my-workflow", "missing-exit-image:latest", "ImagePullBackOff")
	exitHandlerPod.Labels[argocommon.LabelKeyOnExit] = "true"

	podIndexer := newTestPodIndexer(mainPod, exitHandlerPod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(5 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))

	require.Equal(t, 1, fakeExecInterface.patchCount)
	// Exit-handler pods are exempt from the workflow deadline, so the Terminate
	// shutdown strategy is required to stop an exit handler with a bad image.
	assert.Contains(t, string(fakeExecInterface.patchData), `"shutdown":"Terminate"`)
	assert.Contains(t, string(fakeExecInterface.patchData), `"activeDeadlineSeconds":0`)
	assert.Contains(t, string(fakeExecInterface.patchData), "missing-exit-image:latest")
}

func TestCheckAndTerminate_OrdinaryFailurePreservesHealthyExitHandler(t *testing.T) {
	// A task that cannot pull its image must be terminated with the deadline
	// only, so a healthy dsl.ExitHandler cleanup task still runs afterwards.
	taskPod := newWorkflowPod("my-workflow-task", "my-workflow", "bad-image:latest", "ImagePullBackOff")
	cleanupPod := newWorkflowPod("my-workflow-cleanup", "my-workflow", "cleanup-image:latest", "")
	cleanupPod.Labels[argocommon.LabelKeyOnExit] = "true"

	podIndexer := newTestPodIndexer(taskPod, cleanupPod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(5 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))

	require.Equal(t, 1, fakeExecInterface.patchCount)
	assert.Contains(t, string(fakeExecInterface.patchData), `"activeDeadlineSeconds":0`)
	assert.NotContains(t, string(fakeExecInterface.patchData), "shutdown", "healthy exit handlers must be allowed to run")
}

func TestCheckAndTerminate_PrefersExitHandlerWhenBothExpired(t *testing.T) {
	taskPod := newWorkflowPod("my-workflow-task", "my-workflow", "bad-image:latest", "ImagePullBackOff")
	exitHandlerPod := newWorkflowPod("my-workflow-onexit", "my-workflow", "bad-exit-image:latest", "ErrImagePull")
	exitHandlerPod.Labels[argocommon.LabelKeyOnExit] = "true"

	podIndexer := newTestPodIndexer(taskPod, exitHandlerPod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(5 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))

	require.Equal(t, 1, fakeExecInterface.patchCount)
	assert.Contains(t, string(fakeExecInterface.patchData), `"shutdown":"Terminate"`)
	assert.Contains(t, string(fakeExecInterface.patchData), "bad-exit-image:latest")
}

func TestCheckAndTerminate_IgnoresPodWithSpoofedLabel(t *testing.T) {
	// A pod carrying the workflow label but not owned by the workflow (for
	// example created by another actor in the namespace) must never cause
	// the workflow to be terminated.
	spoofedPod := newWorkflowPod("spoofed-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff")
	spoofedPod.OwnerReferences = nil
	wrongUIDPod := newWorkflowPod("stale-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff")
	wrongUIDPod.OwnerReferences[0].UID = "some-other-uid"
	wrongKindPod := newWorkflowPod("job-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff")
	wrongKindPod.OwnerReferences[0].APIVersion = "batch/v1"
	wrongKindPod.OwnerReferences[0].Kind = "Job"

	podIndexer := newTestPodIndexer(spoofedPod, wrongUIDPod, wrongKindPod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(time.Hour)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 0, fakeExecInterface.patchCount, "pods not controlled by the workflow must be ignored")
	assert.Empty(t, checker.failureStart, "ignored pods must not be tracked")
}

func TestIsOwnedByWorkflow(t *testing.T) {
	owned := newWorkflowPod("pod", "my-workflow", "img", "")
	assert.True(t, isOwnedByWorkflow(owned, "my-workflow", testWorkflowUID("my-workflow")))
	assert.False(t, isOwnedByWorkflow(owned, "my-workflow", "other-uid"))
	assert.False(t, isOwnedByWorkflow(owned, "other-workflow", testWorkflowUID("my-workflow")))

	nonController := newWorkflowPod("pod", "my-workflow", "img", "")
	nonController.OwnerReferences[0].Controller = nil
	assert.False(t, isOwnedByWorkflow(nonController, "my-workflow", testWorkflowUID("my-workflow")))
}

func TestForget_ResetsGracePeriod(t *testing.T) {
	podIndexer := newTestPodIndexer(newWorkflowPod("failing-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff"))
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Len(t, checker.failureStart, 1)

	checker.Forget("default", "my-workflow")
	assert.Empty(t, checker.failureStart)

	// After Forget the next observation starts a fresh grace period.
	clock.advance(5 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 0, fakeExecInterface.patchCount)
}

func TestCheckAndTerminate_DropsTrackingWhenPodsRecover(t *testing.T) {
	pod := newWorkflowPod("failing-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff")
	indexer := newTestPodIndexer(pod)
	checker, _ := newTestChecker(indexer, nil, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Len(t, checker.failureStart, 1)

	require.NoError(t, indexer.Delete(pod))
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Empty(t, checker.failureStart, "no failing pods means no tracking state")
}

func TestCheckAndTerminate_RequiresWorkflowIdentity(t *testing.T) {
	podIndexer := newTestPodIndexer(newWorkflowPod("failing-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff"))
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	meta := testWorkflowMeta("my-workflow")
	meta.ResourceVersion = ""
	require.NoError(t, checker.CheckAndTerminate(context.Background(), meta))
	clock.advance(5 * time.Minute)

	err := checker.CheckAndTerminate(context.Background(), meta)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no UID or resource version")
	assert.Equal(t, 0, fakeExecInterface.patchCount, "no unconditional patch may be sent")
}

func TestCheckAndTerminate_RetriedRunBetweenDetectionAndPatch(t *testing.T) {
	// The checker decides against a cached workflow (resourceVersion 100). By
	// the time it patches, the user has retried the run, which KFP implements
	// by updating the same workflow name (resourceVersion 101). The stale
	// decision must be discarded and the refreshed workflow re-evaluated.
	podIndexer := newTestPodIndexer(newWorkflowPod("failing-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff"))
	fakeExecInterface := &fakeExecutionInterface{
		liveUID:             testWorkflowUID("my-workflow"),
		liveResourceVersion: "101",
	}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	stale := testWorkflowMeta("my-workflow") // resourceVersion 100
	require.NoError(t, checker.CheckAndTerminate(context.Background(), stale))
	clock.advance(5 * time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), stale), "a conflict is not an error, the decision is just discarded")
	assert.Equal(t, 1, fakeExecInterface.patchCount)
	assert.Len(t, checker.failureStart, 1, "tracking state is kept so the refreshed workflow is re-evaluated immediately")

	// The next sync sees the refreshed workflow. The pod is still failing, so
	// the termination now goes through against the current version.
	refreshed := testWorkflowMeta("my-workflow")
	refreshed.ResourceVersion = "101"
	require.NoError(t, checker.CheckAndTerminate(context.Background(), refreshed))
	assert.Equal(t, 2, fakeExecInterface.patchCount)
	assert.Empty(t, checker.failureStart)
}

func TestCheckAndTerminate_RecreatedRunBetweenDetectionAndPatch(t *testing.T) {
	// A retry that recreates the workflow object yields a new UID. The API
	// server rejects the UID change as an immutable-field error, which must be
	// treated like a conflict.
	podIndexer := newTestPodIndexer(newWorkflowPod("failing-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff"))
	fakeExecInterface := &fakeExecutionInterface{
		liveUID:             "recreated-uid",
		liveResourceVersion: "100",
	}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	stale := testWorkflowMeta("my-workflow")
	require.NoError(t, checker.CheckAndTerminate(context.Background(), stale))
	clock.advance(5 * time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), stale))
	assert.Equal(t, 1, fakeExecInterface.patchCount)
	assert.Len(t, checker.failureStart, 1)
}

func TestCheckAndTerminate_StalledPatchIsCanceledByContext(t *testing.T) {
	podIndexer := newTestPodIndexer(newWorkflowPod("failing-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff"))
	fakeExecInterface := &fakeExecutionInterface{blockUntilCanceled: true}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- checker.CheckAndTerminate(ctx, testWorkflowMeta("my-workflow")) }()

	select {
	case err := <-done:
		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(5 * time.Second):
		t.Fatal("CheckAndTerminate did not return after the context deadline")
	}
	assert.Equal(t, 1, fakeExecInterface.patchCount)
	assert.Len(t, checker.failureStart, 1, "a failed termination keeps the tracking state for the next attempt")
}

func TestCheckAndTerminate_RetryGenerationChangeResetsTracking(t *testing.T) {
	// An in-place retry keeps the workflow UID but stamps a new retry
	// generation. If the workflow informer observes the retry before the pod
	// informer observes the deletion of the previous attempt's failing pod,
	// the checker must not reuse that pod's expired failure clock against the
	// new attempt.
	stalePod := newWorkflowPod("failing-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff")
	podIndexer := newTestPodIndexer(stalePod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	first := testWorkflowMeta("my-workflow")
	require.NoError(t, checker.CheckAndTerminate(context.Background(), first))
	clock.advance(5 * time.Minute)

	retried := testWorkflowMeta("my-workflow") // same UID
	retried.ResourceVersion = "101"
	retried.Annotations = map[string]string{util.AnnotationKeyRetryGeneration: "1"}
	require.NoError(t, checker.CheckAndTerminate(context.Background(), retried))
	assert.Equal(t, 0, fakeExecInterface.patchCount, "the new attempt must get a fresh grace period")
	require.Len(t, checker.failureStart, 1)
	assert.Equal(t, "1", checker.failureStart["default/my-workflow"].retryGeneration)

	// Within the same attempt the clock keeps counting, so if the stale pod
	// really does keep failing for a full grace period the attempt is failed.
	clock.advance(5 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), retried))
	assert.Equal(t, 1, fakeExecInterface.patchCount)
}

func TestCheckAndTerminate_RetryGenerationBumpResetsTracking(t *testing.T) {
	// A second retry of an already retried run moves from one non-empty
	// generation to the next; that must reset tracking just like the first.
	podIndexer := newTestPodIndexer(newWorkflowPod("failing-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff"))
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	gen1 := testWorkflowMeta("my-workflow")
	gen1.Annotations = map[string]string{util.AnnotationKeyRetryGeneration: "1"}
	require.NoError(t, checker.CheckAndTerminate(context.Background(), gen1))
	clock.advance(5 * time.Minute)

	gen2 := testWorkflowMeta("my-workflow")
	gen2.ResourceVersion = "102"
	gen2.Annotations = map[string]string{util.AnnotationKeyRetryGeneration: "2"}
	require.NoError(t, checker.CheckAndTerminate(context.Background(), gen2))
	assert.Equal(t, 0, fakeExecInterface.patchCount)
}

func TestCheckAndTerminate_RecreatedWorkflowResetsTracking(t *testing.T) {
	// A retry that recreates the workflow yields a new UID. Pods of the old
	// attempt are no longer owned by the workflow, and a failing pod of the new
	// attempt starts its own grace period rather than inheriting the old one.
	oldPod := newWorkflowPod("old-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff")
	indexer := newTestPodIndexer(oldPod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(indexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(5 * time.Minute)

	recreated := testWorkflowMeta("my-workflow")
	recreated.UID = "recreated-uid"
	newPod := newWorkflowPod("new-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff")
	newPod.OwnerReferences[0].UID = recreated.UID
	require.NoError(t, indexer.Add(newPod))

	require.NoError(t, checker.CheckAndTerminate(context.Background(), recreated))
	assert.Equal(t, 0, fakeExecInterface.patchCount, "the recreated attempt must get a fresh grace period")
	require.Len(t, checker.failureStart, 1)
	tracked := checker.failureStart["default/my-workflow"]
	assert.Equal(t, recreated.UID, tracked.uid)
	assert.Len(t, tracked.failures, 1, "only the new attempt's pod is tracked")
	_, trackedNew := tracked.failures[imagePullFailureKey{podUID: newPod.UID, container: "main", image: "bad-image:latest"}]
	assert.True(t, trackedNew)
}

func TestWorkflowPodIndexFunc(t *testing.T) {
	keys, err := WorkflowPodIndexFunc(newWorkflowPod("pod", "my-workflow", "image", ""))
	require.NoError(t, err)
	assert.Equal(t, []string{"default/my-workflow"}, keys)

	unlabeled := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "plain"}}
	keys, err = WorkflowPodIndexFunc(unlabeled)
	require.NoError(t, err)
	assert.Empty(t, keys, "pods without the workflow label are not indexed")

	_, err = WorkflowPodIndexFunc("not a pod")
	require.Error(t, err)
}

func TestAddWorkflowPodIndex(t *testing.T) {
	informer := cache.NewSharedIndexInformer(&cache.ListWatch{}, &corev1.Pod{}, 0, cache.Indexers{})
	require.NoError(t, AddWorkflowPodIndex(informer))
	_, ok := informer.GetIndexer().GetIndexers()[WorkflowPodIndexName]
	assert.True(t, ok, "the workflow index must be registered on the informer")
}

func TestCheckAndTerminate_UsesWorkflowIndex(t *testing.T) {
	// The lookup must go through the workflow index rather than scanning the
	// cache: an indexer without that index cannot serve the check.
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	require.NoError(t, indexer.Add(newWorkflowPod("failing-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff")))
	checker, _ := newTestChecker(indexer, nil, 5*time.Minute)

	err := checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to look up pods for workflow default/my-workflow")
}

func TestCheckAndTerminate_RecoveredInitContainerDoesNotStartMainContainerClock(t *testing.T) {
	// Check 1 sees the init container failing to pull. Before check 2 the
	// init image pull recovers and the main container starts failing to
	// pull, with no healthy observation in between. The main container's
	// failure must get its own grace period rather than inheriting the init
	// container's expired clock.
	pod := newWorkflowPod("pod", "my-workflow", "main-image:v1", "")
	pod.Status.InitContainerStatuses = []corev1.ContainerStatus{{
		Name:  "init",
		Image: "init-image:v1",
		State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}},
	}}
	pod.Status.ContainerStatuses[0].State = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "PodInitializing"}}
	indexer := newTestPodIndexer(pod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(indexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(5 * time.Minute)

	recovered := pod.DeepCopy()
	recovered.Status.InitContainerStatuses[0].State = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}}
	recovered.Status.ContainerStatuses[0].State = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ErrImagePull"}}
	require.NoError(t, indexer.Update(recovered))

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 0, fakeExecInterface.patchCount, "the main container's failure has only just started")
	tracked := checker.failureStart["default/my-workflow"]
	require.NotNil(t, tracked)
	assert.Len(t, tracked.failures, 1)
	_, initTracked := tracked.failures[imagePullFailureKey{podUID: pod.UID, container: "init", image: "init-image:v1"}]
	assert.False(t, initTracked, "the recovered init container's entry is dropped")
	start, mainTracked := tracked.failures[imagePullFailureKey{podUID: pod.UID, container: "main", image: "main-image:v1"}]
	assert.True(t, mainTracked)
	assert.Equal(t, clock.current, start)

	// Once the main container has failed for a full grace period of its own,
	// the workflow is terminated for that image.
	clock.advance(5 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 1, fakeExecInterface.patchCount)
	assert.Contains(t, string(fakeExecInterface.patchData), "main-image:v1")
}

func TestCheckAndTerminate_ImageChangeResetsClock(t *testing.T) {
	// The same container failing on a different image (for example after the
	// pod spec was mutated by a webhook or the tag was re-resolved) is a new
	// failure and starts a fresh grace period.
	pod := newWorkflowPod("pod", "my-workflow", "image:v1", "ImagePullBackOff")
	indexer := newTestPodIndexer(pod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(indexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(5 * time.Minute)

	updated := pod.DeepCopy()
	updated.Status.ContainerStatuses[0].Image = "image:v2"
	require.NoError(t, indexer.Update(updated))
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 0, fakeExecInterface.patchCount)
}
