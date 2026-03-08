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
	"fmt"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// fakeImagePullFailureChecker is a test double for ImagePullFailureCheckerInterface.
type fakeImagePullFailureChecker struct {
	called        bool
	namespace     string
	workflowName  string
	errorToReturn error
}

func (f *fakeImagePullFailureChecker) CheckAndTerminate(namespace string, workflowName string) error {
	f.called = true
	f.namespace = namespace
	f.workflowName = workflowName
	return f.errorToReturn
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

	assert.NoError(t, err)
	assert.True(t, checker.called, "Checker should be called for running workflow")
	assert.Equal(t, "MY_NAMESPACE", checker.namespace)
	assert.Equal(t, "MY_NAME", checker.workflowName)
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

	assert.NoError(t, err)
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

	assert.NoError(t, err)
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

	assert.NoError(t, err)
	assert.False(t, checker.called, "Checker should NOT be called for failed workflow")
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
	assert.NoError(t, err)
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

	assert.NoError(t, err)
}
