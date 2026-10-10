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
	"fmt"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Interleave a real workflow report after RetryRun has read the live Workflow
// but before it commits the adopted snapshot. The reporter uses its own guarded
// write and task finalization; no fake success or direct task mutation is used.
type reportBeforeRetryAcknowledgmentStore struct {
	storage.RunStoreInterface
	before func()
}

func (s *reportBeforeRetryAcknowledgmentStore) report() {
	if s.before != nil {
		before := s.before
		s.before = nil
		before()
	}
}

func (s *reportBeforeRetryAcknowledgmentStore) UpdateRunIfRuntimeManifestsUnchanged(run *model.Run, workflow, pipeline model.LargeText) (bool, error) {
	s.report()
	return s.RunStoreInterface.UpdateRunIfRuntimeManifestsUnchanged(run, workflow, pipeline)
}

func (s *reportBeforeRetryAcknowledgmentStore) UpdateRun(run *model.Run) error {
	s.report()
	return s.RunStoreInterface.UpdateRun(run)
}

type snapshotRetryWorkflowClient struct {
	util.ExecutionInterface
}

func (c *snapshotRetryWorkflowClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (util.ExecutionSpec, error) {
	workflow, err := c.ExecutionInterface.Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	return util.NewWorkflow(workflow.(*util.Workflow).DeepCopy()), nil
}

func TestRetryRun_AcknowledgesConcurrentWorkflowReport(t *testing.T) {
	for _, expiredClaim := range []bool{false, true} {
		for _, phase := range []workflowapi.WorkflowPhase{workflowapi.WorkflowRunning, workflowapi.WorkflowFailed, workflowapi.WorkflowSucceeded} {
			t.Run(fmt.Sprintf("expired=%t/%s", expiredClaim, phase), func(t *testing.T) {
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
				workflow.Status.Phase = phase
				if phase != workflowapi.WorkflowRunning {
					workflow.Status.FinishedAt = metav1.NewTime(time.Unix(500, 0))
				}
				workflowClient := client.NewWorkflowClientFake()
				_, err = workflowClient.Create(ctx, workflow, metav1.CreateOptions{})
				require.NoError(t, err)
				if expiredClaim {
					manager.execClient = &retryWorkflowExecClient{workflowClient: &snapshotRetryWorkflowClient{ExecutionInterface: workflowClient}}
				} else {
					manager.execClient = &retryWorkflowExecClient{workflowClient: &snapshotRetryWorkflowClient{ExecutionInterface: &persistentConflictWorkflowClient{FakeWorkflowClient: workflowClient}}}
				}
				var reported *model.Run
				manager.runStore = &reportBeforeRetryAcknowledgmentStore{RunStoreInterface: store.RunStore(), before: func() {
					live, err := manager.getWorkflowClient(workflow.ExecutionNamespace()).Get(ctx, workflow.ExecutionName(), metav1.GetOptions{})
					require.NoError(t, err)
					_, err = manager.ReportWorkflowResource(ctx, live)
					require.NoError(t, err)
					reported, err = store.RunStore().GetRun(run.UUID, true)
					require.NoError(t, err)
					require.EqualValues(t, 1, reported.RetryGeneration)
					require.Equal(t, model.RuntimeState(string(phase)).ToV2(), reported.State)
				}}
				require.NoError(t, manager.RetryRun(ctx, original.UUID), "an already reported retry must be acknowledged without inviting another generation")
				require.NotNil(t, reported)
				after, err := store.RunStore().GetRun(run.UUID, true)
				require.NoError(t, err)
				require.Equal(t, reported, after, "acknowledgment must not rewrite the reporter's state or history")
			})
		}
	}
}

func TestRetryRun_ConflictingTerminalAcknowledgmentDoesNotClaimAgain(t *testing.T) {
	for _, conflict := range []string{"new generation", "replacement workflow", "stale workflow generation"} {
		t.Run(conflict, func(t *testing.T) {
			store, manager, original := initWithOneTimeFailedRun(t)
			defer store.Close()
			ctx := context.Background()
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
			manager.execClient = &retryWorkflowExecClient{workflowClient: &snapshotRetryWorkflowClient{ExecutionInterface: &persistentConflictWorkflowClient{FakeWorkflowClient: workflowClient}}}
			var concurrent *model.Run
			manager.runStore = &reportBeforeRetryAcknowledgmentStore{RunStoreInterface: store.RunStore(), before: func() {
				live, err := manager.getWorkflowClient(workflow.ExecutionNamespace()).Get(ctx, workflow.ExecutionName(), metav1.GetOptions{})
				require.NoError(t, err)
				_, err = manager.ReportWorkflowResource(ctx, live)
				require.NoError(t, err)
				if conflict == "new generation" {
					_, _, _, generation, err := store.RunStore().ClaimRunForRetry(run.UUID, false)
					require.NoError(t, err)
					require.EqualValues(t, 2, generation)
				} else {
					replacement, err := store.RunStore().GetRun(run.UUID, true)
					require.NoError(t, err)
					if conflict == "replacement workflow" {
						live.ExecutionObjectMeta().UID = "replacement-workflow-uid"
					} else {
						live.SetAnnotations(util.AnnotationKeyRetryGeneration, "0")
					}
					replacement.WorkflowRuntimeManifest = model.LargeText(live.ToStringForStore())
					require.NoError(t, store.RunStore().UpdateRun(replacement))
				}
				concurrent, err = store.RunStore().GetRun(run.UUID, true)
				require.NoError(t, err)
			}}
			err = manager.RetryRun(ctx, original.UUID)
			require.Error(t, err)
			require.True(t, util.IsUserErrorCodeMatch(err, codes.FailedPrecondition), "got %v", err)
			require.NotContains(t, err.Error(), "retry the request")
			after, err := store.RunStore().GetRun(run.UUID, true)
			require.NoError(t, err)
			require.Equal(t, concurrent, after, "conflicting acknowledgment must not overwrite or claim another generation")
		})
	}
}

type concurrentRetryWorkflowClient struct {
	util.ExecutionInterface
	responsePhase workflowapi.WorkflowPhase
	reportedPhase workflowapi.WorkflowPhase
}

func (c *concurrentRetryWorkflowClient) Update(ctx context.Context, spec util.ExecutionSpec, opts metav1.UpdateOptions) (util.ExecutionSpec, error) {
	started, err := c.ExecutionInterface.Update(ctx, spec, opts)
	if err != nil {
		return nil, err
	}
	response := util.NewWorkflow(started.(*util.Workflow).DeepCopy())
	response.Status.Phase = c.responsePhase
	if c.responsePhase == workflowapi.WorkflowFailed {
		response.Status.FinishedAt = metav1.NewTime(time.Unix(500, 0))
	}
	reported := util.NewWorkflow(response.DeepCopy())
	reported.Status.Phase = c.reportedPhase
	reported.Status.FinishedAt = metav1.Time{}
	if c.reportedPhase == workflowapi.WorkflowFailed {
		reported.Status.FinishedAt = metav1.NewTime(time.Unix(500, 0))
	}
	if _, err := c.ExecutionInterface.Update(ctx, reported, opts); err != nil {
		return nil, err
	}
	return response, nil
}

func TestRetryRun_ConcurrentReportPreservedAfterRetryResponse(t *testing.T) {
	for _, test := range []struct {
		name                         string
		responsePhase, reportedPhase workflowapi.WorkflowPhase
	}{
		{name: "terminal report beats running response", responsePhase: workflowapi.WorkflowRunning, reportedPhase: workflowapi.WorkflowFailed},
		{name: "running report acknowledges running response", responsePhase: workflowapi.WorkflowRunning, reportedPhase: workflowapi.WorkflowRunning},
		{name: "delayed running report acknowledges terminal response", responsePhase: workflowapi.WorkflowFailed, reportedPhase: workflowapi.WorkflowRunning},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, manager, original := initWithOneTimeFailedRun(t)
			defer store.Close()
			ctx := context.Background()
			manager.execClient = &retryWorkflowExecClient{workflowClient: &concurrentRetryWorkflowClient{
				ExecutionInterface: &snapshotRetryWorkflowClient{ExecutionInterface: store.ExecClient().Execution(original.Namespace)},
				responsePhase:      test.responsePhase,
				reportedPhase:      test.reportedPhase,
			}}
			var reported *model.Run
			manager.runStore = &reportBeforeRetryAcknowledgmentStore{RunStoreInterface: store.RunStore(), before: func() {
				live, err := manager.getWorkflowClient(original.Namespace).Get(ctx, original.K8SName, metav1.GetOptions{})
				require.NoError(t, err)
				_, err = manager.ReportWorkflowResource(ctx, live)
				require.NoError(t, err)
				reported, err = store.RunStore().GetRun(original.UUID, true)
				require.NoError(t, err)
				require.Equal(t, model.RuntimeState(string(test.reportedPhase)).ToV2(), reported.State)
			}}
			require.NoError(t, manager.RetryRun(ctx, original.UUID))
			require.NotNil(t, reported)
			after, err := store.RunStore().GetRun(original.UUID, true)
			require.NoError(t, err)
			require.Equal(t, reported, after, "acknowledgment must preserve the concurrent report")
		})
	}
}
