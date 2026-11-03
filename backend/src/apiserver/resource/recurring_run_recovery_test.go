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
	"testing"

	gc "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type reportingRecurringExecutionClient struct {
	util.ExecutionInterface
	report func(util.ExecutionSpec)
}

func (c *reportingRecurringExecutionClient) Create(ctx context.Context, execution util.ExecutionSpec, options metav1.CreateOptions) (util.ExecutionSpec, error) {
	created, err := c.ExecutionInterface.Create(ctx, execution, options)
	if err == nil {
		c.report(created)
	}
	return created, err
}

func TestRecurringRunReportBeforeCreatorPersistsSelectedVersion(t *testing.T) {
	for _, multiUser := range []bool{false, true} {
		t.Run(fmt.Sprintf("multiuser=%t", multiUser), func(t *testing.T) {
			store, manager, experiment := initWithExperiment(t)
			defer store.Close()
			previous := viper.Get(common.MultiUserMode)
			viper.Set(common.MultiUserMode, multiUser)
			t.Cleanup(func() { viper.Set(common.MultiUserMode, previous) })
			manager.time = fixedRecurringTime{epoch: 200}
			ctx := multiUserContext()
			pipeline, err := manager.CreatePipeline(createPipeline("recovery-source", "", "ns1"))
			require.NoError(t, err)
			version, err := manager.CreatePipelineVersion(createPipelineVersion(pipeline.UUID, "selected", "", "", v2SpecHelloWorld, "", "ns1"))
			require.NoError(t, err)
			job, err := manager.CreateJob(ctx, &model.Job{
				DisplayName: "latest-schedule", Namespace: "ns1", ExperimentId: experiment.UUID, Enabled: true, MaxConcurrency: 1,
				Trigger:      model.Trigger{PeriodicSchedule: model.PeriodicSchedule{PeriodicScheduleStartTimeInSec: util.Int64Pointer(100), IntervalSecond: util.Int64Pointer(10)}},
				PipelineSpec: model.PipelineSpec{PipelineId: pipeline.UUID, RuntimeConfig: model.RuntimeConfig{Parameters: `{"text":"world"}`}},
			})
			require.NoError(t, err)
			require.Empty(t, job.PipelineVersionId)
			input := &model.Run{DisplayName: "helloworld-tick", Namespace: "ns1", ExperimentId: experiment.UUID, RecurringRunId: job.UUID, PipelineSpec: job.PipelineSpec}
			require.NoError(t, manager.PrepareRecurringRun(ctx, input))
			manager.execClient = &retryWorkflowExecClient{workflowClient: &reportingRecurringExecutionClient{
				ExecutionInterface: store.ExecClientFake.Execution("ns1"),
				report: func(workflow util.ExecutionSpec) {
					// Another upload must not change the version selected before this Workflow was created.
					store.UpdateUUID(util.NewFakeUUIDGeneratorOrFatal("123e4567-e89b-12d3-a456-426655440001", nil))
					manager.pipelineStore = store.PipelineStore()
					newer, err := manager.CreatePipelineVersion(createPipelineVersion(pipeline.UUID, "newer", "", "", v2SpecHelloWorld, "", "ns1"))
					require.NoError(t, err)
					_, err = store.db.Exec(`UPDATE "pipeline_versions" SET "CreatedAtInSec" = 300 WHERE "UUID" = ?`, newer.UUID)
					require.NoError(t, err)
					latest, err := manager.GetDefaultPipelineVersion(pipeline.UUID)
					require.NoError(t, err)
					require.Equal(t, newer.UUID, latest.UUID)
					if multiUser {
						workflow.SetAnnotations(recurringRunPipelineVersionAnnotation, newer.UUID)
					}
					_, err = manager.ReportWorkflowResource(ctx, workflow)
					require.NoError(t, err)
				},
			}}
			created, err := manager.CreateRun(ctx, input)
			require.NoError(t, err)
			require.Equal(t, version.UUID, created.PipelineVersionId)
			require.Equal(t, "helloworld-tick", created.DisplayName)
			require.Equal(t, version.Name, created.PipelineName)
			require.NotEmpty(t, created.PipelineSpecManifest)
			require.Equal(t, input.PipelineSpecManifest, created.PipelineSpecManifest)
			stored, err := manager.GetRun(created.UUID)
			require.NoError(t, err)
			require.Equal(t, created.PipelineSpec, stored.PipelineSpec)
			raw, err := yaml.YAMLToJSON([]byte(stored.PipelineSpecManifest))
			require.NoError(t, err)
			spec := &structpb.Struct{}
			require.NoError(t, protojson.Unmarshal(raw, spec))
			resolved, err := kfpapi.New(nil).FetchPipelineSpecFromRun(ctx, &gc.Run{PipelineSource: &gc.Run_PipelineSpec{PipelineSpec: spec}})
			require.NoError(t, err)
			_, err = util.NewScopePathFromStruct(resolved)
			require.NoError(t, err, "the driver must resolve its root without a version API request")
			require.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())
		})
	}
}

func TestRecurringRunReportRequiresExactPipelineSource(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()
	pipeline, err := manager.CreatePipeline(createPipeline("source", "", experiment.Namespace))
	require.NoError(t, err)
	version, err := manager.CreatePipelineVersion(createPipelineVersion(pipeline.UUID, "selected", "", "", v2SpecHelloWorld, "", experiment.Namespace))
	require.NoError(t, err)
	for _, test := range []struct {
		name    string
		source  model.PipelineSpec
		pin     string
		message string
	}{
		{name: "missing pin must not choose latest", source: model.PipelineSpec{PipelineId: pipeline.UUID}, message: "exact pipeline version"},
		{name: "different pipeline", source: model.PipelineSpec{PipelineId: "another-pipeline"}, pin: version.UUID, message: "different pipeline"},
		{name: "different pinned version", source: model.PipelineSpec{PipelineId: pipeline.UUID, PipelineVersionId: "another-version"}, pin: version.UUID, message: "pinned schedule"},
		{name: "historical pinned source", source: model.PipelineSpec{PipelineId: pipeline.UUID, PipelineVersionId: version.UUID}},
		{name: "inline source ignores execution pin", source: model.PipelineSpec{PipelineSpecManifest: model.LargeText(v2SpecHelloWorld)}, pin: "untrusted-version"},
	} {
		t.Run(test.name, func(t *testing.T) {
			workflow := testWorkflowWithoutStatus()
			workflow.SetAnnotations(recurringRunPipelineVersionAnnotation, test.pin)
			restored, err := manager.recurringRunReportPipelineSpec(&model.Job{PipelineSpec: test.source}, workflow, "run-id")
			if test.message != "" {
				require.ErrorContains(t, err, test.message)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, restored.PipelineSpecManifest)
			if test.source.PipelineVersionId != "" {
				require.Equal(t, version.UUID, restored.PipelineVersionId)
			}
		})
	}
	_, err = store.db.Exec(`UPDATE "pipeline_versions" SET "PipelineSpec" = '' WHERE "UUID" = ?`, version.UUID)
	require.NoError(t, err)
	workflow := testWorkflowWithoutStatus()
	workflow.SetAnnotations(recurringRunPipelineVersionAnnotation, version.UUID)
	_, err = manager.recurringRunReportPipelineSpec(&model.Job{PipelineSpec: model.PipelineSpec{PipelineId: pipeline.UUID}}, workflow, "run-id")
	require.ErrorContains(t, err, "Restore the manifest", "missing historical manifests must not use a pipeline-level latest fallback")
}
