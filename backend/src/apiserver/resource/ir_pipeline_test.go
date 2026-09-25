// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"context"
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

func TestCreateRun_IRSourcesPersistCompiledExecution(t *testing.T) {
	for _, source := range []string{"inline", "pipeline", "version", "pipeline-and-version"} {
		t.Run(source, func(t *testing.T) {
			clients, manager, experiment, pipeline, version := initWithExperimentAndPipeline(t)
			defer clients.Close()
			spec := model.PipelineSpec{RuntimeConfig: model.RuntimeConfig{Parameters: `{"text":"hello"}`}}
			switch source {
			case "inline":
				spec.PipelineSpecManifest = model.LargeText(v2SpecHelloWorld)
			case "pipeline":
				spec.PipelineId = pipeline.UUID
			case "version":
				spec.PipelineVersionId = version.UUID
			case "pipeline-and-version":
				spec.PipelineId, spec.PipelineVersionId = pipeline.UUID, version.UUID
			}
			run, err := manager.CreateRun(context.Background(), &model.Run{DisplayName: "IR run", ExperimentId: experiment.UUID, PipelineSpec: spec})
			require.NoError(t, err)
			stored, err := manager.GetRun(run.UUID)
			require.NoError(t, err)
			require.YAMLEq(t, v2SpecHelloWorld, string(stored.PipelineSpecManifest))
			require.Empty(t, stored.WorkflowSpecManifest)
			require.JSONEq(t, `{"text":"hello"}`, string(stored.RuntimeConfig.Parameters))
			require.Equal(t, model.RuntimeStatePending, stored.State)
			require.Equal(t, experiment.UUID, stored.ExperimentId)
			require.Equal(t, "ns1", stored.Namespace)
			execution, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(stored.PipelineRuntimeManifest))
			require.NoError(t, err)
			require.Equal(t, stored.K8SName, execution.ExecutionName())
			require.Equal(t, stored.UUID, execution.ExecutionObjectMeta().Labels[util.LabelKeyWorkflowRunId])
			require.Equal(t, "pipeline-runner", execution.ServiceAccount())
			require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())
		})
	}
}

func TestCreateRunAndRecurringRun_RejectLegacyTemplate(t *testing.T) {
	clients, manager, experiment := initWithExperiment(t)
	defer clients.Close()
	for _, spec := range []model.PipelineSpec{
		{PipelineSpecManifest: model.LargeText(testWorkflow.ToStringForStore())},
		{WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore())},
	} {
		_, err := manager.CreateRun(context.Background(), &model.Run{DisplayName: "legacy", ExperimentId: experiment.UUID, PipelineSpec: spec})
		require.Error(t, err)
		_, err = manager.CreateJob(context.Background(), &model.Job{DisplayName: "legacy", ExperimentId: experiment.UUID, PipelineSpec: spec})
		require.Error(t, err)
	}
	require.Zero(t, clients.ExecClientFake.GetWorkflowCount())
}

func TestRetryRun_CompiledIRSurvivesPipelineVersionDeletion(t *testing.T) {
	clients, manager, experiment, pipeline, version := initWithExperimentAndPipeline(t)
	defer clients.Close()
	run, err := manager.CreateRun(context.Background(), &model.Run{DisplayName: "IR retry", ExperimentId: experiment.UUID,
		PipelineSpec: model.PipelineSpec{PipelineId: pipeline.UUID, PipelineVersionId: version.UUID, RuntimeConfig: model.RuntimeConfig{Parameters: `{"text":"retry"}`}},
	})
	require.NoError(t, err)
	execution, err := clients.ExecClient().Execution(run.Namespace).Get(context.Background(), run.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	workflow := execution.(*util.Workflow)
	workflow.Status.Phase = workflowapi.WorkflowFailed
	syncWorkflowReportWithFakeCluster(t, clients, workflow)
	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err)
	require.NoError(t, manager.DeletePipelineVersion(version.UUID))
	_, err = manager.GetPipelineVersion(version.UUID)
	require.Error(t, err)
	// A deleted source cannot establish provenance for compiler-only dynamic
	// patches. Fail closed, without changing the failed run or claiming a retry.
	err = manager.RetryRun(context.Background(), run.UUID)
	require.ErrorContains(t, err, "podSpecPatch")
	stored, err := manager.GetRun(run.UUID)
	require.NoError(t, err)
	require.Equal(t, model.RuntimeStateFailed, stored.State)
	require.Zero(t, stored.RetryGeneration)

	// A workflow with only literal patches remains retryable after deletion.
	for i := range workflow.Spec.Templates {
		workflow.Spec.Templates[i].PodSpecPatch = ""
	}
	syncWorkflowReportWithFakeCluster(t, clients, workflow)
	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err)
	require.NoError(t, manager.RetryRun(context.Background(), run.UUID))
	stored, err = manager.GetRun(run.UUID)
	require.NoError(t, err)
	require.Equal(t, model.RuntimeStateRunning, stored.State)
	require.Equal(t, int64(1), stored.RetryGeneration)
}

func TestRetryRun_RejectsLegacyBeforeClaimOrExecution(t *testing.T) {
	for _, oldMarker := range []bool{false, true} {
		clients, manager, experiment := initWithExperiment(t)
		workflow := util.NewWorkflow(&workflowapi.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "legacy", Namespace: "ns1", UID: "legacy-uid", Labels: map[string]string{util.LabelKeyWorkflowRunId: "legacy-run"}},
			Status:     workflowapi.WorkflowStatus{Phase: workflowapi.WorkflowFailed},
		})
		if oldMarker {
			workflow.Annotations = map[string]string{"pipelines.kubeflow.org/v2_pipeline": "true"}
		}
		run, err := clients.RunStore().CreateRun(&model.Run{UUID: "legacy-run", DisplayName: "legacy", K8SName: "legacy", Namespace: "ns1", ExperimentId: experiment.UUID,
			RunDetails: model.RunDetails{State: model.RuntimeStateFailed, WorkflowRuntimeManifest: model.LargeText(workflow.ToStringForStore())},
		})
		require.NoError(t, err)
		err = manager.RetryRun(context.Background(), run.UUID)
		require.Error(t, err)
		require.True(t, util.IsUserErrorCodeMatch(err, codes.InvalidArgument))
		require.Contains(t, err.Error(), "create a new run from pipeline IR")
		stored, err := manager.GetRun(run.UUID)
		require.NoError(t, err)
		require.Zero(t, stored.RetryGeneration)
		require.Equal(t, model.RuntimeStateFailed, stored.State)
		require.Zero(t, clients.ExecClientFake.GetWorkflowCount())
		require.NoError(t, clients.Close())
	}
}
