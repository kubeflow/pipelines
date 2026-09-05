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

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfutil "github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRetryRun_ReportedRecurringRunPodSpecPatch(t *testing.T) {
	for _, test := range []struct {
		name           string
		v2             bool
		v1PipelineSpec bool
	}{
		{name: "compiled v2 workflow", v2: true},
		{name: "raw v1 workflow cannot impersonate compiler patch"},
		{name: "v1 pipeline spec cannot impersonate compiler patch", v1PipelineSpec: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var store *FakeClientManager
			var manager *ResourceManager
			var job *model.Job
			switch {
			case test.v2:
				store, manager, job = initWithJobV2(t)
			case test.v1PipelineSpec:
				var experiment *model.Experiment
				store, manager, experiment = initWithExperiment(t)
				var err error
				job, err = manager.CreateJob(context.Background(), &model.Job{
					DisplayName: "j1", Enabled: true, ExperimentId: experiment.UUID,
					PipelineSpec: model.PipelineSpec{PipelineSpecManifest: model.LargeText(testWorkflow.ToStringForStore())},
				})
				require.NoError(t, err)
			default:
				store, manager, job = initWithJob(t)
			}
			defer store.Close()
			ctx := context.Background()
			schedule, err := store.SwfClient().ScheduledWorkflow(job.Namespace).Get(ctx, job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			executionSpec, err := swfutil.NewScheduledWorkflow(schedule).NewWorkflow(11, 11)
			require.NoError(t, err)
			workflow := executionSpec.(*util.Workflow)
			workflow.Namespace = job.Namespace
			if !test.v2 {
				// Existing V1 workflows must not gain the compiler-only exemption on retry.
				workflow.Spec.Templates[0].PodSpecPatch = "{{inputs.parameters.pod-spec-patch}}"
			}
			require.Contains(t, workflow.ToStringForStore(), "{{inputs.parameters.pod-spec-patch}}")
			workflow.Status.Phase = v1alpha1.WorkflowFailed
			syncWorkflowReportWithFakeCluster(t, store, workflow)
			_, err = manager.ReportWorkflowResource(ctx, workflow)
			require.NoError(t, err)

			runID := workflow.Labels[util.LabelKeyWorkflowRunId]
			run, err := manager.GetRun(runID)
			require.NoError(t, err)
			require.Equal(t, job.UUID, run.RecurringRunId)
			require.NotEmpty(t, run.WorkflowSpecManifest)
			require.Equal(t, model.RuntimeStateFailed, run.State)
			if test.v2 || test.v1PipelineSpec {
				require.NotEmpty(t, run.PipelineSpecManifest)
			}

			err = manager.RetryRun(ctx, runID)
			if test.v2 {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "podSpecPatch")
			}
			run, err = manager.GetRun(runID)
			require.NoError(t, err)
			liveWorkflow, err := store.ExecClient().Execution(job.Namespace).Get(ctx, workflow.Name, metav1.GetOptions{})
			require.NoError(t, err)
			if test.v2 {
				assert.Equal(t, model.RuntimeStateRunning, run.State)
				assert.Equal(t, string(v1alpha1.WorkflowRunning), string(liveWorkflow.ExecutionStatus().Condition()))
			} else {
				assert.Equal(t, model.RuntimeStateFailed, run.State)
				assert.Zero(t, run.RetryGeneration)
				assert.Equal(t, string(v1alpha1.WorkflowFailed), string(liveWorkflow.ExecutionStatus().Condition()))
			}
		})
	}
}
