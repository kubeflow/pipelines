// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"context"
	"strconv"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func createUnfinishedResourceDriver(t *testing.T, tasks storage.TaskStoreInterface, run *model.Run) *model.Task {
	t.Helper()
	metadata, err := model.ProtoMessageToJSONData(&api.PipelineTask_StatusMetadata{
		Message: "driver stopped before retry exhaustion",
	})
	require.NoError(t, err)
	attempt := int64(0)
	task, err := tasks.CreateTask(&model.Task{
		DriverRetryGeneration: &run.RetryGeneration, DriverRetryAttempt: &attempt, DriverClaim: true,
		DriverWriteAuthority: &model.DriverTaskAuthority{Generation: run.RetryGeneration},
		Namespace:            "ns1", RunUUID: run.UUID, Name: "retry-driver", ScopePath: "root.retry-driver",
		Type: model.TaskType(api.PipelineTask_RUNTIME), State: model.TaskStatus(api.PipelineTask_RUNNING),
		TypeAttrs: model.JSONData{}, Pods: model.JSONSlice{}, StatusMetadata: metadata,
	})
	require.NoError(t, err)
	return task
}

type taskOnRetryConflictWorkflowClient struct {
	*persistentConflictWorkflowClient
	createTask func()
}

func (c *taskOnRetryConflictWorkflowClient) Update(ctx context.Context, spec util.ExecutionSpec, opts metav1.UpdateOptions) (util.ExecutionSpec, error) {
	if c.createTask != nil {
		c.createTask()
		c.createTask = nil
	}
	return c.persistentConflictWorkflowClient.Update(ctx, spec, opts)
}

func TestRetryRun_TerminalAdoptionFinalizesDriverTasks(t *testing.T) {
	for _, expiredClaim := range []bool{false, true} {
		t.Run(strconv.FormatBool(expiredClaim), func(t *testing.T) {
			store, manager, original := initWithOneTimeFailedRun(t)
			defer store.Close()
			ctx := context.Background()
			if expiredClaim {
				_, _, _, generation, err := store.RunStore().ClaimRunForRetry(original.UUID, false)
				require.NoError(t, err)
				require.EqualValues(t, 1, generation)
				_, err = store.DB().Exec("UPDATE run_details SET RetryClaimedAtInSec = 0 WHERE UUID = ?", original.UUID)
				require.NoError(t, err)
			}
			run, err := manager.GetRun(original.UUID)
			require.NoError(t, err)
			spec, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(run.WorkflowRuntimeManifest))
			require.NoError(t, err)
			require.NoError(t, spec.Decompress())
			retried, _, err := spec.GenerateRetryExecution()
			require.NoError(t, err)
			retried.SetAnnotations(util.AnnotationKeyRetryGeneration, "1")
			workflow := retried.(*util.Workflow)
			workflow.Status.Phase = workflowapi.WorkflowFailed
			workflow.Status.FinishedAt = metav1.NewTime(time.Unix(500, 0))
			workflowClient := client.NewWorkflowClientFake()
			_, err = workflowClient.Create(ctx, workflow, metav1.CreateOptions{})
			require.NoError(t, err)
			var task *model.Task
			if expiredClaim {
				task = createUnfinishedResourceDriver(t, store.TaskStore(), run)
				manager.execClient = &retryWorkflowExecClient{workflowClient: workflowClient}
			} else {
				conflicts := &taskOnRetryConflictWorkflowClient{
					persistentConflictWorkflowClient: &persistentConflictWorkflowClient{FakeWorkflowClient: workflowClient},
					createTask: func() {
						claimed, err := store.RunStore().GetRun(run.UUID, true)
						require.NoError(t, err)
						task = createUnfinishedResourceDriver(t, store.TaskStore(), claimed)
					},
				}
				manager.execClient = &retryWorkflowExecClient{workflowClient: conflicts}
			}
			require.NoError(t, manager.RetryRun(ctx, original.UUID))
			require.NotNil(t, task)
			after, err := store.TaskStore().GetTask(task.UUID)
			require.NoError(t, err)
			require.Equal(t, model.TaskStatus(api.PipelineTask_FAILED), after.State)
			require.EqualValues(t, 500, after.FinishedInSec)
			require.Equal(t, task.StatusMetadata, after.StatusMetadata)
		})
	}
}

func TestReportWorkflowResource_ConcurrentRecurringInsertFinalizesDriverTasks(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	ctx := context.Background()
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "raced-recurring-workflow", Namespace: job.Namespace, UID: "raced-recurring-run",
			Labels:          map[string]string{util.LabelKeyWorkflowRunId: "raced-recurring-run", util.LabelKeyWorkflowPersistedFinalState: "true"},
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubeflow.org/v1beta1", Kind: "ScheduledWorkflow", Name: job.K8SName, UID: types.UID(job.UUID)}},
		},
		Status: workflowapi.WorkflowStatus{Phase: workflowapi.WorkflowFailed, FinishedAt: metav1.NewTime(time.Unix(500, 0))},
	})
	syncWorkflowReportWithFakeCluster(t, store, workflow)
	run, err := store.RunStore().CreateRun(&model.Run{
		UUID: "raced-recurring-run", DisplayName: workflow.ExecutionName(), K8SName: workflow.ExecutionName(),
		ExperimentId: job.ExperimentId, RecurringRunId: job.UUID, Namespace: job.Namespace, PipelineSpec: job.PipelineSpec,
		RunDetails: model.RunDetails{State: model.RuntimeStateRunning, WorkflowRuntimeManifest: model.LargeText(workflow.ToStringForStore())},
	})
	require.NoError(t, err)
	task := createUnfinishedResourceDriver(t, store.TaskStore(), run)
	manager.runStore = &duplicateRecurringRunStore{RunStoreInterface: store.RunStore(), firstGet: true, existingRun: run}
	_, err = manager.ReportWorkflowResource(ctx, workflow)
	require.NoError(t, err)
	after, err := store.TaskStore().GetTask(task.UUID)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(api.PipelineTask_FAILED), after.State)
	require.EqualValues(t, 500, after.FinishedInSec)
	_, err = store.ExecClient().Execution(job.Namespace).Get(ctx, workflow.ExecutionName(), metav1.GetOptions{})
	require.True(t, util.IsNotFound(err), "the completed Workflow is deleted only after task finalization")
}
