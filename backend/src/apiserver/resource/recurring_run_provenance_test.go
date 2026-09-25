// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"fmt"
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// Acknowledging an existing scheduled run must retain the same source-provenance
// checks as retrying it, including after returning from audit to enforcement.
func TestRecurringRunReplayUsesSelectedPipelineProvenance(t *testing.T) {
	for _, multiUser := range []bool{false, true} {
		for _, test := range []struct {
			name                        string
			deletedVersion, newerV2, v2 bool
		}{
			{name: "pinned V1 with unrelated V2 manifest"},
			{name: "deleted V1 pin with unrelated V2 manifest", deletedVersion: true},
			{name: "newer V2 cannot replace V1 pin", newerV2: true},
			{name: "genuine pinned V2", v2: true},
		} {
			t.Run(fmt.Sprintf("multiuser=%t/%s", multiUser, test.name), func(t *testing.T) {
				initEnvVars()
				configureSecurityModes(t, "enforce", "audit")
				viper.Set(common.MultiUserMode, multiUser)
				store := NewFakeClientManagerOrFatalV2()
				defer store.Close()
				manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
				manager.time = fixedRecurringTime{epoch: 200}
				ctx := multiUserContext()
				experiment, err := manager.CreateExperiment(&model.Experiment{Name: "replay-provenance", Namespace: "ns1"})
				require.NoError(t, err)
				workflow := testWorkflowWithoutStatus()
				workflow.Spec.Templates[0].Inputs.Parameters = []workflowapi.Parameter{{
					Name: "pod-spec-patch", Default: workflowapi.AnyStringPtr(`{"serviceAccountName":"helper-sa"}`),
				}}
				workflow.Spec.Templates[0].PodSpecPatch = "{{inputs.parameters.pod-spec-patch}}"
				manifest := workflow.ToStringForStore()
				if test.v2 {
					manifest = v2SpecHelloWorld
				}
				pipeline, err := manager.CreatePipeline(createPipeline("replay-source", "", experiment.Namespace))
				require.NoError(t, err)
				version, err := manager.CreatePipelineVersion(createPipelineVersion(
					pipeline.UUID, "selected-source", "", "", manifest, "", experiment.Namespace))
				require.NoError(t, err)
				job, err := manager.CreateJob(ctx, &model.Job{
					DisplayName: "replay-provenance", Namespace: experiment.Namespace, ExperimentId: experiment.UUID,
					Enabled: true, MaxConcurrency: 1,
					Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{
						PeriodicScheduleStartTimeInSec: util.Int64Pointer(100), IntervalSecond: util.Int64Pointer(10),
					}},
					PipelineSpec: model.PipelineSpec{
						PipelineVersionId: version.UUID,
						Parameters:        `[{"name":"param1","value":"world"}]`,
						RuntimeConfig:     model.RuntimeConfig{Parameters: `{"text":"world"}`},
					},
				})
				require.NoError(t, err)
				submit := func() (*model.Run, error) {
					run := &model.Run{
						DisplayName: "same-tick", RecurringRunId: job.UUID,
						Namespace: experiment.Namespace, ExperimentId: experiment.UUID, PipelineSpec: job.PipelineSpec,
					}
					if err := manager.PrepareRecurringRun(ctx, run); err != nil {
						return nil, err
					}
					return manager.CreateRun(ctx, run)
				}
				first, err := submit()
				require.NoError(t, err)
				require.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())
				if !test.v2 {
					// Simulate records created before unused caller-supplied V2
					// manifests were cleared when the selected source was V1.
					_, err = store.db.Exec(`UPDATE "run_details" SET "PipelineSpecManifest" = ? WHERE "UUID" = ?`, v2SpecHelloWorld, first.UUID)
					require.NoError(t, err)
				}
				if test.newerV2 {
					store.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal("123e4567-e89b-12d3-a456-426655440001", nil))
					manager.pipelineStore = store.PipelineStore()
					newer, err := manager.CreatePipelineVersion(createPipelineVersion(
						pipeline.UUID, "newer-v2", "", "", v2SpecHelloWorld, "", experiment.Namespace))
					require.NoError(t, err)
					require.NoError(t, manager.UpdatePipelineDefaultVersion(pipeline.UUID, newer.UUID))
				}
				if test.deletedVersion {
					require.NoError(t, manager.DeletePipelineVersion(version.UUID))
				}
				before, err := manager.GetRun(first.UUID)
				require.NoError(t, err)
				tickBefore, err := store.JobStore().GetRecurringRunState(job.UUID)
				require.NoError(t, err)
				for _, mode := range []string{"enforce", "audit"} {
					viper.Set(common.WorkflowIdentityMode, mode)
					replayed, err := submit()
					if mode == "enforce" && !test.v2 {
						require.ErrorContains(t, err, "podSpecPatch")
					} else {
						require.NoError(t, err)
						require.Equal(t, first.UUID, replayed.UUID)
					}
					after, err := manager.GetRun(first.UUID)
					require.NoError(t, err)
					require.Equal(t, before, after, "acknowledgement must not mutate the retained run")
					tickAfter, err := store.JobStore().GetRecurringRunState(job.UUID)
					require.NoError(t, err)
					require.Equal(t, tickBefore, tickAfter, "acknowledgement must not claim another tick")
					require.Equal(t, 1, store.ExecClientFake.GetWorkflowCount(), "acknowledgement must not create another execution")
				}
			})
		}
	}
}
