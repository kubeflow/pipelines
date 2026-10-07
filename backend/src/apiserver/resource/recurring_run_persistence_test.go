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
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Kubernetes can notify the persistence agent before Create returns to the API
// server. Run the report at that exact boundary, without relying on timing.
type firstReportBeforeCreateReturnsClient struct {
	util.ExecutionInterface
	beforeReturn func(util.ExecutionSpec)
}

func (c *firstReportBeforeCreateReturnsClient) Create(ctx context.Context, execution util.ExecutionSpec, options metav1.CreateOptions) (util.ExecutionSpec, error) {
	created, err := c.ExecutionInterface.Create(ctx, execution, options)
	if err != nil {
		return nil, err
	}
	// The creator still receives the original response even if the live workflow
	// has progressed and its newer status has already been reported.
	response := util.NewWorkflow(created.(*util.Workflow).DeepCopy())
	c.beforeReturn(created)
	return response, nil
}

func TestCreateRunFirstRecurringReportPreservesResolvedInputs(t *testing.T) {
	for _, multiUser := range []bool{false, true} {
		for _, test := range []struct {
			phase workflowapi.WorkflowPhase
			state model.RuntimeState
		}{
			{phase: workflowapi.WorkflowRunning, state: model.RuntimeStateRunning},
			{phase: workflowapi.WorkflowSucceeded, state: model.RuntimeStateSucceeded},
		} {
			t.Run(fmt.Sprintf("multiuser=%t/%s", multiUser, test.phase), func(t *testing.T) {
				initEnvVars()
				previousMode := viper.Get(common.MultiUserMode)
				viper.Set(common.MultiUserMode, multiUser)
				t.Cleanup(func() { viper.Set(common.MultiUserMode, previousMode) })
				store := NewFakeClientManagerOrFatalV2()
				defer store.Close()
				manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
				manager.time = fixedRecurringTime{epoch: 200}
				ctx := multiUserContext()
				experiment, err := manager.CreateExperiment(&model.Experiment{Name: "first-report", Namespace: "ns1"})
				require.NoError(t, err)
				pipeline, err := manager.CreatePipeline(createPipeline("follow-latest", "", experiment.Namespace))
				require.NoError(t, err)
				version, err := manager.CreatePipelineVersion(createPipelineVersion(
					pipeline.UUID, "selected-version", "", "", v2SpecHelloWorld, "", experiment.Namespace))
				require.NoError(t, err)
				job, err := manager.CreateJob(ctx, &model.Job{
					DisplayName: "follow-latest", Namespace: experiment.Namespace, ExperimentId: experiment.UUID,
					Enabled: true, MaxConcurrency: 1,
					Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{
						PeriodicScheduleStartTimeInSec: util.Int64Pointer(100), IntervalSecond: util.Int64Pointer(10),
					}},
					PipelineSpec: model.PipelineSpec{
						PipelineId: pipeline.UUID,
						RuntimeConfig: model.RuntimeConfig{
							Parameters: `{"text":"selected-value"}`, PipelineRoot: "selected-root",
						},
					},
				})
				require.NoError(t, err)
				require.Empty(t, job.PipelineVersionId, "the recurring run must follow latest")
				require.Empty(t, job.PipelineSpecManifest)
				dispatcher := &pausedRecurringRunDispatcher{}
				manager.pluginDispatcher = dispatcher
				input := &model.Run{
					DisplayName: "scheduled-tick", RecurringRunId: job.UUID,
					Namespace: job.Namespace, ExperimentId: job.ExperimentId, PipelineSpec: job.PipelineSpec,
				}
				require.NoError(t, manager.PrepareRecurringRun(ctx, input))
				var firstReport *model.Run
				workflowClient := store.ExecClientFake.Execution(job.Namespace)
				manager.execClient = &retryWorkflowExecClient{workflowClient: &firstReportBeforeCreateReturnsClient{
					ExecutionInterface: workflowClient,
					beforeReturn: func(execution util.ExecutionSpec) {
						// "Latest" changes after compilation. Recovery must retain the
						// selected version rather than resolving the current default.
						newManifest := strings.Replace(v2SpecHelloWorld, "image: python:3.11", "image: python:3.12", 1)
						require.NotEqual(t, v2SpecHelloWorld, newManifest)
						_, err := manager.CreatePipelineVersion(createPipelineVersion(
							pipeline.UUID, "newer-version", "", "", newManifest, "", experiment.Namespace))
						require.NoError(t, err)
						latest, err := manager.GetDefaultPipelineVersion(pipeline.UUID)
						require.NoError(t, err)
						require.NotEqual(t, version.UUID, latest.UUID)
						workflow := execution.(*util.Workflow)
						workflow.Status.Phase = test.phase
						if test.phase == workflowapi.WorkflowSucceeded {
							workflow.Status.FinishedAt = metav1.NewTime(time.Unix(200, 0))
						}
						execution, err = workflowClient.Update(ctx, execution, metav1.UpdateOptions{})
						require.NoError(t, err)
						_, err = manager.ReportWorkflowResource(ctx, execution)
						require.NoError(t, err)
						firstReport, err = manager.GetRun(input.UUID)
						require.NoError(t, err)
						require.Equal(t, "scheduled-tick", firstReport.DisplayName)
						require.Equal(t, version.UUID, firstReport.PipelineVersionId)
						require.Equal(t, version.Name, firstReport.PipelineName)
						require.Equal(t, pipeline.UUID, firstReport.PipelineId)
						require.Equal(t, input.PipelineSpecManifest, firstReport.PipelineSpecManifest)
						require.NotEmpty(t, firstReport.PipelineSpecManifest)
						require.Equal(t, input.RuntimeConfig, firstReport.RuntimeConfig)
						require.Equal(t, execution.ServiceAccount(), firstReport.ServiceAccount)
						require.Equal(t, test.state, firstReport.State)
						if test.phase == workflowapi.WorkflowSucceeded {
							require.Equal(t, int64(200), firstReport.FinishedAtInSec)
						}
					},
				}}
				created, err := manager.CreateRun(ctx, input)
				require.NoError(t, err)
				require.NotNil(t, firstReport)
				require.Equal(t, firstReport.UUID, created.UUID)
				require.Equal(t, firstReport.DisplayName, created.DisplayName)
				require.Equal(t, firstReport.PipelineSpec, created.PipelineSpec)
				require.Equal(t, test.state, created.State, "creator must preserve the newer report")
				require.Equal(t, firstReport.FinishedAtInSec, created.FinishedAtInSec)
				persisted, err := manager.GetRun(created.UUID)
				require.NoError(t, err)
				require.Equal(t, test.state, persisted.State)
				require.Equal(t, firstReport.FinishedAtInSec, persisted.FinishedAtInSec)
				require.Equal(t, firstReport.PipelineSpec, persisted.PipelineSpec)
				require.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())
				require.Zero(t, dispatcher.endCalls.Load())
			})
		}
	}
}

func TestRecurringRunReportPipelineSpecValidatesSelectedSource(t *testing.T) {
	for _, test := range []struct {
		name      string
		errorText string
	}{
		{name: "wrong pipeline", errorText: "belongs to pipeline"},
		{name: "different pinned version", errorText: "differs from the pinned recurring-run version"},
		{name: "stored version for inline pipeline", errorText: "an inline pipeline cannot select a stored pipeline version"},
		{name: "missing selected version", errorText: "Failed to recover the selected recurring-run pipeline version"},
		{name: "deleted pinned version retains snapshot"},
		{name: "inline pipeline without stored version"},
		{name: "unannotated legacy workflow"},
	} {
		t.Run(test.name, func(t *testing.T) {
			previousMode := viper.Get(common.MultiUserMode)
			viper.Set(common.MultiUserMode, false)
			t.Cleanup(func() { viper.Set(common.MultiUserMode, previousMode) })
			store, manager, pipeline, version := initWithPipeline(t)
			defer store.Close()
			job := &model.Job{PipelineSpec: model.PipelineSpec{
				PipelineId: pipeline.UUID,
				RuntimeConfig: model.RuntimeConfig{
					Parameters: `{"text":"retained-value"}`, PipelineRoot: "retained-root",
				},
			}}
			workflow := util.NewWorkflow(&workflowapi.Workflow{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{annotationKeyRecurringRunPipelineVersion: version.UUID},
			}})
			switch test.name {
			case "wrong pipeline":
				job.PipelineId = "another-pipeline"
			case "different pinned version":
				job.PipelineVersionId = "another-version"
				job.PipelineSpecManifest = version.PipelineSpec
			case "stored version for inline pipeline":
				job.PipelineId = ""
				job.PipelineSpecManifest = version.PipelineSpec
			case "missing selected version":
				workflow.SetAnnotations(annotationKeyRecurringRunPipelineVersion, "missing-version")
			case "deleted pinned version retains snapshot":
				job.PipelineVersionId = version.UUID
				job.PipelineName = version.Name
				job.PipelineSpecManifest = version.PipelineSpec
				require.NoError(t, manager.DeletePipelineVersion(version.UUID))
			case "inline pipeline without stored version":
				job.PipelineId = ""
				job.PipelineSpecManifest = version.PipelineSpec
				workflow.SetAnnotations(annotationKeyRecurringRunPipelineVersion, "")
			case "unannotated legacy workflow":
				workflow.Annotations = nil
			}
			original := job.PipelineSpec
			resolved, err := manager.recurringRunReportPipelineSpec(job, workflow)
			if test.errorText != "" {
				require.ErrorContains(t, err, test.errorText)
				require.Empty(t, resolved, "a rejected source must not yield a partial run specification")
			} else {
				require.NoError(t, err)
				require.Equal(t, original, resolved)
			}
			require.Equal(t, original, job.PipelineSpec, "recovery must not pin or mutate the recurring run")
		})
	}
}

func TestRecurringRunReportPipelineSpecRequiresMatchingSchedulingClaim(t *testing.T) {
	for _, test := range []struct {
		name      string
		claim     bool
		wrongRun  bool
		versionID string
		wantError bool
	}{
		{name: "missing claim", wantError: true},
		{name: "wrong run ID", claim: true, wrongRun: true, wantError: true},
		{name: "wrong pipeline version", claim: true, versionID: "unclaimed-version", wantError: true},
		{name: "matching inline claim", claim: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			previousMode := viper.Get(common.MultiUserMode)
			viper.Set(common.MultiUserMode, false)
			t.Cleanup(func() { viper.Set(common.MultiUserMode, previousMode) })
			store, manager, job := initWithJob(t)
			defer store.Close()
			viper.Set(common.MultiUserMode, true)
			if test.claim {
				_, err := store.JobStore().ClaimRecurringRun(job.UUID, "first-tick", 0, 110, 200, "")
				require.NoError(t, err)
			}
			runID := util.NewDeterministicUUID(job.UUID + "/tick/1")
			if test.wrongRun {
				runID = util.NewDeterministicUUID(job.UUID + "/tick/2")
			}
			workflow := util.NewWorkflow(&workflowapi.Workflow{ObjectMeta: metav1.ObjectMeta{
				Labels:      map[string]string{util.LabelKeyWorkflowRunId: runID},
				Annotations: map[string]string{annotationKeyRecurringRunPipelineVersion: test.versionID},
			}})
			resolved, err := manager.recurringRunReportPipelineSpec(job, workflow)
			if test.wantError {
				require.ErrorContains(t, err, "workflow does not match the selected scheduling claim")
				require.Empty(t, resolved)
			} else {
				require.NoError(t, err)
				require.Equal(t, job.PipelineSpec, resolved)
			}
		})
	}
}
