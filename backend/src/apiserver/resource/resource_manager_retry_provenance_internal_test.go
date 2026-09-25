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
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
)

type provenancePodDeletionCounter struct {
	client.KubernetesCoreInterface
	deletes int
}

func (c *provenancePodDeletionCounter) PodClient(namespace string) coreclient.PodInterface {
	return &provenanceCountingPodClient{PodInterface: c.KubernetesCoreInterface.PodClient(namespace), deletes: &c.deletes}
}

type provenanceCountingPodClient struct {
	coreclient.PodInterface
	deletes *int
}

func (c *provenanceCountingPodClient) Delete(ctx context.Context, name string, options metav1.DeleteOptions) error {
	*c.deletes++
	return c.PodInterface.Delete(ctx, name, options)
}

func TestRetryRun_UsesSelectedPipelineProvenance(t *testing.T) {
	for _, test := range []struct {
		name                                                                    string
		legacy, deletedVersion, missingSource, missingPin, newerV2, literal, v2 bool
	}{
		{name: "new V1 run clears unused V2 manifest"},
		{name: "legacy V1 run retains unrelated V2 manifest", legacy: true},
		{name: "latest V2 version cannot replace V1 pin", legacy: true, newerV2: true},
		{name: "deleted V1 pin cannot use unrelated V2 manifest", legacy: true, deletedVersion: true},
		{name: "missing pinned source cannot use newer V2 object", legacy: true, missingSource: true, newerV2: true},
		{name: "missing version pin cannot use latest V2", legacy: true, missingPin: true, newerV2: true},
		{name: "literal V1 patch can retry after pin deletion", legacy: true, deletedVersion: true, literal: true},
		{name: "genuine pinned V2 workflow can retry", v2: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			configureWorkflowIdentityAuditTest(t, true)
			store, manager, experiment := initWithExperimentAndUnauthorizedSAR(t)
			defer store.Close()
			ctx := multiUserContext()
			workflow := testWorkflowWithoutStatus()
			workflow.Spec.Templates[0].Inputs.Parameters = []workflowapi.Parameter{{
				Name:    "pod-spec-patch",
				Default: workflowapi.AnyStringPtr(`{"serviceAccountName":"nested-sa"}`),
			}}
			workflow.Spec.Templates[0].PodSpecPatch = "{{inputs.parameters.pod-spec-patch}}"
			if test.literal {
				workflow.Spec.Templates[0].PodSpecPatch = `{"serviceAccountName":"pipeline-runner"}`
			}
			selectedManifest := workflow.ToStringForStore()
			if test.v2 {
				selectedManifest = v2SpecHelloWorld
			}
			pipeline, err := manager.CreatePipeline(createPipeline("raw-v1", "", experiment.Namespace))
			require.NoError(t, err)
			version, err := manager.CreatePipelineVersion(createPipelineVersion(
				pipeline.UUID, "selected-version", "", "", selectedManifest, "", experiment.Namespace))
			require.NoError(t, err)

			run, err := manager.CreateRun(ctx, &model.Run{
				DisplayName:  "raw-v1-run",
				ExperimentId: experiment.UUID,
				PipelineSpec: model.PipelineSpec{
					PipelineVersionId:    version.UUID,
					PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
					WorkflowSpecManifest: model.LargeText(workflow.ToStringForStore()),
					Parameters:           `[{"name":"param1","value":"world"}]`,
					RuntimeConfig:        model.RuntimeConfig{Parameters: `{"text":"world"}`},
				},
			})
			require.NoError(t, err)
			if test.v2 {
				assert.Empty(t, run.WorkflowSpecManifest)
				require.NotEmpty(t, run.PipelineSpecManifest)
			} else {
				assert.Empty(t, run.PipelineSpecManifest, "the unrelated supplied V2 spec must not become retry provenance")
				selectedWorkflow, err := util.NewWorkflowFromBytes([]byte(run.WorkflowSpecManifest))
				require.NoError(t, err)
				require.Equal(t, workflow.Spec.Templates[0].PodSpecPatch, selectedWorkflow.Spec.Templates[0].PodSpecPatch)
			}
			require.Equal(t, version.UUID, run.PipelineVersionId)

			live, err := store.ExecClient().Execution(run.Namespace).Get(ctx, run.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			failed := util.NewWorkflow(live.(*util.Workflow).DeepCopy())
			failed.Status.Phase = workflowapi.WorkflowFailed
			failed.Status.Nodes = workflowapi.Nodes{"failed-pod": {
				ID: "failed-pod", Name: "failed-pod", Type: workflowapi.NodeTypePod, Phase: workflowapi.NodeFailed,
			}}
			syncWorkflowReportWithFakeCluster(t, store, failed)
			_, err = manager.ReportWorkflowResource(ctx, failed)
			require.NoError(t, err)
			if test.legacy {
				// Older CreateRun versions kept this unused caller-supplied field.
				_, err = store.db.Exec(`UPDATE "run_details" SET "PipelineSpecManifest" = ? WHERE "UUID" = ?`, v2SpecHelloWorld, run.UUID)
				require.NoError(t, err)
			}
			if test.newerV2 {
				store.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal("123e4567-e89b-12d3-a456-426655440001", nil))
				manager.pipelineStore = store.PipelineStore()
				newer, err := manager.CreatePipelineVersion(createPipelineVersion(
					pipeline.UUID, "newer-v2-version", "", "", v2SpecHelloWorld, "", experiment.Namespace))
				require.NoError(t, err)
				require.NoError(t, manager.UpdatePipelineDefaultVersion(pipeline.UUID, newer.UUID))
				latest, err := manager.pipelineStore.GetLatestPipelineVersion(pipeline.UUID)
				require.NoError(t, err)
				require.Equal(t, newer.UUID, latest.UUID)
			}
			if test.deletedVersion {
				require.NoError(t, manager.DeletePipelineVersion(version.UUID))
			}
			if test.missingSource {
				_, err = store.db.Exec(`UPDATE "pipeline_versions" SET "PipelineSpec" = '' WHERE "UUID" = ?`, version.UUID)
				require.NoError(t, err)
				require.NoError(t, manager.objectStore.AddFile(ctx, []byte(v2SpecHelloWorld), manager.objectStore.GetPipelineKey(pipeline.UUID)))
			}
			if test.missingPin {
				_, err = store.db.Exec(`UPDATE "run_details" SET "PipelineVersionId" = '' WHERE "UUID" = ?`, run.UUID)
				require.NoError(t, err)
			}
			before, err := manager.GetRun(run.UUID)
			require.NoError(t, err)
			require.Equal(t, model.RuntimeStateFailed, before.State)
			beforeWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(context.Background(), run.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			beforeManifest := beforeWorkflow.ToStringForStore()
			podCounter := &provenancePodDeletionCounter{KubernetesCoreInterface: manager.k8sCoreClient}
			manager.k8sCoreClient = podCounter

			viper.Set(common.WorkflowIdentityMode, "enforce")
			err = manager.RetryRun(ctx, run.UUID)
			if test.v2 || test.literal {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "podSpecPatch")
				assert.Zero(t, podCounter.deletes, "authorization must precede pod deletion")
			}
			after, err := manager.GetRun(run.UUID)
			require.NoError(t, err)
			afterWorkflow, err := store.ExecClient().Execution(run.Namespace).Get(ctx, run.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			if test.v2 || test.literal {
				assert.Equal(t, model.RuntimeStateRunning, after.State)
				assert.Equal(t, int64(1), after.RetryGeneration)
				assert.Equal(t, string(workflowapi.WorkflowRunning), string(afterWorkflow.ExecutionStatus().Condition()))
			} else {
				assert.Equal(t, before, after, "authorization must fail before retry state is claimed or reset")
				assert.Equal(t, beforeManifest, afterWorkflow.ToStringForStore())
			}
		})
	}
}

func TestCreateRunAndJob_PersistOnlySelectedPipelineSource(t *testing.T) {
	for _, v2 := range []bool{false, true} {
		for _, job := range []bool{false, true} {
			name := "V1/run"
			if v2 {
				name = "V2/run"
			}
			if job {
				name += "/job"
			}
			t.Run(name, func(t *testing.T) {
				store, manager, experiment := initWithExperiment(t)
				defer store.Close()
				workflowManifest := testWorkflowWithoutStatus().ToStringForStore()
				manifest := workflowManifest
				if v2 {
					manifest = v2SpecHelloWorld
				}
				pipeline, err := manager.CreatePipeline(createPipeline("source", "", experiment.Namespace))
				require.NoError(t, err)
				version, err := manager.CreatePipelineVersion(createPipelineVersion(
					pipeline.UUID, "selected-version", "", "", manifest, "", experiment.Namespace))
				require.NoError(t, err)
				pipelineSpec := model.PipelineSpec{
					PipelineVersionId:    version.UUID,
					PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
					WorkflowSpecManifest: model.LargeText(workflowManifest),
					Parameters:           `[{"name":"param1","value":"world"}]`,
					RuntimeConfig:        model.RuntimeConfig{Parameters: `{"text":"world"}`},
				}
				var stored model.PipelineSpec
				if job {
					created, err := manager.CreateJob(context.Background(), &model.Job{
						DisplayName: "source-job", ExperimentId: experiment.UUID, PipelineSpec: pipelineSpec,
					})
					require.NoError(t, err)
					persisted, err := manager.GetJob(created.UUID)
					require.NoError(t, err)
					stored = persisted.PipelineSpec
				} else {
					created, err := manager.CreateRun(context.Background(), &model.Run{
						DisplayName: "source-run", ExperimentId: experiment.UUID, PipelineSpec: pipelineSpec,
					})
					require.NoError(t, err)
					persisted, err := manager.GetRun(created.UUID)
					require.NoError(t, err)
					stored = persisted.PipelineSpec
				}
				assert.Equal(t, version.UUID, stored.PipelineVersionId)
				if v2 {
					assert.Equal(t, version.PipelineSpec, stored.PipelineSpecManifest)
					assert.Empty(t, stored.WorkflowSpecManifest)
				} else {
					assert.Equal(t, version.PipelineSpec, stored.WorkflowSpecManifest)
					assert.Empty(t, stored.PipelineSpecManifest)
				}
			})
		}
	}
}
