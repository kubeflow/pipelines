// Copyright 2018 The Kubeflow Authors
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
	"fmt"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// fakeImagePullFailureChecker is a test double for ImagePullFailureChecker.
type fakeImagePullFailureChecker struct {
	called        bool
	forgotten     bool
	namespace     string
	workflowName  string
	workflowUID   types.UID
	hadDeadline   bool
	errorToReturn error
	// blockUntilCanceled makes CheckAndTerminate behave like a stalled API
	// request: it returns only once the context is done.
	blockUntilCanceled bool
}

func (f *fakeImagePullFailureChecker) CheckAndTerminate(ctx context.Context, workflow *metav1.ObjectMeta) error {
	f.called = true
	f.namespace = workflow.Namespace
	f.workflowName = workflow.Name
	f.workflowUID = workflow.UID
	_, f.hadDeadline = ctx.Deadline()
	if f.blockUntilCanceled {
		<-ctx.Done()
		return ctx.Err()
	}
	return f.errorToReturn
}

func (f *fakeImagePullFailureChecker) Forget(namespace string, workflowName string) {
	f.forgotten = true
	f.namespace = namespace
	f.workflowName = workflowName
}

func TestWorkflow_Save_Success(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"},
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, false, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.Equal(t, nil, err)
}

func TestWorkflow_Save_NotFoundDuringGet(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, false, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Workflow not found")
}

func TestWorkflow_Save_ErrorDuringGet(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", nil)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, true, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "transient failure")
}

func TestWorkflow_Save_PermanentFailureWhileReporting(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	pipelineFake.SetError(util.NewCustomError(fmt.Errorf("Error"), util.CUSTOM_CODE_PERMANENT,
		"My Permanent Error"))

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"},
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, false, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "permanent failure")
}

func TestWorkflow_Save_TransientFailureWhileReporting(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	pipelineFake.SetError(util.NewCustomError(fmt.Errorf("Error"), util.CUSTOM_CODE_TRANSIENT,
		"My Transient Error"))

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"},
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, true, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "transient failure")
}

func TestWorkflow_Save_RetryableReportSkipsMetrics(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()
	pipelineFake.SetError(util.NewCustomError(fmt.Errorf("stale terminal report"), util.CUSTOM_CODE_TRANSIENT,
		"My Transient Error"))

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"},
		},
		Status: workflowapi.WorkflowStatus{
			Nodes: map[string]workflowapi.NodeStatus{
				"node-1": {
					ID:           "node-1",
					TemplateName: "template-1",
					Phase:        workflowapi.NodeSucceeded,
					Outputs: &workflowapi.Outputs{
						Artifacts: []workflowapi.Artifact{{Name: "mlpipeline-metrics"}},
					},
				},
			},
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, true, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
}

func TestWorkflow_Save_SkippedDueToFinalStatue(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	// Add this will result in failure unless reporting is skipped
	pipelineFake.SetError(util.NewCustomError(fmt.Errorf("Error"), util.CUSTOM_CODE_PERMANENT,
		"My Permanent Error"))

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels:    map[string]string{util.LabelKeyWorkflowPersistedFinalState: "true"},
		},
		Status: workflowapi.WorkflowStatus{
			FinishedAt: metav1.Now(),
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, false, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.Equal(t, nil, err)
}

func TestWorkflow_Save_FinalStatueNotSkippedDueToExceedTTL(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	// Add this will result in failure unless reporting is skipped
	pipelineFake.SetError(util.NewCustomError(fmt.Errorf("Error"), util.CUSTOM_CODE_PERMANENT,
		"My Permanent Error"))

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels: map[string]string{
				util.LabelKeyWorkflowRunId:               "MY_UUID",
				util.LabelKeyWorkflowPersistedFinalState: "true",
			},
		},
		Status: workflowapi.WorkflowStatus{
			FinishedAt: metav1.Now(),
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 1)

	// Sleep 2 seconds to make sure workflow passed TTL
	time.Sleep(2 * time.Second)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, false, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "permanent failure")
}

func TestWorkflow_Save_SkippedDDueToMissingRunID(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	// Add this will result in failure unless reporting is skipped
	pipelineFake.SetError(util.NewCustomError(fmt.Errorf("Error"), util.CUSTOM_CODE_PERMANENT,
		"My Permanent Error"))

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	assert.Equal(t, false, util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT))
	assert.Equal(t, nil, err)
}

func TestWorkflow_Save_CheckerCalledForRunningWorkflow(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()
	checker := &fakeImagePullFailureChecker{}

	// Workflow with Running phase (not in final state).
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			UID:       "MY_WORKFLOW_UID",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"},
		},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowRunning,
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)
	saver.SetImagePullFailureChecker(checker)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	require.NoError(t, err)
	assert.True(t, checker.called, "Checker should be called for running workflow")
	assert.Equal(t, "MY_NAMESPACE", checker.namespace)
	assert.Equal(t, "MY_NAME", checker.workflowName)
	assert.Equal(t, types.UID("MY_WORKFLOW_UID"), checker.workflowUID)
	assert.True(t, checker.hadDeadline, "checker must run under a bounded context")
}

func TestWorkflow_Save_CheckerCalledForPendingWorkflow(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()
	checker := &fakeImagePullFailureChecker{}

	// Workflow with empty phase (Pending/unknown, not in final state).
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"},
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)
	saver.SetImagePullFailureChecker(checker)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	require.NoError(t, err)
	assert.True(t, checker.called, "Checker should be called for non-final-state workflow")
}

func TestWorkflow_Save_CheckerSkippedForCompletedWorkflow(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()
	checker := &fakeImagePullFailureChecker{}

	// Workflow in Succeeded (final) state.
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"},
		},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowSucceeded,
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)
	saver.SetImagePullFailureChecker(checker)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	require.NoError(t, err)
	assert.False(t, checker.called, "Checker should NOT be called for completed workflow")
}

func TestWorkflow_Save_CheckerSkippedForFailedWorkflow(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()
	checker := &fakeImagePullFailureChecker{}

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"},
		},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowFailed,
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)
	saver.SetImagePullFailureChecker(checker)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	require.NoError(t, err)
	assert.False(t, checker.called, "Checker should NOT be called for failed workflow")
	assert.True(t, checker.forgotten, "Checker state should be dropped for failed workflow")
	assert.Equal(t, "MY_NAMESPACE", checker.namespace)
	assert.Equal(t, "MY_NAME", checker.workflowName)
}

func TestWorkflow_Save_CheckerErrorDoesNotBlockReporting(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()
	checker := &fakeImagePullFailureChecker{
		errorToReturn: fmt.Errorf("failed to list pods"),
	}

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"},
		},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowRunning,
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)
	saver.SetImagePullFailureChecker(checker)

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	// Checker error should be logged but Save should still succeed.
	require.NoError(t, err)
	assert.True(t, checker.called)
	// Verify the workflow was still reported to the pipeline server.
	assert.NotNil(t, pipelineFake.GetWorkflow("MY_NAMESPACE", "MY_NAME"))
}

func TestWorkflow_Save_NoCheckerSetDoesNotPanic(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	// Running workflow, but no checker set (feature disabled).
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"},
		},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowRunning,
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)
	// Do NOT set checker — should not panic.

	err := saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20)

	require.NoError(t, err)
}

func TestWorkflow_Save_StalledCheckerDoesNotBlockReporting(t *testing.T) {
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()
	checker := &fakeImagePullFailureChecker{blockUntilCanceled: true}

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"},
		},
		Status: workflowapi.WorkflowStatus{
			Phase: workflowapi.WorkflowRunning,
		},
	})

	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)
	saver.SetImagePullFailureChecker(checker)
	saver.imagePullFailureCheckTimeout = 50 * time.Millisecond

	done := make(chan error, 1)
	go func() { done <- saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20) }()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Save did not return: a stalled checker must be canceled by the per-call timeout")
	}
	assert.True(t, checker.called)
	assert.NotNil(t, pipelineFake.GetWorkflow("MY_NAMESPACE", "MY_NAME"), "workflow must still be reported after the checker timed out")
}

func TestWorkflow_Save_PersistedFinalWorkflowStillForgetsTracking(t *testing.T) {
	// The persisted-final shortcut returns before reporting. Tracking state
	// must still be dropped there, otherwise a workflow that was adopted and
	// persisted as terminal by the API server while this agent still saw it
	// running would keep its consumed grace period until the run is retried.
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()
	checker := &fakeImagePullFailureChecker{}

	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels: map[string]string{
				util.LabelKeyWorkflowRunId:               "MY_UUID",
				util.LabelKeyWorkflowPersistedFinalState: "true",
			},
		},
		Status: workflowapi.WorkflowStatus{
			Phase:      workflowapi.WorkflowFailed,
			FinishedAt: metav1.Now(),
		},
	})
	workflowFake.Put("MY_NAMESPACE", "MY_NAME", workflow)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)
	saver.SetImagePullFailureChecker(checker)

	require.NoError(t, saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20))
	assert.Nil(t, pipelineFake.GetWorkflow("MY_NAMESPACE", "MY_NAME"), "the persisted-final shortcut must still skip reporting")
	assert.True(t, checker.forgotten, "tracking state must be dropped even when the shortcut is taken")
	assert.False(t, checker.called)
	assert.Equal(t, "MY_NAMESPACE", checker.namespace)
	assert.Equal(t, "MY_NAME", checker.workflowName)
}

func TestWorkflow_Save_RetryAfterPersistedCompletionStartsFreshGracePeriod(t *testing.T) {
	// End-to-end through the saver with a real checker: a running workflow
	// consumes its grace period, is then observed already persisted as
	// terminal (so the saver takes the shortcut), and is finally retried in
	// place while the previous attempt's failing pod is still in the pod
	// cache. The retried attempt must not be terminated by the old clock.
	workflowFake := client.NewWorkflowClientFake()
	pipelineFake := client.NewPipelineClientFake()

	stalePod := newWorkflowPod("failing-pod", "MY_NAME", "bad-image:latest", "ImagePullBackOff")
	stalePod.Namespace = "MY_NAMESPACE"
	podIndexer := newTestPodIndexer(stalePod)
	fakeExecInterface := &fakeExecutionInterface{}
	checker, clock := newTestChecker(podIndexer, &fakeExecutionClient{executionInterface: fakeExecInterface}, 5*time.Minute)

	saver := NewWorkflowSaver(workflowFake, pipelineFake, 100)
	saver.SetImagePullFailureChecker(checker)

	newWorkflow := func(phase workflowapi.WorkflowPhase, resourceVersion string, extraLabels, annotations map[string]string) util.ExecutionSpec {
		labels := map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"}
		for k, v := range extraLabels {
			labels[k] = v
		}
		return util.NewWorkflow(&workflowapi.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       "MY_NAMESPACE",
				Name:            "MY_NAME",
				UID:             testWorkflowUID("MY_NAME"),
				ResourceVersion: resourceVersion,
				Labels:          labels,
				Annotations:     annotations,
			},
			Status: workflowapi.WorkflowStatus{Phase: phase, FinishedAt: metav1.Now()},
		})
	}

	// Attempt 1 is running with a failing pod; the grace period starts.
	workflowFake.Put("MY_NAMESPACE", "MY_NAME", newWorkflow(workflowapi.WorkflowRunning, "100", nil, nil))
	require.NoError(t, saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20))
	assert.Len(t, checker.failureStart, 1)
	clock.advance(5 * time.Minute)

	// The live workflow finished and was already persisted by the API server;
	// the saver takes the persisted-final shortcut.
	workflowFake.Put("MY_NAMESPACE", "MY_NAME", newWorkflow(workflowapi.WorkflowFailed, "101",
		map[string]string{util.LabelKeyWorkflowPersistedFinalState: "true"}, nil))
	require.NoError(t, saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20))
	assert.Empty(t, checker.failureStart, "terminal workflow must drop tracking even on the shortcut path")

	// The run is retried in place while the old pod is still cached.
	workflowFake.Put("MY_NAMESPACE", "MY_NAME", newWorkflow(workflowapi.WorkflowRunning, "102", nil,
		map[string]string{util.AnnotationKeyRetryGeneration: "1"}))
	require.NoError(t, saver.Save("MY_KEY", "MY_NAMESPACE", "MY_NAME", 20))
	assert.Equal(t, 0, fakeExecInterface.patchCount, "the retried attempt must not inherit the previous attempt's failure clock")
	require.Len(t, checker.failureStart, 1)
	assert.Equal(t, "1", checker.failureStart["MY_NAMESPACE/MY_NAME"].retryGeneration)
}
