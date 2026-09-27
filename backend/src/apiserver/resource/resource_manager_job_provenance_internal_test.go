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

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestChangeJobMode_UsesSelectedPipelineProvenance(t *testing.T) {
	for _, test := range []struct {
		name                                                                       string
		v2, pinned, bothFields, deletedVersion, missingSource, missingPin, literal bool
		pipelineOnly, invalidSource, wantAllowed                                   bool
	}{
		{name: "normalized V2 job", v2: true, wantAllowed: true},
		{name: "legacy inline V2 job with both fields", v2: true, bothFields: true, wantAllowed: true},
		{name: "legacy pinned V2 job with both fields", v2: true, pinned: true, bothFields: true, wantAllowed: true},
		{name: "referenced V1 job with unrelated V2 manifest", pinned: true, bothFields: true},
		{name: "deleted V2 pin", v2: true, pinned: true, deletedVersion: true},
		{name: "missing pinned V2 source", v2: true, pinned: true, missingSource: true},
		{name: "missing V2 pin with retained reference", v2: true, pinned: true, missingPin: true},
		{name: "literal workflow after legacy source deletion", pinned: true, bothFields: true, deletedVersion: true, literal: true, wantAllowed: true},
		{name: "raw V1 pipeline field is not compiler provenance", pipelineOnly: true},
		{name: "invalid pipeline field is not compiler provenance", pipelineOnly: true, invalidSource: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			configureWorkflowIdentityAuditTest(t, true)
			store, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)
			defer store.Close()
			ctx := multiUserContext()
			workflow := testWorkflowWithoutStatus()
			workflow.Spec.Templates[0].Inputs.Parameters = []workflowapi.Parameter{{
				Name: "pod-spec-patch", Default: workflowapi.AnyStringPtr(`{"serviceAccountName":"nested-sa"}`),
			}}
			workflow.Spec.Templates[0].PodSpecPatch = "{{inputs.parameters.pod-spec-patch}}"
			if test.literal {
				workflow.Spec.Templates[0].PodSpecPatch = `{"serviceAccountName":"pipeline-runner"}`
			}
			manifest := workflow.ToStringForStore()
			if test.v2 {
				manifest = v2SpecHelloWorld
			}
			pipelineSpec := model.PipelineSpec{
				PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
				Parameters:           `[{"name":"param1","value":"world"}]`,
				RuntimeConfig:        model.RuntimeConfig{Parameters: `{"text":"world"}`},
			}
			if test.pinned {
				pipeline, err := manager.CreatePipeline(createPipeline("job-source", "", experiment.Namespace))
				require.NoError(t, err)
				version, err := manager.CreatePipelineVersion(createPipelineVersion(
					pipeline.UUID, "selected-version", "", "", v2SpecHelloWorld, "", experiment.Namespace))
				require.NoError(t, err)
				pipelineSpec.PipelineVersionId = version.UUID
			}
			job, err := manager.CreateJob(ctx, &model.Job{
				DisplayName: "source-job", Enabled: true, ExperimentId: experiment.UUID, PipelineSpec: pipelineSpec,
			})
			require.NoError(t, err)
			require.NoError(t, manager.ChangeJobMode(ctx, job.UUID, false))
			if !test.v2 && test.pinned {
				_, err = store.db.Exec(`UPDATE "pipeline_versions" SET "PipelineSpec" = ? WHERE "UUID" = ?`, manifest, job.PipelineVersionId)
				require.NoError(t, err)
			}
			if test.bothFields {
				// Historical rows could retain both the selected and unused source fields.
				_, err = store.db.Exec(`UPDATE "jobs" SET "PipelineSpecManifest" = ?, "WorkflowSpecManifest" = ? WHERE "UUID" = ?`, v2SpecHelloWorld, manifest, job.UUID)
				require.NoError(t, err)
			}
			if test.pipelineOnly {
				storedManifest := manifest
				if test.invalidSource {
					storedManifest = "not a pipeline"
				}
				_, err = store.db.Exec(`UPDATE "jobs" SET "PipelineSpecManifest" = ?, "WorkflowSpecManifest" = '' WHERE "UUID" = ?`, storedManifest, job.UUID)
				require.NoError(t, err)
			}
			if test.deletedVersion {
				require.NoError(t, manager.DeletePipelineVersion(job.PipelineVersionId))
			}
			if test.missingSource {
				_, err = store.db.Exec(`UPDATE "pipeline_versions" SET "PipelineSpec" = '' WHERE "UUID" = ?`, job.PipelineVersionId)
				require.NoError(t, err)
			}
			if test.missingPin {
				_, err = store.db.Exec(`UPDATE "jobs" SET "PipelineVersionId" = '' WHERE "UUID" = ?`, job.UUID)
				require.NoError(t, err)
				_, err = store.db.Exec(`DELETE FROM "resource_references" WHERE "ResourceUUID" = ? AND "ResourceType" = ? AND "ReferenceType" = ?`, job.UUID, model.JobResourceType, model.PipelineVersionResourceType)
				require.NoError(t, err)
			}
			before, err := manager.GetJob(job.UUID)
			require.NoError(t, err)
			require.False(t, before.Enabled)
			if test.missingPin {
				require.Empty(t, before.PipelineVersionId)
				require.NotEmpty(t, before.PipelineId)
			}
			beforeSchedule, err := store.SwfClient().ScheduledWorkflow(job.Namespace).Get(ctx, job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, job.UUID, string(beforeSchedule.UID))
			beforeSchedule = beforeSchedule.DeepCopy()
			executionSpec, err := util.ScheduleSpecToExecutionSpec(util.ArgoWorkflow, beforeSchedule.Spec.Workflow)
			require.NoError(t, err)
			if test.literal {
				workflow := executionSpec.(*util.Workflow)
				for i := range workflow.Spec.Templates {
					workflow.Spec.Templates[i].PodSpecPatch = ""
				}
				beforeSchedule.Spec.Workflow.Spec = workflow.ToStringForStore()
				_, err = store.SwfClient().ScheduledWorkflow(job.Namespace).Update(ctx, beforeSchedule)
				require.NoError(t, err)
			} else {
				require.Contains(t, executionSpec.ToStringForStore(), "{{inputs.parameters.pod-spec-patch}}")
			}
			patchCounter := &patchCountingSwfClient{SwfClientInterface: manager.swfClient}
			manager.swfClient = patchCounter

			viper.Set(common.WorkflowIdentityMode, "enforce")
			err = manager.ChangeJobMode(ctx, job.UUID, true)
			switch {
			case test.wantAllowed:
				require.NoError(t, err)
			case test.invalidSource:
				require.Error(t, err)
			case test.v2:
				require.ErrorContains(t, err, "podSpecPatch")
			default:
				require.ErrorContains(t, err, "legacy Argo Workflow pipelines are no longer supported")
			}
			after, err := manager.GetJob(job.UUID)
			require.NoError(t, err)
			afterSchedule, err := store.SwfClient().ScheduledWorkflow(job.Namespace).Get(ctx, job.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			if test.wantAllowed {
				assert.True(t, after.Enabled)
				assert.True(t, afterSchedule.Spec.Enabled)
				assert.Equal(t, 1, patchCounter.patchCalls)
			} else {
				assert.Zero(t, patchCounter.patchCalls, "authorization must precede ScheduledWorkflow changes")
				assert.Equal(t, before, after, "authorization must precede persisted job changes")
				assert.Equal(t, beforeSchedule, afterSchedule)
			}
		})
	}
}
