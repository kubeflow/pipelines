// Copyright 2026 The Kubeflow Authors
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
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

// newTestCheckerWithConditions builds a checker with the given conditions on
// top of the indexer, with a fake clock.
func newTestCheckerWithConditions(podIndexer cache.Indexer, executionClient util.ExecutionClient, conditions ...StuckPodCondition) (*stuckPodChecker, *fakeClock) {
	clock := &fakeClock{current: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	checker := NewStuckPodChecker(podIndexer, indexerPodGetter(podIndexer), executionClient, conditions...).(*stuckPodChecker)
	checker.now = clock.now
	return checker, clock
}

// newUnschedulablePod builds a Pending pod owned by workflowName that the
// scheduler reported as unschedulable with the given message.
func newUnschedulablePod(name, workflowName, message string) *corev1.Pod {
	pod := newWorkflowPod(name, workflowName, "image:latest", "")
	pod.Status.ContainerStatuses = nil
	pod.Status.Phase = corev1.PodPending
	pod.Status.Conditions = []corev1.PodCondition{{
		Type:    corev1.PodScheduled,
		Status:  corev1.ConditionFalse,
		Reason:  corev1.PodReasonUnschedulable,
		Message: message,
	}}
	return pod
}

func TestGetUnschedulableFailures(t *testing.T) {
	unschedulable := newUnschedulablePod("p", "my-workflow", "0/3 nodes are available: 3 Insufficient cpu.")
	assert.Equal(t, []stuckPodFailure{{message: "0/3 nodes are available: 3 Insufficient cpu."}}, getUnschedulableFailures(unschedulable))

	scheduled := newUnschedulablePod("p", "my-workflow", "")
	scheduled.Spec.NodeName = "node-1"
	assert.Empty(t, getUnschedulableFailures(scheduled), "a pod bound to a node is not unschedulable")

	running := newUnschedulablePod("p", "my-workflow", "")
	running.Status.Phase = corev1.PodRunning
	assert.Empty(t, getUnschedulableFailures(running), "only Pending pods can be unschedulable")

	gated := newUnschedulablePod("p", "my-workflow", "")
	gated.Status.Conditions[0].Reason = corev1.PodReasonSchedulingGated
	assert.Empty(t, getUnschedulableFailures(gated), "a pod held back by scheduling gates is waiting on a controller, not the scheduler")

	placed := newUnschedulablePod("p", "my-workflow", "")
	placed.Status.Conditions[0].Status = corev1.ConditionTrue
	assert.Empty(t, getUnschedulableFailures(placed))

	assert.Empty(t, getUnschedulableFailures(&corev1.Pod{}))
}

func TestCheckAndTerminate_UnschedulablePodTerminatesAfterGracePeriod(t *testing.T) {
	podIndexer := newTestPodIndexer(newUnschedulablePod("big-pod", "my-workflow", "0/3 nodes are available: 3 Insufficient memory."))
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestCheckerWithConditions(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, UnschedulableCondition(time.Hour))

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(59 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 0, fakeExecInterface.patchCount, "the pod may still be scheduled within the grace period")

	clock.advance(time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	require.Equal(t, 1, fakeExecInterface.patchCount)

	var patch struct {
		Metadata struct {
			Annotations map[string]string `json:"annotations"`
		} `json:"metadata"`
		Spec struct {
			ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds"`
		} `json:"spec"`
	}
	require.NoError(t, json.Unmarshal(fakeExecInterface.patchData, &patch))
	assert.Equal(t, util.TerminationReasonUnschedulable, patch.Metadata.Annotations[util.AnnotationKeyTerminationReason])
	assert.NotContains(t, patch.Metadata.Annotations, util.AnnotationKeyFailedImage, "an unschedulable pod has no failed image")
	require.NotNil(t, patch.Spec.ActiveDeadlineSeconds)
	assert.Equal(t, int64(0), *patch.Spec.ActiveDeadlineSeconds)
	assert.Empty(t, checker.failureStart, "tracking state is dropped after a successful termination")
}

func TestCheckAndTerminate_UnschedulableMessageChangeKeepsClock(t *testing.T) {
	// The scheduler rewrites the message whenever the cluster changes (node
	// counts, taints). The message is informational, so it must not restart
	// the grace period of a pod that has been unschedulable all along.
	pod := newUnschedulablePod("big-pod", "my-workflow", "0/3 nodes are available: 3 Insufficient cpu.")
	podIndexer := newTestPodIndexer(pod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestCheckerWithConditions(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, UnschedulableCondition(time.Hour))

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(40 * time.Minute)

	changed := pod.DeepCopy()
	changed.Status.Conditions[0].Message = "0/4 nodes are available: 4 Insufficient cpu."
	require.NoError(t, podIndexer.Update(changed))
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(20 * time.Minute)

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 1, fakeExecInterface.patchCount)
}

func TestCheckAndTerminate_PodScheduledWithinGracePeriodResetsClock(t *testing.T) {
	pod := newUnschedulablePod("big-pod", "my-workflow", "0/3 nodes are available.")
	podIndexer := newTestPodIndexer(pod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestCheckerWithConditions(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, UnschedulableCondition(time.Hour))

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(50 * time.Minute)

	// The autoscaler added a node and the pod was placed.
	scheduled := pod.DeepCopy()
	scheduled.Spec.NodeName = "new-node"
	scheduled.Status.Conditions[0].Status = corev1.ConditionTrue
	require.NoError(t, podIndexer.Update(scheduled))
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Empty(t, checker.failureStart, "a scheduled pod must not be tracked")

	clock.advance(time.Hour)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	assert.Equal(t, 0, fakeExecInterface.patchCount)
}

func TestCheckAndTerminate_UnschedulableDecisionIsConfirmedLive(t *testing.T) {
	// The cached pod is unschedulable, but the live pod has been placed.
	cached := newUnschedulablePod("big-pod", "my-workflow", "0/3 nodes are available.")
	podIndexer := newTestPodIndexer(cached)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestCheckerWithConditions(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, UnschedulableCondition(time.Minute))
	checker.getPod = func(_ context.Context, _, _ string) (*corev1.Pod, error) {
		live := cached.DeepCopy()
		live.Spec.NodeName = "node-1"
		live.Status.Conditions[0].Status = corev1.ConditionTrue
		return live, nil
	}

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))

	assert.Equal(t, 0, fakeExecInterface.patchCount, "a pod the live read shows as scheduled must not terminate the workflow")
	assert.Empty(t, checker.failureStart, "a failure the live pod disproved is no longer tracked")
}

func TestCheckAndTerminate_ConditionsHaveIndependentGracePeriods(t *testing.T) {
	// An image pull failure (5 minutes) and an unschedulable pod (1 hour) in
	// the same workflow: the pull failure must terminate first, and the
	// annotations must name it, not the pod still within its longer grace.
	podIndexer := newTestPodIndexer(
		newWorkflowPod("pull-pod", "my-workflow", "bad-image:latest", "ImagePullBackOff"),
		newUnschedulablePod("big-pod", "my-workflow", "0/3 nodes are available."),
	)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestCheckerWithConditions(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface},
		ImagePullCondition(5*time.Minute), UnschedulableCondition(time.Hour))

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(5 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))

	require.Equal(t, 1, fakeExecInterface.patchCount)
	var patch struct {
		Metadata struct {
			Annotations map[string]string `json:"annotations"`
		} `json:"metadata"`
	}
	require.NoError(t, json.Unmarshal(fakeExecInterface.patchData, &patch))
	assert.Equal(t, util.TerminationReasonImagePullFailure, patch.Metadata.Annotations[util.AnnotationKeyTerminationReason])
	assert.Equal(t, "bad-image:latest", patch.Metadata.Annotations[util.AnnotationKeyFailedImage])
}

func TestCheckAndTerminate_OnlyConfiguredConditionsAreChecked(t *testing.T) {
	podIndexer := newTestPodIndexer(newUnschedulablePod("big-pod", "my-workflow", "0/3 nodes are available."))
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestCheckerWithConditions(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, ImagePullCondition(time.Minute))

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(24 * time.Hour)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))

	assert.Equal(t, 0, fakeExecInterface.patchCount, "unschedulable handling is opt-in and must not run when only image pull is configured")
	assert.Empty(t, checker.failureStart)
}

func TestCheckAndTerminate_UnschedulableTrackingIsScopedToWorkflowAttempt(t *testing.T) {
	podIndexer := newTestPodIndexer(newUnschedulablePod("big-pod", "my-workflow", "0/3 nodes are available."))
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestCheckerWithConditions(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, UnschedulableCondition(time.Hour))

	require.NoError(t, checker.CheckAndTerminate(context.Background(), testWorkflowMeta("my-workflow")))
	clock.advance(59 * time.Minute)

	// The run was retried in place: same UID, new retry generation. The pod is
	// still cached from the previous attempt and must not inherit its clock.
	retried := testWorkflowMeta("my-workflow")
	retried.Annotations = map[string]string{util.AnnotationKeyRetryGeneration: "1"}
	clock.advance(2 * time.Minute)
	require.NoError(t, checker.CheckAndTerminate(context.Background(), retried))

	assert.Equal(t, 0, fakeExecInterface.patchCount)
}
