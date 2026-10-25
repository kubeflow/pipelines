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
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestRecurringRunTaskCredentials(t *testing.T) {
	for _, source := range []string{"inline", "pinned", "follow-latest"} {
		t.Run(source, func(t *testing.T) {
			initEnvVars()
			old := viper.Get(common.MultiUserMode)
			viper.Set(common.MultiUserMode, true)
			t.Cleanup(func() { viper.Set(common.MultiUserMode, old) })
			store := NewFakeClientManagerOrFatalV2()
			defer store.Close()
			manager := NewResourceManager(store, &ResourceManagerOptions{})
			manager.time = fixedRecurringTime{epoch: 200}
			ctx := multiUserContext()
			exp, err := manager.CreateExperiment(&model.Experiment{Name: "scheduled-credentials", Namespace: "ns1"})
			require.NoError(t, err)
			spec := model.PipelineSpec{PipelineSpecManifest: model.LargeText(v2SpecHelloWorld), RuntimeConfig: model.RuntimeConfig{Parameters: `{"text":"scheduled"}`}}
			if source != "inline" {
				pipeline, err := manager.CreatePipeline(createPipeline("scheduled-credentials", "", "ns1"))
				require.NoError(t, err)
				version, err := manager.CreatePipelineVersion(createPipelineVersion(pipeline.UUID, "v1", "v1", "", v2SpecHelloWorld, "", "ns1"))
				require.NoError(t, err)
				spec.PipelineSpecManifest = ""
				if source == "pinned" {
					spec.PipelineVersionId = version.UUID
				} else {
					spec.PipelineId = pipeline.UUID
				}
			}
			job, err := manager.CreateJob(ctx, &model.Job{DisplayName: "existing-schedule", Namespace: "ns1", ExperimentId: exp.UUID, Enabled: true, MaxConcurrency: 1, PipelineSpec: spec, Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{PeriodicScheduleStartTimeInSec: util.Int64Pointer(100), IntervalSecond: util.Int64Pointer(10)}}})
			require.NoError(t, err)
			request := &model.Run{DisplayName: "tick", RecurringRunId: job.UUID}
			require.NoError(t, manager.PrepareRecurringRun(ctx, request))
			run, err := manager.CreateRun(ctx, request)
			require.NoError(t, err)
			execution, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(run.PipelineRuntimeManifest))
			require.NoError(t, err)
			require.NoError(t, execution.(*util.Workflow).ValidateTaskRuntimeCredentials(common.GetTaskTokenReviewAudience()))
			require.Equal(t, job.Namespace, run.Namespace)
			require.Equal(t, job.ServiceAccount, run.ServiceAccount)
			require.Equal(t, job.UUID, run.RecurringRunId)
			if source == "pinned" {
				require.Equal(t, spec.PipelineVersionId, run.PipelineVersionId)
			}
			saved, err := manager.GetJob(job.UUID)
			require.NoError(t, err)
			require.Equal(t, job.PipelineSpec, saved.PipelineSpec)
		})
	}
}
