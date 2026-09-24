// Copyright 2018 The Kubeflow Authors
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

package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const testPluginsExperimentName = "my-exp"
const testPluginsRecurringExperimentName = "recurring-exp"
const testPluginsJobName = "test-job"
const testPluginsUnsafeJavaScriptURL = "javascript:alert(1)"
const testPluginsURLBase = "https://example.com/"

func setPluginLimitsConfigForTest(t *testing.T, values map[string]string) {
	t.Helper()
	for key, value := range values {
		viper.Set(key, value)
	}
	t.Cleanup(func() {
		viper.Reset()
		// Restore package TestMain default; viper.Reset() clears all viper state.
		viper.Set(common.PipelineURLValidationEnabled, "false")
	})
}

func strPtr(s string) *string {
	return &s
}

func testLargeTextPtr(s string) *model.LargeText {
	lt := model.LargeText(s)
	return &lt
}

// createPluginInputMapWithNKeys builds n plugin input entries with a small valid payload.
func createPluginInputMapWithNKeys(n int) map[string]*structpb.Struct {
	input := make(map[string]*structpb.Struct, n)
	for i := range n {
		input[fmt.Sprintf("plugin-%d", i)] = &structpb.Struct{
			Fields: map[string]*structpb.Value{"k": structpb.NewStringValue("ok")},
		}
	}
	return input
}

func createPluginOutputMapWithNKeys(n int) map[string]*apiv2beta1.PluginOutput {
	output := make(map[string]*apiv2beta1.PluginOutput, n)
	for i := range n {
		output[fmt.Sprintf("plugin-%d", i)] = &apiv2beta1.PluginOutput{
			Entries: map[string]*apiv2beta1.MetadataValue{
				"run_url": {
					Value:      structpb.NewStringValue(testPluginsURLBase),
					RenderType: apiv2beta1.MetadataValue_URL.Enum(),
				},
			},
			State: apiv2beta1.PluginState_PLUGIN_RUNNING,
		}
	}
	return output
}

func TestToModelExperiment(t *testing.T) {
	tests := []struct {
		name                    string
		experiment              *apiv2beta1.Experiment
		wantError               bool
		errorMessage            string
		expectedModelExperiment *model.Experiment
	}{

		{
			"Happy pass v2",
			&apiv2beta1.Experiment{
				DisplayName: "exp2",
				Description: "API V2beta1 test experiment",
				Namespace:   "ns2",
			},
			false,
			"",
			&model.Experiment{
				Name:         "exp2",
				Description:  "API V2beta1 test experiment",
				Namespace:    "ns2",
				StorageState: model.StorageStateAvailable,
			},
		},
		{
			"Empty namespace v2",
			&apiv2beta1.Experiment{
				DisplayName: "exp2",
				Description: "API V2beta1 test experiment",
			},
			false,
			"",
			&model.Experiment{
				Name:         "exp2",
				Description:  "API V2beta1 test experiment",
				Namespace:    "",
				StorageState: model.StorageStateAvailable,
			},
		},
		{
			"missing name v2",
			&apiv2beta1.Experiment{
				DisplayName: "",
				Description: "API V2beta1 test experiment",
				Namespace:   "ns2",
			},
			true,
			"Experiment must have a non-empty name",
			nil,
		},
	}

	for _, tc := range tests {
		modelExperiment, err := toModelExperiment(tc.experiment)
		if tc.wantError {
			if err == nil {
				t.Errorf("TesttoModelExperiment(%v) expect error but got nil", tc.name)
			} else if !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("TesttoModelExperiment(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
			}
		} else {
			if err != nil {
				t.Errorf("TesttoModelExperiment(%v) expect no error but got %v", tc.name, err)
			} else if !cmp.Equal(tc.expectedModelExperiment, modelExperiment) {
				t.Errorf("TesttoModelExperiment(%v) expect (%+v) but got (%+v)", tc.name, tc.expectedModelExperiment, modelExperiment)
			}
		}
	}
}

func TestToModelPipeline(t *testing.T) {
	tests := []struct {
		name                  string
		pipeline              *apiv2beta1.Pipeline
		wantError             bool
		errorMessage          string
		expectedModelPipeline *model.Pipeline
	}{

		{
			"Empty namespace v2",
			&apiv2beta1.Pipeline{
				DisplayName: "p6",
				Description: "This is a pipeline6",
				Namespace:   "",
			},
			false,
			"",
			&model.Pipeline{
				Name:        "p6",
				DisplayName: "p6",
				Description: "This is a pipeline6",
				Status:      model.PipelineCreating,
				Namespace:   "",
			},
		},
		{
			"Valid namespace v2",
			&apiv2beta1.Pipeline{
				DisplayName: "p7",
				Description: "This is a pipeline7",
				Namespace:   "ns2",
				Error:       &status.Status{Message: "test error"},
			},
			false,
			"",
			&model.Pipeline{
				Name:        "p7",
				DisplayName: "p7",
				Description: "This is a pipeline7",
				Status:      model.PipelineCreating,
				Namespace:   "ns2",
			},
		},
		{
			"Empty name v2",
			&apiv2beta1.Pipeline{
				DisplayName: "",
				Description: "This is a pipeline8",
				Namespace:   "ns3",
			},
			false,
			"",
			&model.Pipeline{
				Name:        "",
				DisplayName: "",
				Description: "This is a pipeline8",
				Status:      model.PipelineCreating,
				Namespace:   "ns3",
			},
		},
		{
			name: "name too long v2",
			pipeline: &apiv2beta1.Pipeline{
				DisplayName: strings.Repeat("a", 129), // Max is 128
				Description: "This is a pipeline with a very long name",
				Namespace:   "ns",
			},
			wantError:             true,
			errorMessage:          "Pipeline.Name length cannot exceed 128",
			expectedModelPipeline: nil,
		},
		{
			name: "namespace too long v2",
			pipeline: &apiv2beta1.Pipeline{
				DisplayName: "p_long_ns",
				Description: "This is a pipeline with a very long namespace",
				Namespace:   strings.Repeat("n", 64), // Max is 63
			},
			wantError:             true,
			errorMessage:          "Pipeline.Namespace length cannot exceed 63",
			expectedModelPipeline: nil,
		},
	}

	for _, tc := range tests {
		modelPipeline, err := toModelPipeline(tc.pipeline)
		if tc.wantError {
			if err == nil {
				t.Errorf("TesttoModelExperiment(%v) expect error but got nil", tc.name)
			} else if !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("TesttoModelExperiment(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
			}
		} else {
			if err != nil {
				t.Errorf("TesttoModelPipeline(%v) expect no error but got %v", tc.name, err)
			} else if !cmp.Equal(tc.expectedModelPipeline, modelPipeline) {
				t.Errorf("TesttoModelPipeline(%v) expect (%+v) but got (%+v)", tc.name, tc.expectedModelPipeline, modelPipeline)
			}
		}
	}
}

func TestToModelPipelineVersion(t *testing.T) {
	tests := []struct {
		name                    string
		pipeline                *apiv2beta1.PipelineVersion
		expectedPipelineVersion *model.PipelineVersion
		isError                 bool
		errMsg                  string
	}{

		{
			"happy pipeline version v2",
			&apiv2beta1.PipelineVersion{
				DisplayName:   "Version 2 v2beta1",
				PipelineId:    "pipeline 333",
				PackageUrl:    &apiv2beta1.Url{PipelineUrl: "http://package/3333"},
				CodeSourceUrl: "http://repo/3333",
				Description:   "This is pipeline version 333",
			},
			&model.PipelineVersion{
				Name:            "Version 2 v2beta1",
				DisplayName:     "Version 2 v2beta1",
				PipelineId:      "pipeline 333",
				PipelineSpecURI: "http://package/3333",
				CodeSourceUrl:   "http://repo/3333",
				Description:     "This is pipeline version 333",
				Status:          model.PipelineVersionCreating,
			},
			false,
			"",
		},
		{
			name: "name too long v2",
			pipeline: &apiv2beta1.PipelineVersion{
				DisplayName:   strings.Repeat("a", 128), // Max is 127
				PipelineId:    "pipeline 333",
				PackageUrl:    &apiv2beta1.Url{PipelineUrl: "http://package/3333"},
				CodeSourceUrl: "http://repo/3333",
				Description:   "This is pipeline version 333",
			},
			expectedPipelineVersion: nil,
			isError:                 true,
			errMsg:                  "PipelineVersion.Name length cannot exceed 127",
		},
		{
			"missing package Url v2",
			&apiv2beta1.PipelineVersion{
				DisplayName: "Version 2 v2beta1",
				PipelineId:  "pipeline 333",
				Description: "This is pipeline version 333",
			},
			nil,
			true,
			"Invalid input error: Failed to convert v2beta1 API pipeline version to its internal representation due to missing pipeline URL",
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				pipelineVersion, err := toModelPipelineVersion(tt.pipeline)
				if tt.isError {
					assert.NotNil(t, err)
					assert.Contains(t, err.Error(), tt.errMsg)
				} else {
					assert.Nil(t, err)
				}
				assert.Equal(t, tt.expectedPipelineVersion, pipelineVersion)
			},
		)
	}
}

func TestToApiPipeline(t *testing.T) {
	tests := []struct {
		name             string
		pipeline         *model.Pipeline
		expectedPipeline *apiv2beta1.Pipeline
	}{
		{
			"happy case",
			&model.Pipeline{
				UUID:           "p1",
				Name:           "pipeline1",
				DisplayName:    "pipeline1",
				Description:    "This is pipeline1",
				Namespace:      "ns1",
				CreatedAtInSec: 1,
			},
			&apiv2beta1.Pipeline{
				PipelineId:  "p1",
				Name:        "pipeline1",
				DisplayName: "pipeline1",
				Description: "This is pipeline1",
				CreatedAt:   &timestamppb.Timestamp{Seconds: 1},
				Namespace:   "ns1",
			},
		},
		{
			"nil input",
			nil,
			&apiv2beta1.Pipeline{
				Error: util.ToRpcStatus(
					util.NewInternalServerError(
						errors.New("Pipeline cannot be nil"),
						"Failed to convert a pipeline to API pipeline",
					),
				),
			},
		},
		{
			"empty uuid",
			&model.Pipeline{
				Name:           "pipeline1",
				DisplayName:    "pipeline1",
				Description:    "This is pipeline1",
				Namespace:      "ns1",
				CreatedAtInSec: 1,
			},
			&apiv2beta1.Pipeline{
				Error: util.ToRpcStatus(
					util.NewInternalServerError(
						errors.New("Pipeline id cannot be empty"),
						"Failed to convert a pipeline to API pipeline",
					),
				),
			},
		},
		{
			"zero create time",
			&model.Pipeline{
				UUID:        "p1",
				Name:        "pipeline1",
				DisplayName: "pipeline1",
				Description: "This is pipeline1",
				Namespace:   "ns1",
			},
			&apiv2beta1.Pipeline{
				PipelineId: "p1",
				Error: util.ToRpcStatus(
					util.NewInternalServerError(
						errors.New("Pipeline create time cannot be 0"),
						"Failed to convert a pipeline to API pipeline",
					),
				),
			},
		},
		{
			"empty name",
			&model.Pipeline{
				UUID:           "p1",
				Description:    "This is pipeline1",
				Namespace:      "ns1",
				CreatedAtInSec: 1,
			},
			&apiv2beta1.Pipeline{
				PipelineId: "p1",
				Error: util.ToRpcStatus(
					util.NewInternalServerError(
						errors.New("Pipeline name cannot be empty"),
						"Failed to convert a pipeline to API pipeline",
					),
				),
			},
		},
		{
			"empty namespace",
			&model.Pipeline{
				UUID:           "p1",
				Name:           "pipeline1",
				DisplayName:    "pipeline1",
				Description:    "This is pipeline1",
				CreatedAtInSec: 1,
			},
			&apiv2beta1.Pipeline{
				PipelineId:  "p1",
				Name:        "pipeline1",
				DisplayName: "pipeline1",
				Description: "This is pipeline1",
				CreatedAt:   &timestamppb.Timestamp{Seconds: 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := toApiPipeline(tt.pipeline)
			assert.Equal(t, tt.expectedPipeline, pipeline)
		})
	}
}

func TestToApiPipelines(t *testing.T) {
	modelPipelines := []*model.Pipeline{
		{
			UUID:           "p1",
			Name:           "pipeline1",
			DisplayName:    "pipeline1",
			Description:    "This is pipeline1",
			Namespace:      "ns1",
			CreatedAtInSec: 1,
		},
		nil,
		{
			Name:           "pipeline1",
			DisplayName:    "pipeline1",
			Description:    "This is pipeline1",
			Namespace:      "ns1",
			CreatedAtInSec: 1,
		},
		{
			UUID:        "p1",
			Name:        "pipeline1",
			DisplayName: "pipeline1",
			Description: "This is pipeline1",
			Namespace:   "ns1",
		},
		{
			UUID:           "p1",
			Description:    "This is pipeline1",
			Namespace:      "ns1",
			CreatedAtInSec: 1,
		},
		{
			UUID:           "p1",
			Name:           "pipeline1",
			DisplayName:    "pipeline1",
			Description:    "This is pipeline1",
			CreatedAtInSec: 1,
		},
	}
	apiPipelines := toApiPipelines(modelPipelines)
	expectedPipelines := []*apiv2beta1.Pipeline{
		{
			PipelineId:  "p1",
			Name:        "pipeline1",
			DisplayName: "pipeline1",
			Description: "This is pipeline1",
			CreatedAt:   &timestamppb.Timestamp{Seconds: 1},
			Namespace:   "ns1",
		},
		{
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					errors.New("Pipeline cannot be nil"),
					"Failed to convert a pipeline to API pipeline",
				),
			),
		},
		{
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					errors.New("Pipeline id cannot be empty"),
					"Failed to convert a pipeline to API pipeline",
				),
			),
		},
		{
			PipelineId: "p1",
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					errors.New("Pipeline create time cannot be 0"),
					"Failed to convert a pipeline to API pipeline",
				),
			),
		},
		{
			PipelineId: "p1",
			Error: util.ToRpcStatus(
				util.NewInternalServerError(
					errors.New("Pipeline name cannot be empty"),
					"Failed to convert a pipeline to API pipeline",
				),
			),
		},
		{
			PipelineId:  "p1",
			Name:        "pipeline1",
			DisplayName: "pipeline1",
			Description: "This is pipeline1",
			CreatedAt:   &timestamppb.Timestamp{Seconds: 1},
		},
	}
	assert.Equal(t, expectedPipelines, apiPipelines)

	modelPipelines2 := make([]*model.Pipeline, 0)
	apiPipelines2 := toApiPipelines(modelPipelines2)
	expectedPipelines2 := make([]*apiv2beta1.Pipeline, 0)
	assert.Equal(t, expectedPipelines2, apiPipelines2)
}

func TestToApiExperiments(t *testing.T) {
	exp1 := &model.Experiment{
		UUID:                  "exp1",
		CreatedAtInSec:        1,
		LastRunCreatedAtInSec: 1,
		Name:                  "experiment1",
		Description:           "My name is experiment1",
		StorageState:          "AVAILABLE",
	}
	exp2 := &model.Experiment{
		UUID:                  "exp2",
		CreatedAtInSec:        2,
		LastRunCreatedAtInSec: 2,
		Name:                  "experiment2",
		Description:           "My name is experiment2",
		StorageState:          "ARCHIVED",
	}
	exp3 := &model.Experiment{
		UUID:                  "exp3",
		CreatedAtInSec:        1,
		LastRunCreatedAtInSec: 1,
		Name:                  "experiment3",
		Description:           "experiment3 was created using V1 APIV1BETA1",
		StorageState:          "STORAGESTATE_AVAILABLE",
	}
	exp4 := &model.Experiment{
		UUID:                  "exp4",
		CreatedAtInSec:        2,
		LastRunCreatedAtInSec: 2,
		Name:                  "experiment4",
		Description:           "experiment4 was created using V1 APIV1BETA1",
		StorageState:          "STORAGESTATE_ARCHIVED",
	}
	exp5 := &model.Experiment{
		UUID:                  "exp5",
		CreatedAtInSec:        1,
		LastRunCreatedAtInSec: 1,
		Name:                  "experiment5",
		Description:           "My name is experiment5",
		StorageState:          "this is invalid storage state",
	}
	apiExps := toApiExperiments([]*model.Experiment{exp1, exp2, exp3, exp4, nil, exp5})
	expectedApiExps := []*apiv2beta1.Experiment{
		{
			ExperimentId:     "exp1",
			DisplayName:      "experiment1",
			Description:      "My name is experiment1",
			CreatedAt:        timestamppb.New(time.Unix(1, 0)),
			LastRunCreatedAt: timestamppb.New(time.Unix(1, 0)),
			StorageState:     apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["AVAILABLE"]),
		},
		{
			ExperimentId:     "exp2",
			DisplayName:      "experiment2",
			Description:      "My name is experiment2",
			CreatedAt:        timestamppb.New(time.Unix(2, 0)),
			LastRunCreatedAt: timestamppb.New(time.Unix(2, 0)),
			StorageState:     apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["ARCHIVED"]),
		},
		{
			ExperimentId:     "exp3",
			DisplayName:      "experiment3",
			Description:      "experiment3 was created using V1 APIV1BETA1",
			CreatedAt:        timestamppb.New(time.Unix(1, 0)),
			LastRunCreatedAt: timestamppb.New(time.Unix(1, 0)),
			StorageState:     apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["AVAILABLE"]),
		},
		{
			ExperimentId:     "exp4",
			DisplayName:      "experiment4",
			Description:      "experiment4 was created using V1 APIV1BETA1",
			CreatedAt:        timestamppb.New(time.Unix(2, 0)),
			LastRunCreatedAt: timestamppb.New(time.Unix(2, 0)),
			StorageState:     apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["ARCHIVED"]),
		},
		{},
		{
			ExperimentId:     "exp5",
			DisplayName:      "experiment5",
			Description:      "My name is experiment5",
			CreatedAt:        timestamppb.New(time.Unix(1, 0)),
			LastRunCreatedAt: timestamppb.New(time.Unix(1, 0)),
			StorageState:     apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["STORAGE_STATE_UNSPECIFIED"]),
		},
	}
	assert.Equal(t, expectedApiExps, apiExps)
}

func TestToMapProtoStructParameters(t *testing.T) {
	expectedApiParameters := map[string]*structpb.Value{
		"param2": structpb.NewStringValue("world"),
	}
	modelParameters := `{"param2":"world"}`
	actualApiParameters := toMapProtoStructParameters(modelParameters)
	assert.True(t, proto.Equal(&structpb.Struct{Fields: expectedApiParameters}, &structpb.Struct{Fields: actualApiParameters}))

	expectedApiParameters = map[string]*structpb.Value{
		"pipeline-root": structpb.NewStringValue("gs://my-bucket/tfx_taxi_simple/{{workflow.uid}}"),
		"version":       structpb.NewStringValue("2"),
	}
	modelParameters = `{"pipeline-root":"gs://my-bucket/tfx_taxi_simple/{{workflow.uid}}","version":"2"}`
	actualApiParameters = toMapProtoStructParameters(modelParameters)
	assert.True(t, proto.Equal(&structpb.Struct{Fields: expectedApiParameters}, &structpb.Struct{Fields: actualApiParameters}))

}

func TestToMapProtoStructParameters_HistoricalArrays(t *testing.T) {
	for _, tc := range []struct {
		name       string
		parameters string
		want       map[string]interface{}
	}{
		{"legacy strings", `[{"name":"text","value":"hello"},{"name":"number","value":"2"},{"name":"flag","value":"true"},{"name":"empty","value":""}]`, map[string]interface{}{"text": "hello", "number": "2", "flag": "true", "empty": ""}},
		{"native types", `{"number":2,"flag":true,"nested":{"key":"value"}}`, map[string]interface{}{"number": float64(2), "flag": true, "nested": map[string]interface{}{"key": "value"}}},
		{"malformed JSON", `[{`, nil},
		{"invalid legacy value", `[{"name":"number","value":2}]`, nil},
		{"scalar", `42`, nil},
		{"empty array", `[]`, map[string]interface{}{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := toMapProtoStructParameters(tc.parameters)
			if tc.want == nil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, tc.want, (&structpb.Struct{Fields: got}).AsMap())
		})
	}
}

func TestHistoricalParametersRemainVisible(t *testing.T) {
	for _, tc := range []struct {
		name              string
		runtimeParameters model.LargeText
		want              string
	}{
		{"legacy fallback", "", "historical"},
		{"native takes precedence", `{"text":"native"}`, "native"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := model.PipelineSpec{
				Parameters:    `[{"name":"text","value":"historical"}]`,
				RuntimeConfig: model.RuntimeConfig{Parameters: tc.runtimeParameters},
			}
			run := toApiRun(&model.Run{PipelineSpec: spec})
			require.NotNil(t, run.GetRuntimeConfig())
			assert.Equal(t, tc.want, run.GetRuntimeConfig().GetParameters()["text"].GetStringValue())
			job := toApiRecurringRun(&model.Job{PipelineSpec: spec})
			require.NotNil(t, job.GetRuntimeConfig())
			assert.Equal(t, tc.want, job.GetRuntimeConfig().GetParameters()["text"].GetStringValue())
		})
	}
}

func TestToApiRecurringRun(t *testing.T) {
	modelJob := &model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		K8SName:     "name1",
		Enabled:     true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(2),
				Cron:                       util.StringPointer("2 * *"),
			},
		},
		MaxConcurrency: 2,
		NoCatchup:      true,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "{\"param1\":\"world\"}",
				PipelineRoot: "job-1-root",
			},
		},
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
	}
	expectedRecurringRun := &apiv2beta1.RecurringRun{
		RecurringRunId: "job1",
		DisplayName:    "name 1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		CreatedAt:      timestamppb.New(time.Unix(2, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(2, 0)),
		MaxConcurrency: 2,
		NoCatchup:      true,
		PipelineSource: &apiv2beta1.RecurringRun_PipelineVersionReference{
			PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
				PipelineId: "1",
			},
		},
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(2, 0)),
				Cron:      "2 * *",
			}},
		},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			Parameters: map[string]*structpb.Value{
				"param1": {Kind: &structpb.Value_StringValue{StringValue: "world"}},
			},
			PipelineRoot: "job-1-root",
		},
		Status: apiv2beta1.RecurringRun_ENABLED,
	}
	modelJob2 := &model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		K8SName:     "name1",
		Enabled:     false,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(2),
				Cron:                       util.StringPointer("2 * *"),
			},
		},
		MaxConcurrency: 2,
		NoCatchup:      true,
		PipelineSpec: model.PipelineSpec{
			PipelineId:        "p1",
			PipelineVersionId: "pv1",
			PipelineName:      "p1",
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "{\"param1\":\"world\"}",
				PipelineRoot: "job-1-root",
			},
		},
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
	}
	expectedRecurringRun2 := &apiv2beta1.RecurringRun{
		RecurringRunId: "job1",
		DisplayName:    "name 1",
		Mode:           apiv2beta1.RecurringRun_DISABLE,
		CreatedAt:      timestamppb.New(time.Unix(2, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(2, 0)),
		MaxConcurrency: 2,
		NoCatchup:      true,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(2, 0)),
				Cron:      "2 * *",
			}},
		},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineVersionReference{
			PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
				PipelineId:        "p1",
				PipelineVersionId: "pv1",
			},
		},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			Parameters: map[string]*structpb.Value{
				"param1": {Kind: &structpb.Value_StringValue{StringValue: "world"}},
			},
			PipelineRoot: "job-1-root",
		},
		Status: apiv2beta1.RecurringRun_DISABLED,
	}
	apiRecurringRun := toApiRecurringRun(modelJob)
	// Compare the string representation of ApiRuns, since these structs have internal fields
	// used only by protobuff, and may be different. The .String() method marshal all
	// exported fields into string format.
	// See https://github.com/stretchr/testify/issues/758
	assert.Equal(t, expectedRecurringRun.String(), apiRecurringRun.String())

	apiRecurringRun2 := toApiRecurringRun(modelJob2)
	// Compare the string representation of ApiRuns, since these structs have internal fields
	// used only by protobuff, and may be different. The .String() method marshal all
	// exported fields into string format.
	// See https://github.com/stretchr/testify/issues/758
	assert.Equal(t, expectedRecurringRun2.String(), apiRecurringRun2.String())
}

func Test_toModelRuntimeState(t *testing.T) {
	tests := []struct {
		name     string
		apiState interface{}
		wantV1   model.RuntimeState
		wantV2   model.RuntimeState
		wantErr  bool
		errMsg   string
	}{
		{
			"V1 pending",
			"Pending",
			model.RuntimeStatePendingV1,
			model.RuntimeStatePending,
			false,
			"",
		},
		{
			"V1 Running",
			"Running",
			model.RuntimeStateRunningV1,
			model.RuntimeStateRunning,
			false,
			"",
		},
		{
			"V1 Succeeded",
			"Succeeded",
			model.RuntimeStateSucceededV1,
			model.RuntimeStateSucceeded,
			false,
			"",
		},
		{
			"V1 Skipped",
			"Skipped",
			model.RuntimeStateSkippedV1,
			model.RuntimeStateSkipped,
			false,
			"",
		},
		{
			"V1 Failed",
			"Failed",
			model.RuntimeStateFailedV1,
			model.RuntimeStateFailed,
			false,
			"",
		},
		{
			"V1 Error",
			"Error",
			model.RuntimeStateFailedV1,
			model.RuntimeStateFailed,
			false,
			"",
		},
		{
			"V1 Empty",
			"",
			model.RuntimeStateUnknownV1,
			model.RuntimeStateUnspecified,
			false,
			"",
		},
		{
			"V1 Unknown",
			"Unknown",
			model.RuntimeStateUnknownV1,
			model.RuntimeStateUnspecified,
			false,
			"",
		},
		{
			"V1 NO_STATUS",
			"NO_STATUS",
			model.RuntimeStateUnknownV1,
			model.RuntimeStateUnspecified,
			false,
			"",
		},
		{
			"V1 Terminating",
			"Terminating",
			model.RuntimeStateTerminatingV1,
			model.RuntimeStateCancelling,
			false,
			"",
		},
		{
			"V1 Ready",
			"Ready",
			model.RuntimeStateRunningV1,
			model.RuntimeStateRunning,
			false,
			"",
		},
		{
			"V1 Done",
			"Done",
			model.RuntimeStateSucceededV1,
			model.RuntimeStateSucceeded,
			false,
			"",
		},
		{
			"V1 wrong value",
			"wrong value",
			model.RuntimeStateUnknownV1,
			model.RuntimeStateUnspecified,
			false,
			"",
		},

		{
			"V2 RUNTIME_STATE_UNSPECIFIED",
			"RUNTIME_STATE_UNSPECIFIED",
			model.RuntimeStateUnknownV1,
			model.RuntimeStateUnspecified,
			false,
			"",
		},
		{
			"V2 RUNNING",
			"RUNNING",
			model.RuntimeStateRunningV1,
			model.RuntimeStateRunning,
			false,
			"",
		},
		{
			"V2 SUCCEEDED",
			"SUCCEEDED",
			model.RuntimeStateSucceededV1,
			model.RuntimeStateSucceeded,
			false,
			"",
		},
		{
			"V2 SKIPPED",
			"SKIPPED",
			model.RuntimeStateSkippedV1,
			model.RuntimeStateSkipped,
			false,
			"",
		},
		{
			"V2 CANCELED",
			"CANCELED",
			model.RuntimeStateFailedV1,
			model.RuntimeStateCanceled,
			false,
			"",
		},
		{
			"V2 PAUSED",
			"PAUSED",
			model.RuntimeStatePendingV1,
			model.RuntimeStatePaused,
			false,
			"",
		},
		{
			"V2 Empty",
			"",
			model.RuntimeStateUnknownV1,
			model.RuntimeStateUnspecified,
			false,
			"",
		},
		{
			"V2 PENDING",
			"PENDING",
			model.RuntimeStatePendingV1,
			model.RuntimeStatePending,
			false,
			"",
		},
		{
			"V2 RuntimeState_CANCELED",
			apiv2beta1.RuntimeState_CANCELED,
			model.RuntimeStateFailedV1,
			model.RuntimeStateCanceled,
			false,
			"",
		},
		{
			"nil",
			nil,
			model.RuntimeStateUnknownV1,
			model.RuntimeStateUnspecified,
			false,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toModelRuntimeState(tt.apiState)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Equal(t, "", string(got))
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
				assert.True(t, got.ToV2().IsValid())
				assert.Equal(t, tt.wantV1, got.ToExecutionPhase())
				assert.Equal(t, tt.wantV2, got.ToV2())
				assert.Equal(t, string(tt.wantV2), got.ToString())
			}
		})
	}
}

func Test_toApiRuntimeState(t *testing.T) {
	tests := []struct {
		name       string
		modelState model.RuntimeState
		want       apiv2beta1.RuntimeState
	}{
		{
			"v1 Error",
			model.RuntimeStateErrorV1,
			apiv2beta1.RuntimeState_FAILED,
		},
		{
			"v1 NO_STATUS",
			model.RuntimeState(model.LegacyStateNoStatus),
			apiv2beta1.RuntimeState_RUNTIME_STATE_UNSPECIFIED,
		},
		{
			"v1 succeeded",
			model.RuntimeStateSucceededV1,
			apiv2beta1.RuntimeState_SUCCEEDED,
		},
		{
			"v2 succeeded",
			model.RuntimeStateSucceeded,
			apiv2beta1.RuntimeState_SUCCEEDED,
		},
		{
			"v2 canceling",
			model.RuntimeStateCancelling,
			apiv2beta1.RuntimeState_CANCELING,
		},
		{
			"v2 paused",
			model.RuntimeStatePaused,
			apiv2beta1.RuntimeState_PAUSED,
		},
		{
			"v2 Unspecified",
			model.RuntimeStateUnspecified,
			apiv2beta1.RuntimeState_RUNTIME_STATE_UNSPECIFIED,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toApiRuntimeState(&tt.modelState); got != tt.want {
				t.Errorf("toApiRuntimeStateV1() = %v, want %v", tt.want, got)
			}
		})
	}
}

func Test_toModelRuntimeStatus(t *testing.T) {
	tests := []struct {
		name      string
		apiStatus *apiv2beta1.RuntimeStatus
		want      *model.RuntimeStatus
		wantErr   bool
		errMsg    string
	}{
		{
			"Empty",
			&apiv2beta1.RuntimeStatus{},
			&model.RuntimeStatus{
				UpdateTimeInSec: 0,
				State:           model.RuntimeStateUnspecified,
				Error:           nil,
			},
			false,
			"",
		},
		{
			"nil",
			nil,
			&model.RuntimeStatus{
				UpdateTimeInSec: 0,
				State:           "",
				Error:           nil,
			},
			false,
			"",
		},
		{
			"Error",
			&apiv2beta1.RuntimeStatus{
				Error: util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
			},
			&model.RuntimeStatus{
				UpdateTimeInSec: 0,
				State:           model.RuntimeStateUnspecified,
				Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value"))),
			},
			false,
			"",
		},
		{
			"Tipestamp",
			&apiv2beta1.RuntimeStatus{
				UpdateTime: &timestamppb.Timestamp{Seconds: 100},
			},
			&model.RuntimeStatus{
				UpdateTimeInSec: 100,
				State:           model.RuntimeStateUnspecified,
				Error:           nil,
			},
			false,
			"",
		},
		{
			"State",
			&apiv2beta1.RuntimeStatus{
				State: apiv2beta1.RuntimeState_CANCELING,
			},
			&model.RuntimeStatus{
				UpdateTimeInSec: 0,
				State:           model.RuntimeStateCancelling,
				Error:           nil,
			},
			false,
			"",
		},
		{
			"Full spec",
			&apiv2beta1.RuntimeStatus{
				UpdateTime: &timestamppb.Timestamp{Seconds: 100},
				State:      apiv2beta1.RuntimeState_CANCELING,
				Error:      util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
			},
			&model.RuntimeStatus{
				UpdateTimeInSec: 100,
				State:           model.RuntimeStateCancelling,
				Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value"))),
			},
			false,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toModelRuntimeStatus(tt.apiStatus)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, got)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_toModelRuntimeStatuses(t *testing.T) {
	arg := []*apiv2beta1.RuntimeStatus{
		{},
		nil,
		{
			Error: util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
		},
		{
			UpdateTime: &timestamppb.Timestamp{Seconds: 100},
		},
		{
			State: apiv2beta1.RuntimeState_CANCELING,
		},
		{
			UpdateTime: &timestamppb.Timestamp{Seconds: 100},
			State:      apiv2beta1.RuntimeState_CANCELING,
			Error:      util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
		},
	}
	expected := []*model.RuntimeStatus{
		{
			UpdateTimeInSec: 0,
			State:           model.RuntimeStateUnspecified,
			Error:           nil,
		},
		{
			UpdateTimeInSec: 0,
			State:           "",
			Error:           nil,
		},
		{
			UpdateTimeInSec: 0,
			State:           model.RuntimeStateUnspecified,
			Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value"))),
		},
		{
			UpdateTimeInSec: 100,
			State:           model.RuntimeStateUnspecified,
			Error:           nil,
		},
		{
			UpdateTimeInSec: 0,
			State:           model.RuntimeStateCancelling,
			Error:           nil,
		},
		{
			UpdateTimeInSec: 100,
			State:           model.RuntimeStateCancelling,
			Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value"))),
		},
	}
	got, err := toModelRuntimeStatuses(arg)
	assert.Nil(t, err)
	assert.Equal(t, expected, got)
}

func Test_toApiRuntimeStatus(t *testing.T) {
	tests := []struct {
		name        string
		modelStatus *model.RuntimeStatus
		want        *apiv2beta1.RuntimeStatus
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"full spec",
			&model.RuntimeStatus{
				UpdateTimeInSec: 100,
				State:           model.RuntimeStateCancelling,
				Error:           util.NewInvalidInputError("Invalid input: %s", "sample value"),
			},
			&apiv2beta1.RuntimeStatus{
				UpdateTime: &timestamppb.Timestamp{Seconds: 100},
				State:      apiv2beta1.RuntimeState_CANCELING,
				Error:      util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
			},
		},
		{
			"state",
			&model.RuntimeStatus{
				State: model.RuntimeStateCancelling,
			},
			&apiv2beta1.RuntimeStatus{
				State: apiv2beta1.RuntimeState_CANCELING,
			},
		},
		{
			"error",
			&model.RuntimeStatus{
				Error: util.NewInvalidInputError("Invalid input: %s", "sample value"),
			},
			&apiv2beta1.RuntimeStatus{
				Error: util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
			},
		},
		{
			"timestamp",
			&model.RuntimeStatus{
				UpdateTimeInSec: 100,
			},
			&apiv2beta1.RuntimeStatus{
				UpdateTime: &timestamppb.Timestamp{Seconds: 100},
			},
		},
		{
			"v1 error state",
			&model.RuntimeStatus{
				UpdateTimeInSec: 100,
				State:           model.RuntimeStateErrorV1,
				Error:           util.NewInvalidInputError("Invalid input: %s", "sample value"),
			},
			&apiv2beta1.RuntimeStatus{
				UpdateTime: &timestamppb.Timestamp{Seconds: 100},
				State:      apiv2beta1.RuntimeState_FAILED,
				Error:      util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
			},
		},
		{
			"v1 unknown state",
			&model.RuntimeStatus{
				UpdateTimeInSec: 100,
				State:           model.RuntimeStateUnknownV1,
				Error:           util.NewInvalidInputError("Invalid input: %s", "sample value"),
			},
			&apiv2beta1.RuntimeStatus{
				UpdateTime: &timestamppb.Timestamp{Seconds: 100},
				State:      apiv2beta1.RuntimeState_RUNTIME_STATE_UNSPECIFIED,
				Error:      util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
			},
		},
		{
			"v1 wrong state",
			&model.RuntimeStatus{
				UpdateTimeInSec: 100,
				State:           model.RuntimeState("WRONG STATE"),
				Error:           util.NewInvalidInputError("Invalid input: %s", "sample value"),
			},
			&apiv2beta1.RuntimeStatus{
				UpdateTime: &timestamppb.Timestamp{Seconds: 100},
				State:      apiv2beta1.RuntimeState_RUNTIME_STATE_UNSPECIFIED,
				Error:      util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
			},
		},
		{
			"v1 empty state",
			&model.RuntimeStatus{
				UpdateTimeInSec: 100,
				State:           model.RuntimeState(""),
				Error:           util.NewInvalidInputError("Invalid input: %s", "sample value"),
			},
			&apiv2beta1.RuntimeStatus{
				UpdateTime: &timestamppb.Timestamp{Seconds: 100},
				State:      apiv2beta1.RuntimeState_RUNTIME_STATE_UNSPECIFIED,
				Error:      util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toApiRuntimeStatus(tt.modelStatus)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_toApiRuntimeStatuses(t *testing.T) {
	arg := []*model.RuntimeStatus{
		nil,
		{
			UpdateTimeInSec: 100,
			State:           model.RuntimeStateCancelling,
			Error:           util.NewInvalidInputError("Invalid input: %s", "sample value"),
		},
		{
			UpdateTimeInSec: 100,
			State:           model.RuntimeStateErrorV1,
			Error:           util.NewInvalidInputError("Invalid input: %s", "sample value"),
		},
		{
			UpdateTimeInSec: 100,
			State:           model.RuntimeStateUnknownV1,
			Error:           util.NewInvalidInputError("Invalid input: %s", "sample value"),
		},
		{
			UpdateTimeInSec: 100,
			State:           model.RuntimeState("WRONG STATE"),
			Error:           util.NewInvalidInputError("Invalid input: %s", "sample value"),
		},
		{
			UpdateTimeInSec: 100,
			State:           model.RuntimeState(""),
			Error:           util.NewInvalidInputError("Invalid input: %s", "sample value"),
		},
	}
	expected := []*apiv2beta1.RuntimeStatus{
		nil,
		{
			UpdateTime: &timestamppb.Timestamp{Seconds: 100},
			State:      apiv2beta1.RuntimeState_CANCELING,
			Error:      util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
		},
		{
			UpdateTime: &timestamppb.Timestamp{Seconds: 100},
			State:      apiv2beta1.RuntimeState_FAILED,
			Error:      util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
		},
		{
			UpdateTime: &timestamppb.Timestamp{Seconds: 100},
			State:      apiv2beta1.RuntimeState_RUNTIME_STATE_UNSPECIFIED,
			Error:      util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
		},
		{
			UpdateTime: &timestamppb.Timestamp{Seconds: 100},
			State:      apiv2beta1.RuntimeState_RUNTIME_STATE_UNSPECIFIED,
			Error:      util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
		},
		{
			UpdateTime: &timestamppb.Timestamp{Seconds: 100},
			State:      apiv2beta1.RuntimeState_RUNTIME_STATE_UNSPECIFIED,
			Error:      util.ToRpcStatus(util.NewInvalidInputError("Invalid input: %s", "sample value")),
		},
	}
	got := toApiRuntimeStatuses(arg)
	assert.Equal(t, expected, got)
}

func TestToModelRun(t *testing.T) {
	tests := []struct {
		name    string
		arg     *apiv2beta1.Run
		want    *model.Run
		wantErr bool
		errMsg  string
	}{
		{
			"v2 full pipeline version",
			&apiv2beta1.Run{
				ExperimentId: "exp1",
				RunId:        "run1",
				DisplayName:  "name1",
				Description:  "this is a run",
				StorageState: apiv2beta1.Run_ARCHIVED,
				PipelineSource: &apiv2beta1.Run_PipelineVersionReference{
					PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
						PipelineId:        "p1",
						PipelineVersionId: "pv1",
					},
				},
				RuntimeConfig: &apiv2beta1.RuntimeConfig{
					Parameters: map[string]*structpb.Value{
						"param2": structpb.NewStringValue("world"),
					},
				},
				ServiceAccount: "sa1",
				CreatedAt:      &timestamppb.Timestamp{Seconds: 1},
				ScheduledAt:    &timestamppb.Timestamp{Seconds: 2},
				FinishedAt:     &timestamppb.Timestamp{Seconds: 3},
				State:          apiv2beta1.RuntimeState_FAILED,
				Error:          util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid")),
				RunDetails: &apiv2beta1.RunDetails{
					PipelineContextId:    10,
					PipelineRunContextId: 11,
				},
				RecurringRunId: "job1",
				StateHistory: []*apiv2beta1.RuntimeStatus{
					{
						UpdateTime: &timestamppb.Timestamp{Seconds: 9},
						State:      apiv2beta1.RuntimeState_FAILED,
						Error:      util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid")),
					},
				},
			},
			&model.Run{
				UUID:           "run1",
				ExperimentId:   "exp1",
				DisplayName:    "name1",
				Description:    "this is a run",
				ServiceAccount: "sa1",
				RecurringRunId: "job1",
				StorageState:   model.StorageStateArchived,
				PipelineSpec: model.PipelineSpec{
					RuntimeConfig: model.RuntimeConfig{
						Parameters: "{\"param2\":\"world\"}",
					},
					PipelineId:        "p1",
					PipelineVersionId: "pv1",
					PipelineName:      "pipelines/pv1",
				},
				RunDetails: model.RunDetails{
					State: model.RuntimeStateFailed,
					StateHistory: []*model.RuntimeStatus{
						{
							UpdateTimeInSec: 9,
							State:           model.RuntimeStateFailed,
							Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid"))),
						},
					},
					CreatedAtInSec:       1,
					ScheduledAtInSec:     2,
					FinishedAtInSec:      3,
					PipelineContextId:    0,
					PipelineRunContextId: 0,
				},
				ResourceReferences: nil,
				Namespace:          "",
				K8SName:            "",
			},
			false,
			"",
		},
		{
			"v2 full pipeline spec",
			&apiv2beta1.Run{
				ExperimentId: "exp1",
				RunId:        "run1",
				DisplayName:  "name1",
				Description:  "this is a run",
				StorageState: apiv2beta1.Run_ARCHIVED,
				PipelineSource: &apiv2beta1.Run_PipelineSpec{
					PipelineSpec: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"String":  structpb.NewStringValue("pv2"),
							"Boolean": structpb.NewBoolValue(false),
							"Number":  structpb.NewNumberValue(19.1),
							"Struct": structpb.NewStructValue(
								&structpb.Struct{
									Fields: map[string]*structpb.Value{
										"InnerNull": structpb.NewNullValue(),
										"InnerList": structpb.NewListValue(
											&structpb.ListValue{
												Values: []*structpb.Value{
													structpb.NewStringValue("a"),
													structpb.NewStringValue("b"),
												},
											},
										),
									},
								},
							),
						},
					},
				},
				RuntimeConfig: &apiv2beta1.RuntimeConfig{
					Parameters: map[string]*structpb.Value{
						"param2": structpb.NewStringValue("world"),
					},
				},
				ServiceAccount: "sa1",
				CreatedAt:      &timestamppb.Timestamp{Seconds: 1},
				ScheduledAt:    &timestamppb.Timestamp{Seconds: 2},
				FinishedAt:     &timestamppb.Timestamp{Seconds: 3},
				State:          apiv2beta1.RuntimeState_RUNNING,
				RunDetails: &apiv2beta1.RunDetails{
					PipelineContextId:    10,
					PipelineRunContextId: 11,
				},
				RecurringRunId: "job1",
				StateHistory: []*apiv2beta1.RuntimeStatus{
					{
						UpdateTime: &timestamppb.Timestamp{Seconds: 9},
						State:      apiv2beta1.RuntimeState_RUNNING,
					},
				},
			},
			&model.Run{
				UUID:           "run1",
				ExperimentId:   "exp1",
				DisplayName:    "name1",
				Description:    "this is a run",
				ServiceAccount: "sa1",
				RecurringRunId: "job1",
				StorageState:   model.StorageStateArchived,
				PipelineSpec: model.PipelineSpec{
					RuntimeConfig: model.RuntimeConfig{
						Parameters: "{\"param2\":\"world\"}",
					},
					PipelineSpecManifest: "Boolean: false\nNumber: 19.1\nString: pv2\nStruct:\n  InnerList:\n  - a\n  - b\n  InnerNull: null\n",
				},
				RunDetails: model.RunDetails{
					State: model.RuntimeStateRunning,
					StateHistory: []*model.RuntimeStatus{
						{
							UpdateTimeInSec: 9,
							State:           model.RuntimeStateRunning,
						},
					},
					CreatedAtInSec:       1,
					ScheduledAtInSec:     2,
					FinishedAtInSec:      3,
					PipelineContextId:    0,
					PipelineRunContextId: 0,
				},
				ResourceReferences: nil,
				Namespace:          "",
				K8SName:            "",
			},
			false,
			"",
		},
		{ // all fields are same as "v2 full pipeline version except invalid ExperimentId
			"v2 ExperimentId overflow",
			&apiv2beta1.Run{
				ExperimentId: strings.Repeat("e", 65),
				RunId:        "run1",
				DisplayName:  "name1",
				Description:  "this is a run",
				StorageState: apiv2beta1.Run_ARCHIVED,
				PipelineSource: &apiv2beta1.Run_PipelineVersionReference{
					PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
						PipelineId:        "p1",
						PipelineVersionId: "pv1",
					},
				},
				RuntimeConfig: &apiv2beta1.RuntimeConfig{
					Parameters: map[string]*structpb.Value{
						"param2": structpb.NewStringValue("world"),
					},
				},
				ServiceAccount: "sa1",
				CreatedAt:      &timestamppb.Timestamp{Seconds: 1},
				ScheduledAt:    &timestamppb.Timestamp{Seconds: 2},
				FinishedAt:     &timestamppb.Timestamp{Seconds: 3},
				State:          apiv2beta1.RuntimeState_FAILED,
				Error:          util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid")),
				RunDetails: &apiv2beta1.RunDetails{
					PipelineContextId:    10,
					PipelineRunContextId: 11,
				},
				RecurringRunId: "job1",
				StateHistory: []*apiv2beta1.RuntimeStatus{
					{
						UpdateTime: &timestamppb.Timestamp{Seconds: 9},
						State:      apiv2beta1.RuntimeState_FAILED,
						Error:      util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid")),
					},
				},
			},
			nil,
			true,
			"Run.ExperimentId length cannot exceed 64",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toModelRun(tt.arg)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, got)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_toApiRun(t *testing.T) {
	tests := []struct {
		name    string
		arg     *model.Run
		want    *apiv2beta1.Run
		wantErr bool
		errMsg  string
	}{
		{
			"V1 no refs",
			&model.Run{
				UUID:           "run123",
				K8SName:        "name123",
				StorageState:   model.StorageStateArchived,
				DisplayName:    "displayName123",
				Description:    "this is run",
				Namespace:      "ns123",
				RecurringRunId: "job123",
				ExperimentId:   "exp123",
				ServiceAccount: "sa1",
				RunDetails: model.RunDetails{
					CreatedAtInSec:          1,
					ScheduledAtInSec:        2,
					FinishedAtInSec:         3,
					Conditions:              "running",
					WorkflowRuntimeManifest: "workflow123",
				},
				PipelineSpec: model.PipelineSpec{
					WorkflowSpecManifest: "Name: manifest\nVersion: v1",
					RuntimeConfig: model.RuntimeConfig{
						Parameters:   "{\"param2\":\"world\",\"param3\":true,\"param4\":[1,2,3],\"param5\":12,\"param6\":{\"structParam1\":\"hello\",\"structParam2\":32}}",
						PipelineRoot: "model-pipeline-root",
					},
				},
			},
			&apiv2beta1.Run{
				ExperimentId:   "exp123",
				RunId:          "run123",
				DisplayName:    "displayName123",
				StorageState:   apiv2beta1.Run_ARCHIVED,
				Description:    "this is run",
				RecurringRunId: "job123",
				ServiceAccount: "sa1",
				State:          apiv2beta1.RuntimeState_RUNNING,
				PipelineSource: &apiv2beta1.Run_PipelineSpec{
					PipelineSpec: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"Name":    structpb.NewStringValue("manifest"),
							"Version": structpb.NewStringValue("v1"),
						},
					},
				},
				CreatedAt:   &timestamppb.Timestamp{Seconds: 1},
				ScheduledAt: &timestamppb.Timestamp{Seconds: 2},
				FinishedAt:  &timestamppb.Timestamp{Seconds: 3},
				RuntimeConfig: &apiv2beta1.RuntimeConfig{
					Parameters: map[string]*structpb.Value{
						"param2": structpb.NewStringValue("world"),
						"param3": structpb.NewBoolValue(true),
						"param4": structpb.NewListValue(
							&structpb.ListValue{
								Values: []*structpb.Value{
									structpb.NewNumberValue(1),
									structpb.NewNumberValue(2),
									structpb.NewNumberValue(3),
								},
							},
						),
						"param5": structpb.NewNumberValue(12),
						"param6": structpb.NewStructValue(
							&structpb.Struct{
								Fields: map[string]*structpb.Value{
									"structParam1": structpb.NewStringValue("hello"),
									"structParam2": structpb.NewNumberValue(32),
								},
							},
						),
					},
					PipelineRoot: "model-pipeline-root",
				},
			},
			false,
			"",
		},
		{
			"V1 refs",
			&model.Run{
				UUID:           "run123",
				K8SName:        "name123",
				StorageState:   model.StorageStateArchived,
				DisplayName:    "displayName123",
				Description:    "this is run",
				ServiceAccount: "sa1",
				RunDetails: model.RunDetails{
					CreatedAtInSec:          1,
					ScheduledAtInSec:        2,
					FinishedAtInSec:         3,
					Conditions:              "running",
					WorkflowRuntimeManifest: "workflow123",
				},
				PipelineSpec: model.PipelineSpec{
					PipelineId:        "p1",
					PipelineVersionId: "pv1",
				},
				ResourceReferences: []*model.ResourceReference{
					{ResourceType: model.ExperimentResourceType, ReferenceUUID: "exp123"},
					{ResourceType: model.JobResourceType, ReferenceUUID: "job123"},
					{ResourceType: model.NamespaceResourceType, ReferenceUUID: "name_space"},
				},
			},
			&apiv2beta1.Run{
				RunId:          "run123",
				DisplayName:    "displayName123",
				StorageState:   apiv2beta1.Run_ARCHIVED,
				Description:    "this is run",
				ServiceAccount: "sa1",
				State:          apiv2beta1.RuntimeState_RUNNING,
				PipelineSource: &apiv2beta1.Run_PipelineVersionReference{
					PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
						PipelineId:        "p1",
						PipelineVersionId: "pv1",
					},
				},
				CreatedAt:   &timestamppb.Timestamp{Seconds: 1},
				ScheduledAt: &timestamppb.Timestamp{Seconds: 2},
				FinishedAt:  &timestamppb.Timestamp{Seconds: 3},
			},
			false,
			"",
		},
		{
			"v2 full spec",
			&model.Run{
				UUID:           "run1",
				ExperimentId:   "exp1",
				DisplayName:    "name1",
				Description:    "this is a run",
				ServiceAccount: "sa1",
				RecurringRunId: "job1",
				StorageState:   model.StorageStateArchived,
				PipelineSpec: model.PipelineSpec{
					RuntimeConfig: model.RuntimeConfig{
						Parameters: "{\"param2\":\"world\"}",
					},
					PipelineSpecManifest: "Boolean: false\nNumber: 19.1\nString: pv2\nStruct:\n  InnerList:\n  - a\n  - b\n  InnerNull: null\n",
				},
				RunDetails: model.RunDetails{
					State: model.RuntimeStateFailed,
					StateHistory: []*model.RuntimeStatus{
						{
							UpdateTimeInSec: 9,
							State:           model.RuntimeStateFailed,
							Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid"))),
						},
					},
					CreatedAtInSec:       1,
					ScheduledAtInSec:     2,
					FinishedAtInSec:      3,
					PipelineContextId:    10,
					PipelineRunContextId: 11,
				},
				ResourceReferences: nil,
				Namespace:          "",
				K8SName:            "",
			},
			&apiv2beta1.Run{
				ExperimentId: "exp1",
				RunId:        "run1",
				DisplayName:  "name1",
				Description:  "this is a run",
				StorageState: apiv2beta1.Run_ARCHIVED,
				PipelineSource: &apiv2beta1.Run_PipelineSpec{
					PipelineSpec: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"String":  structpb.NewStringValue("pv2"),
							"Boolean": structpb.NewBoolValue(false),
							"Number":  structpb.NewNumberValue(19.1),
							"Struct": structpb.NewStructValue(
								&structpb.Struct{
									Fields: map[string]*structpb.Value{
										"InnerNull": structpb.NewNullValue(),
										"InnerList": structpb.NewListValue(
											&structpb.ListValue{
												Values: []*structpb.Value{
													structpb.NewStringValue("a"),
													structpb.NewStringValue("b"),
												},
											},
										),
									},
								},
							),
						},
					},
				},
				RuntimeConfig: &apiv2beta1.RuntimeConfig{
					Parameters: map[string]*structpb.Value{
						"param2": structpb.NewStringValue("world"),
					},
				},
				ServiceAccount: "sa1",
				CreatedAt:      &timestamppb.Timestamp{Seconds: 1},
				ScheduledAt:    &timestamppb.Timestamp{Seconds: 2},
				FinishedAt:     &timestamppb.Timestamp{Seconds: 3},
				State:          apiv2beta1.RuntimeState_FAILED,
				RunDetails: &apiv2beta1.RunDetails{
					PipelineContextId:    10,
					PipelineRunContextId: 11,
				},
				RecurringRunId: "job1",
				StateHistory: []*apiv2beta1.RuntimeStatus{
					{
						UpdateTime: &timestamppb.Timestamp{Seconds: 9},
						State:      apiv2beta1.RuntimeState_FAILED,
						Error:      util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid")),
					},
				},
			},
			false,
			"",
		},
		{
			"v2 error runtime config",
			&model.Run{
				UUID:           "run1",
				ExperimentId:   "exp1",
				DisplayName:    "name1",
				Description:    "this is a run",
				ServiceAccount: "sa1",
				RecurringRunId: "job1",
				StorageState:   model.StorageStateArchived,
				PipelineSpec: model.PipelineSpec{
					RuntimeConfig: model.RuntimeConfig{
						Parameters: "{\"param2\":\"world\"}}",
					},
					PipelineSpecManifest: "Boolean: false\nNumber: 19.1\nString: pv2\nStruct:\n  InnerList:\n  - a\n  - b\n  InnerNull: null\n",
				},
				RunDetails: model.RunDetails{
					State: model.RuntimeStateCancelling,
					StateHistory: []*model.RuntimeStatus{
						{
							UpdateTimeInSec: 9,
							State:           model.RuntimeStateCancelling,
						},
					},
					CreatedAtInSec:       1,
					ScheduledAtInSec:     2,
					FinishedAtInSec:      3,
					PipelineContextId:    10,
					PipelineRunContextId: 11,
					TaskDetails:          []*model.Task{},
				},
				ResourceReferences: nil,
				Namespace:          "",
				K8SName:            "",
			},
			&apiv2beta1.Run{
				RunId:          "run1",
				ExperimentId:   "exp1",
				DisplayName:    "name1",
				Description:    "this is a run",
				ServiceAccount: "sa1",
				RecurringRunId: "job1",
				StorageState:   apiv2beta1.Run_ARCHIVED,
				State:          apiv2beta1.RuntimeState_CANCELING,
				StateHistory: []*apiv2beta1.RuntimeStatus{
					{
						UpdateTime: &timestamppb.Timestamp{Seconds: 9},
						State:      apiv2beta1.RuntimeState_CANCELING,
					},
				},
				CreatedAt:   &timestamppb.Timestamp{Seconds: 1},
				ScheduledAt: &timestamppb.Timestamp{Seconds: 2},
				FinishedAt:  &timestamppb.Timestamp{Seconds: 3},
				RunDetails: &apiv2beta1.RunDetails{ //nolint:staticcheck // Verify backward-compatible legacy run details.
					PipelineContextId:    10,
					PipelineRunContextId: 11,
				},
				PipelineSource: &apiv2beta1.Run_PipelineSpec{
					PipelineSpec: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"Boolean": structpb.NewBoolValue(false),
							"Number":  structpb.NewNumberValue(19.1),
							"String":  structpb.NewStringValue("pv2"),
							"Struct": structpb.NewStructValue(
								&structpb.Struct{
									Fields: map[string]*structpb.Value{
										"InnerNull": structpb.NewNullValue(),
										"InnerList": structpb.NewListValue(
											&structpb.ListValue{
												Values: []*structpb.Value{
													structpb.NewStringValue("a"),
													structpb.NewStringValue("b"),
												},
											},
										),
									},
								},
							),
						},
					},
				},
			},
			true,
			"Failed to parse runtime config",
		},
		{
			"v2 error pipeline source",
			&model.Run{
				UUID:           "run1",
				ExperimentId:   "exp1",
				DisplayName:    "name1",
				Description:    "this is a run",
				ServiceAccount: "sa1",
				RecurringRunId: "job1",
				StorageState:   model.StorageStateArchived,
				PipelineSpec: model.PipelineSpec{
					RuntimeConfig: model.RuntimeConfig{
						Parameters: "{\"param2\":\"world\"}",
					},
				},
				RunDetails: model.RunDetails{
					State: model.RuntimeStatePaused,
					StateHistory: []*model.RuntimeStatus{
						{
							UpdateTimeInSec: 9,
							State:           model.RuntimeStatePaused,
						},
					},
					CreatedAtInSec:       1,
					ScheduledAtInSec:     2,
					FinishedAtInSec:      3,
					PipelineContextId:    10,
					PipelineRunContextId: 11,
					TaskDetails:          []*model.Task{},
				},
				ResourceReferences: nil,
				Metrics:            nil,
				Namespace:          "",
				K8SName:            "",
			},
			&apiv2beta1.Run{
				RunId:          "run1",
				ExperimentId:   "exp1",
				DisplayName:    "name1",
				Description:    "this is a run",
				ServiceAccount: "sa1",
				RecurringRunId: "job1",
				StorageState:   apiv2beta1.Run_ARCHIVED,
				State:          apiv2beta1.RuntimeState_PAUSED,
				StateHistory: []*apiv2beta1.RuntimeStatus{
					{
						UpdateTime: &timestamppb.Timestamp{Seconds: 9},
						State:      apiv2beta1.RuntimeState_PAUSED,
					},
				},
				CreatedAt:   &timestamppb.Timestamp{Seconds: 1},
				ScheduledAt: &timestamppb.Timestamp{Seconds: 2},
				FinishedAt:  &timestamppb.Timestamp{Seconds: 3},
				RunDetails: &apiv2beta1.RunDetails{ //nolint:staticcheck // Verify backward-compatible legacy run details.
					PipelineContextId:    10,
					PipelineRunContextId: 11,
				},
				RuntimeConfig: &apiv2beta1.RuntimeConfig{
					Parameters: map[string]*structpb.Value{
						"param2": structpb.NewStringValue("world"),
					},
				},
			},
			true,
			"Failed to convert internal run representation to its API counterpart due to missing pipeline source",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toApiRun(tt.arg)
			if tt.wantErr {
				assert.Contains(t, got.Error.Message, tt.errMsg)
				if tt.want.GetRuntimeConfig() != nil && got.GetRuntimeConfig() != nil {
					wp := tt.want.GetRuntimeConfig().GetParameters()
					gp := got.GetRuntimeConfig().GetParameters()
					for k := range wp {
						wp1, err := wp[k].MarshalJSON()
						assert.Nil(t, err)
						gp1, err := gp[k].MarshalJSON()
						assert.Nil(t, err)
						assert.Equal(t, wp1, gp1)
					}
					tt.want.RuntimeConfig.Parameters = got.RuntimeConfig.Parameters
				}
				if tt.want.GetPipelineSpec() != nil && got.GetPipelineSpec() != nil {
					w, err := tt.want.GetPipelineSpec().MarshalJSON()
					assert.Nil(t, err)
					g, err := got.GetPipelineSpec().MarshalJSON()
					assert.Nil(t, err)
					assert.Equal(t, w, g)
					tt.want.PipelineSource = got.GetPipelineSource()
				}
				tt.want.Error = got.GetError()
				assert.Equal(t, tt.want, got)
			} else {
				if tt.want.GetPipelineSpec() != nil {
					w, err := tt.want.GetPipelineSpec().MarshalJSON()
					assert.Nil(t, err)
					g, err := got.GetPipelineSpec().MarshalJSON()
					assert.Nil(t, err)
					assert.Equal(t, w, g)
					tt.want.PipelineSource = got.GetPipelineSource()
				}
				if tt.want.GetRuntimeConfig() != nil {
					wp := tt.want.GetRuntimeConfig().GetParameters()
					gp := got.GetRuntimeConfig().GetParameters()
					for k := range wp {
						wp1, err := wp[k].MarshalJSON()
						assert.Nil(t, err)
						gp1, err := gp[k].MarshalJSON()
						assert.Nil(t, err)
						assert.Equal(t, wp1, gp1)
					}
					for k := range gp {
						wp1, err := wp[k].MarshalJSON()
						assert.Nil(t, err)
						gp1, err := gp[k].MarshalJSON()
						assert.Nil(t, err)
						assert.Equal(t, wp1, gp1)
					}
					tt.want.RuntimeConfig.Parameters = got.RuntimeConfig.Parameters
				}
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestPluginsInputToJSON(t *testing.T) {
	t.Run("nil map returns nil", func(t *testing.T) {
		got, err := pluginsInputToJSON(nil)
		require.NoError(t, err)
		assert.Nil(t, got, "nil input should produce nil *string, not empty string")
	})

	t.Run("empty map returns nil", func(t *testing.T) {
		got, err := pluginsInputToJSON(map[string]*structpb.Struct{})
		require.NoError(t, err)
		assert.Nil(t, got, "empty input should produce nil *string, not empty string")
	})

	t.Run("single key round-trips", func(t *testing.T) {
		input := map[string]*structpb.Struct{
			"mlflow": {Fields: map[string]*structpb.Value{
				"experiment_name": structpb.NewStringValue(testPluginsExperimentName),
			}},
		}
		got, err := pluginsInputToJSON(input)
		require.NoError(t, err)
		require.NotNil(t, got)
		parsed, err := jsonToPluginsInput(got)
		require.NoError(t, err)
		require.Len(t, parsed, 1)
		require.Contains(t, parsed, "mlflow")
		assert.Equal(t, input["mlflow"].Fields, parsed["mlflow"].Fields)
	})

	t.Run("multiple keys round-trip", func(t *testing.T) {
		input := map[string]*structpb.Struct{
			"mlflow": {Fields: map[string]*structpb.Value{
				"experiment_name": structpb.NewStringValue(testPluginsExperimentName),
			}},
			"other": {Fields: map[string]*structpb.Value{
				"key": structpb.NewBoolValue(true),
			}},
		}
		got, err := pluginsInputToJSON(input)
		require.NoError(t, err)
		require.NotNil(t, got)
		parsed, err := jsonToPluginsInput(got)
		require.NoError(t, err)
		require.Len(t, parsed, len(input))
		for k, v := range input {
			require.Contains(t, parsed, k)
			assert.Equal(t, v.Fields, parsed[k].Fields)
		}
	})
}

func TestJSONToPluginsInput(t *testing.T) {
	tests := []struct {
		name    string
		input   *string
		wantNil bool
		wantErr bool
	}{
		{
			name:    "nil pointer",
			input:   nil,
			wantNil: true,
		},
		{
			name:    "empty string",
			input:   strPtr(""),
			wantNil: true,
		},
		{
			name:  "valid JSON",
			input: strPtr(`{"mlflow":{"experiment_name":"` + testPluginsExperimentName + `"}}`),
		},
		{
			name:    "malformed JSON",
			input:   strPtr(`{not valid`),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonToPluginsInput(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Len(t, got, 1)
			require.Contains(t, got, "mlflow")
			assert.Equal(t, testPluginsExperimentName, got["mlflow"].Fields["experiment_name"].GetStringValue())
		})
	}
}

func TestToApiExperimentStorageState(t *testing.T) {
	tests := []struct {
		name     string
		state    model.StorageState
		expected apiv2beta1.Experiment_StorageState
	}{
		{"empty string defaults to unspecified", model.StorageState(""), apiv2beta1.Experiment_STORAGE_STATE_UNSPECIFIED},
		{"archived v2", model.StorageStateArchived, apiv2beta1.Experiment_ARCHIVED},
		{"archived v1", model.StorageStateArchived.ToV2(), apiv2beta1.Experiment_ARCHIVED},
		{"available v2", model.StorageStateAvailable, apiv2beta1.Experiment_AVAILABLE},
		{"available v1", model.StorageStateAvailable.ToV2(), apiv2beta1.Experiment_AVAILABLE},
		{"unspecified v2", model.StorageStateUnspecified, apiv2beta1.Experiment_STORAGE_STATE_UNSPECIFIED},
		{"unspecified v1", model.StorageStateUnspecified.ToV2(), apiv2beta1.Experiment_STORAGE_STATE_UNSPECIFIED},
		{"unknown defaults to unspecified", model.StorageState("UNKNOWN"), apiv2beta1.Experiment_STORAGE_STATE_UNSPECIFIED},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := testCase.state
			result := toApiExperimentStorageState(&state)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestPluginsOutputToJSON(t *testing.T) {
	t.Run("nil map returns nil", func(t *testing.T) {
		got, err := pluginsOutputToJSON(nil)
		require.NoError(t, err)
		assert.Nil(t, got, "nil input should produce nil *string, not empty string")
	})

	t.Run("empty map returns nil", func(t *testing.T) {
		got, err := pluginsOutputToJSON(map[string]*apiv2beta1.PluginOutput{})
		require.NoError(t, err)
		assert.Nil(t, got, "empty input should produce nil *string, not empty string")
	})

	t.Run("with entries and state round-trips", func(t *testing.T) {
		input := map[string]*apiv2beta1.PluginOutput{
			"mlflow": {
				Entries: map[string]*apiv2beta1.MetadataValue{
					"run_url": {
						Value:      structpb.NewStringValue("https://mlflow.example.com/runs/abc"),
						RenderType: apiv2beta1.MetadataValue_URL.Enum(),
					},
					"experiment_id": {
						Value: structpb.NewStringValue("42"),
					},
				},
				State:        apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
				StateMessage: "MLflow run created",
			},
			"other": {
				State:        apiv2beta1.PluginState_PLUGIN_RUNNING,
				StateMessage: "in progress",
			},
		}
		got, err := pluginsOutputToJSON(input)
		require.NoError(t, err)
		require.NotNil(t, got)
		parsed, err := jsonToPluginsOutput(got)
		require.NoError(t, err)
		require.Len(t, parsed, len(input))
		for k, v := range input {
			require.Contains(t, parsed, k)
			assert.Equal(t, v.State, parsed[k].State)
			assert.Equal(t, v.StateMessage, parsed[k].StateMessage)
			require.Len(t, parsed[k].Entries, len(v.Entries))
			for ek, ev := range v.Entries {
				require.Contains(t, parsed[k].Entries, ek)
				assert.Equal(t, ev.Value.GetStringValue(), parsed[k].Entries[ek].Value.GetStringValue())
				assert.Equal(t, ev.RenderType, parsed[k].Entries[ek].RenderType)
			}
		}
	})
}

func TestJSONToPluginsOutput(t *testing.T) {
	tests := []struct {
		name    string
		input   *string
		wantNil bool
		wantErr bool
	}{
		{
			name:    "nil pointer",
			input:   nil,
			wantNil: true,
		},
		{
			name:    "empty string",
			input:   strPtr(""),
			wantNil: true,
		},
		{
			name:    "malformed JSON",
			input:   strPtr(`{broken`),
			wantErr: true,
		},
		{
			name:  "valid JSON with enum fields",
			input: strPtr(`{"mlflow":{"entries":{"run_url":{"value":"https://mlflow.example.com","renderType":"URL"}},"state":"PLUGIN_SUCCEEDED","stateMessage":"ok"}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonToPluginsOutput(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Len(t, got, 1)
			require.Contains(t, got, "mlflow")
			assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, got["mlflow"].State)
			assert.Equal(t, "ok", got["mlflow"].StateMessage)
			require.Len(t, got["mlflow"].Entries, 1)
			require.Contains(t, got["mlflow"].Entries, "run_url")
			assert.Equal(t, "https://mlflow.example.com", got["mlflow"].Entries["run_url"].Value.GetStringValue())
		})
	}
}

func TestValidatePluginsOutput(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]*apiv2beta1.PluginOutput
		wantErr bool
	}{
		{
			name:  "nil map",
			input: nil,
		},
		{
			name:  "empty map",
			input: map[string]*apiv2beta1.PluginOutput{},
		},
		{
			name: "valid http URL content type",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value:      structpb.NewStringValue("http://example.com/run/1"),
							RenderType: apiv2beta1.MetadataValue_URL.Enum(),
						},
					},
				},
			},
		},
		{
			name: "valid https URL content type",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value:      structpb.NewStringValue("https://example.com/run/1"),
							RenderType: apiv2beta1.MetadataValue_URL.Enum(),
						},
					},
				},
			},
		},
		{
			name: "plain string without scheme",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_id": {
							Value: structpb.NewStringValue("abc123"),
						},
					},
				},
			},
		},
		{
			name: "javascript scheme without URL content type is allowed",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value: structpb.NewStringValue(testPluginsUnsafeJavaScriptURL),
						},
					},
				},
			},
		},
		{
			name: "data scheme without URL content type is allowed",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value: structpb.NewStringValue("data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg=="),
						},
					},
				},
			},
		},
		{
			name: "vbscript scheme without URL content type is allowed",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value: structpb.NewStringValue("vbscript:msgbox(1)"),
						},
					},
				},
			},
		},
		{
			name: "url content type with ftp rejected",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value:      structpb.NewStringValue("ftp://example.com/run/1"),
							RenderType: apiv2beta1.MetadataValue_URL.Enum(),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "url content type with malformed URL rejected",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value:      structpb.NewStringValue("http://%"),
							RenderType: apiv2beta1.MetadataValue_URL.Enum(),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "url content type with empty string rejected",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value:      structpb.NewStringValue(""),
							RenderType: apiv2beta1.MetadataValue_URL.Enum(),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "url content type with whitespace-only string rejected",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value:      structpb.NewStringValue("   "),
							RenderType: apiv2beta1.MetadataValue_URL.Enum(),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "url content type with javascript rejected",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value:      structpb.NewStringValue(testPluginsUnsafeJavaScriptURL),
							RenderType: apiv2beta1.MetadataValue_URL.Enum(),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "url content type with mixed-case javascript and leading spaces rejected",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value:      structpb.NewStringValue("  JaVaScRiPt:alert(1)"),
							RenderType: apiv2beta1.MetadataValue_URL.Enum(),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "mixed valid and invalid entries",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_id": {
							Value: structpb.NewStringValue("abc123"),
						},
						"run_url": {
							Value:      structpb.NewStringValue(testPluginsUnsafeJavaScriptURL),
							RenderType: apiv2beta1.MetadataValue_URL.Enum(),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "url content type with non-string value rejected",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value:      structpb.NewNumberValue(42),
							RenderType: apiv2beta1.MetadataValue_URL.Enum(),
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePluginsOutput(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
func Test_toApiRun_PreservesTopLevelStateOnTaskConversionError(t *testing.T) {
	run := &model.Run{
		UUID:           "run-task-error",
		ExperimentId:   "exp-task-error",
		DisplayName:    "run with bad task",
		StorageState:   model.StorageStateArchived,
		RecurringRunId: "job-1",
		ServiceAccount: "pipeline-runner",
		TaskCount:      1,
		RunDetails: model.RunDetails{
			State:            model.RuntimeStatePending,
			CreatedAtInSec:   10,
			ScheduledAtInSec: 11,
			FinishedAtInSec:  12,
		},
		PipelineSpec: model.PipelineSpec{
			PipelineId:        "pipeline-1",
			PipelineVersionId: "pipeline-version-1",
		},
		Tasks: []*model.Task{
			{
				UUID:             "task-1",
				RunUUID:          "run-task-error",
				Name:             "bad-task",
				DisplayName:      "bad-task",
				Namespace:        "kubeflow",
				CreatedAtInSec:   9,
				State:            model.TaskStatus(apiv2beta1.PipelineTask_RUNNING),
				Type:             model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
				OutputParameters: model.JSONSlice{"not-a-valid-task-output-parameter"},
			},
		},
	}

	got := toApiRun(run)
	if assert.NotNil(t, got) {
		assert.Equal(t, run.UUID, got.GetRunId())
		assert.Equal(t, run.ExperimentId, got.GetExperimentId())
		assert.Equal(t, apiv2beta1.RuntimeState_PENDING, got.GetState())
		assert.Equal(t, apiv2beta1.Run_ARCHIVED, got.GetStorageState())
		assert.Equal(t, int32(1), got.GetTaskCount())
		assert.Nil(t, got.GetTasks())
		if assert.NotNil(t, got.GetError()) {
			assert.Contains(t, got.GetError().GetMessage(), "Failed to convert task to API format")
		}
	}
}

func TestToApiPipelineVersion_OnlyPipelineSpecURI(t *testing.T) {
	pv := &model.PipelineVersion{
		UUID:            "version-1",
		Name:            "v1",
		DisplayName:     "v1",
		CreatedAtInSec:  100,
		PipelineId:      "pipeline-1",
		PipelineSpecURI: "http://package/v1",
	}
	result := toApiPipelineVersion(pv)
	assert.Empty(t, result.CodeSourceUrl)
	assert.NotNil(t, result.PackageUrl)
	assert.Equal(t, "http://package/v1", result.PackageUrl.PipelineUrl)
}

func TestToApiPipelineVersion_OnlyCodeSourceUrl(t *testing.T) {
	pv := &model.PipelineVersion{
		UUID:           "version-1",
		Name:           "v1",
		DisplayName:    "v1",
		CreatedAtInSec: 100,
		PipelineId:     "pipeline-1",
		CodeSourceUrl:  "http://repo/v1",
	}
	result := toApiPipelineVersion(pv)
	assert.Equal(t, "http://repo/v1", result.CodeSourceUrl)
	assert.Nil(t, result.PackageUrl)
}

func TestToApiPipelineVersions_Empty(t *testing.T) {
	result := toApiPipelineVersions([]*model.PipelineVersion{})
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestToApiPipelineVersions_SingleVersion(t *testing.T) {
	versions := []*model.PipelineVersion{
		{
			UUID:           "version-1",
			Name:           "v1",
			DisplayName:    "v1",
			CreatedAtInSec: 100,
			PipelineId:     "pipeline-1",
		},
	}
	result := toApiPipelineVersions(versions)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "version-1", result[0].PipelineVersionId)
	assert.Equal(t, "v1", result[0].DisplayName)
}

func TestToApiPipelineVersions_MultipleVersions(t *testing.T) {
	versions := []*model.PipelineVersion{
		{
			UUID:           "version-1",
			Name:           "v1",
			CreatedAtInSec: 100,
			PipelineId:     "pipeline-1",
		},
		{
			UUID:           "version-2",
			Name:           "v2",
			CreatedAtInSec: 200,
			PipelineId:     "pipeline-1",
		},
	}
	result := toApiPipelineVersions(versions)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, "version-1", result[0].PipelineVersionId)
	assert.Equal(t, "version-2", result[1].PipelineVersionId)
}

func TestToPipelineSpecRuntimeConfig_Nil(t *testing.T) {
	result := toPipelineSpecRuntimeConfig(nil)
	assert.NotNil(t, result)
	assert.Empty(t, result.ParameterValues)
	assert.Empty(t, result.GcsOutputDirectory)
}

func TestToPipelineSpecRuntimeConfig_WithParams(t *testing.T) {
	serializedParameters := `{"param1":"value1","param2":"value2"}`
	runtimeConfig := &model.RuntimeConfig{
		Parameters:   model.LargeText(serializedParameters),
		PipelineRoot: "gs://my-bucket/pipeline-root",
	}
	result := toPipelineSpecRuntimeConfig(runtimeConfig)
	assert.NotNil(t, result)
	assert.Equal(t, "gs://my-bucket/pipeline-root", result.GcsOutputDirectory)
	assert.NotNil(t, result.ParameterValues)
	assert.Equal(t, 2, len(result.ParameterValues))
}

func TestToPipelineSpecRuntimeConfig_InvalidJSON(t *testing.T) {
	runtimeConfig := &model.RuntimeConfig{
		Parameters:   model.LargeText("not valid json"),
		PipelineRoot: "gs://my-bucket/pipeline-root",
	}
	result := toPipelineSpecRuntimeConfig(runtimeConfig)
	// toMapProtoStructParameters returns nil on invalid JSON that also fails
	// v1 parameter parsing, causing toPipelineSpecRuntimeConfig to return nil.
	assert.Nil(t, result)
}

func TestValidatePluginsInputLimits(t *testing.T) {
	tooLongPayloadValue := strings.Repeat("a", 55000)

	tests := []struct {
		name            string
		input           map[string]*structpb.Struct
		wantErrContains string
	}{
		{
			name:  "nil map",
			input: nil,
		},
		{
			name:  "empty map",
			input: map[string]*structpb.Struct{},
		},
		{
			name: "accepts multiple small plugin input payloads",
			input: map[string]*structpb.Struct{
				"plugin-0": {Fields: map[string]*structpb.Value{"k": structpb.NewStringValue("ok")}},
				"plugin-1": {Fields: map[string]*structpb.Value{"k": structpb.NewStringValue("ok")}},
			},
		},
		{
			name:            "rejects plugin input map with too many keys",
			input:           createPluginInputMapWithNKeys(common.DefaultPluginMaxKeys + 1),
			wantErrContains: pluginErrPluginsInputTooManyKeys,
		},
		{
			name: "rejects plugin input entry exceeding per-plugin size",
			input: map[string]*structpb.Struct{
				"plugin-0": {
					Fields: map[string]*structpb.Value{
						"k": structpb.NewStringValue(strings.Repeat("a", common.DefaultPluginMaxPayloadBytes*2)),
					},
				},
			},
			wantErrContains: fmt.Sprintf(pluginErrPluginsInputEntrySize, "plugin-0"),
		},
		{
			name: "rejects plugin input map exceeding total payload size",
			input: map[string]*structpb.Struct{
				"plugin-0": {Fields: map[string]*structpb.Value{"k": structpb.NewStringValue(tooLongPayloadValue)}},
				"plugin-1": {Fields: map[string]*structpb.Value{"k": structpb.NewStringValue(tooLongPayloadValue)}},
				"plugin-2": {Fields: map[string]*structpb.Value{"k": structpb.NewStringValue(tooLongPayloadValue)}},
				"plugin-3": {Fields: map[string]*structpb.Value{"k": structpb.NewStringValue(tooLongPayloadValue)}},
				"plugin-4": {Fields: map[string]*structpb.Value{"k": structpb.NewStringValue(tooLongPayloadValue)}},
			},
			wantErrContains: pluginErrPluginsInputTotalSize,
		},
		{
			name: "nesting too deep",
			input: map[string]*structpb.Struct{
				"mlflow": makeDeepStruct(common.DefaultPluginMaxNestingDepth + 1),
			},
			wantErrContains: fmt.Sprintf(pluginErrPluginsInputNestingDepth, "mlflow"),
		},
		{
			name: "at configured boundaries",
			input: map[string]*structpb.Struct{
				"mlflow": makeDeepStruct(common.DefaultPluginMaxNestingDepth),
			},
		},
		{
			name: "reject nil plugin struct entry",
			input: map[string]*structpb.Struct{
				"mlflow": nil,
			},
			wantErrContains: fmt.Sprintf(pluginErrPluginsInputNilEntry, "mlflow"),
		},
		{
			name: "reject unset value kind in plugins_input",
			input: map[string]*structpb.Struct{
				"mlflow": {
					Fields: map[string]*structpb.Value{
						"broken": {},
					},
				},
			},
			wantErrContains: fmt.Sprintf(pluginErrPluginsInputInvalidValue, "mlflow"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits, err := common.GetPluginLimitsConfig()
			require.NoError(t, err)
			err = validatePluginsInputLimits(tt.input, limits)
			if tt.wantErrContains != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErrContains)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidatePluginsOutputLimits(t *testing.T) {
	tooLongPayloadValue := strings.Repeat("a", 55000)

	tests := []struct {
		name            string
		input           map[string]*apiv2beta1.PluginOutput
		wantErrContains string
	}{
		{
			name: "allow nil plugin output entry",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": nil,
			},
		},
		{
			name: "accepts multiple small plugin output payloads",
			input: map[string]*apiv2beta1.PluginOutput{
				"plugin-0": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {Value: structpb.NewStringValue(testPluginsURLBase), RenderType: apiv2beta1.MetadataValue_URL.Enum()},
					},
					State: apiv2beta1.PluginState_PLUGIN_RUNNING,
				},
				"plugin-1": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {Value: structpb.NewStringValue(testPluginsURLBase), RenderType: apiv2beta1.MetadataValue_URL.Enum()},
					},
					State: apiv2beta1.PluginState_PLUGIN_RUNNING,
				},
			},
		},
		{
			name:            "rejects plugin output map with too many keys",
			input:           createPluginOutputMapWithNKeys(common.DefaultPluginMaxKeys + 1),
			wantErrContains: pluginErrPluginsOutputTooManyKeys,
		},
		{
			name: "rejects plugin output entry exceeding per-plugin size",
			input: map[string]*apiv2beta1.PluginOutput{
				"plugin-0": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value:      structpb.NewStringValue(testPluginsURLBase + strings.Repeat("a", common.DefaultPluginMaxPayloadBytes*2)),
							RenderType: apiv2beta1.MetadataValue_URL.Enum(),
						},
					},
					State: apiv2beta1.PluginState_PLUGIN_RUNNING,
				},
			},
			wantErrContains: fmt.Sprintf(pluginErrPluginsOutputEntrySize, "plugin-0"),
		},
		{
			name: "rejects plugin output map exceeding total payload size",
			input: map[string]*apiv2beta1.PluginOutput{
				"plugin-0": {Entries: map[string]*apiv2beta1.MetadataValue{"run_url": {Value: structpb.NewStringValue(testPluginsURLBase + tooLongPayloadValue), RenderType: apiv2beta1.MetadataValue_URL.Enum()}}, State: apiv2beta1.PluginState_PLUGIN_RUNNING},
				"plugin-1": {Entries: map[string]*apiv2beta1.MetadataValue{"run_url": {Value: structpb.NewStringValue(testPluginsURLBase + tooLongPayloadValue), RenderType: apiv2beta1.MetadataValue_URL.Enum()}}, State: apiv2beta1.PluginState_PLUGIN_RUNNING},
				"plugin-2": {Entries: map[string]*apiv2beta1.MetadataValue{"run_url": {Value: structpb.NewStringValue(testPluginsURLBase + tooLongPayloadValue), RenderType: apiv2beta1.MetadataValue_URL.Enum()}}, State: apiv2beta1.PluginState_PLUGIN_RUNNING},
				"plugin-3": {Entries: map[string]*apiv2beta1.MetadataValue{"run_url": {Value: structpb.NewStringValue(testPluginsURLBase + tooLongPayloadValue), RenderType: apiv2beta1.MetadataValue_URL.Enum()}}, State: apiv2beta1.PluginState_PLUGIN_RUNNING},
				"plugin-4": {Entries: map[string]*apiv2beta1.MetadataValue{"run_url": {Value: structpb.NewStringValue(testPluginsURLBase + tooLongPayloadValue), RenderType: apiv2beta1.MetadataValue_URL.Enum()}}, State: apiv2beta1.PluginState_PLUGIN_RUNNING},
			},
			wantErrContains: pluginErrPluginsOutputTotalSize,
		},
		{
			name: "nested metadata value too deep",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"nested": {
							Value: makeDeepValue(common.DefaultPluginMaxNestingDepth + 1),
						},
					},
					State: apiv2beta1.PluginState_PLUGIN_RUNNING,
				},
			},
			wantErrContains: fmt.Sprintf(pluginErrPluginsOutputNestingDepth, "mlflow", "nested"),
		},
		{
			name: "at configured boundaries",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"nested": {
							Value: makeDeepValue(common.DefaultPluginMaxNestingDepth),
						},
					},
					State: apiv2beta1.PluginState_PLUGIN_RUNNING,
				},
			},
		},
		{
			name: "reject nil metadata entry",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": nil,
					},
					State: apiv2beta1.PluginState_PLUGIN_RUNNING,
				},
			},
			wantErrContains: fmt.Sprintf(pluginErrPluginsOutputNilMetadata, "mlflow", "run_url"),
		},
		{
			name: "reject nil metadata value",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value: nil,
						},
					},
					State: apiv2beta1.PluginState_PLUGIN_RUNNING,
				},
			},
			wantErrContains: fmt.Sprintf(pluginErrPluginsOutputNilValue, "mlflow", "run_url"),
		},
		{
			name: "reject unset value kind in plugins_output",
			input: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value: &structpb.Value{},
						},
					},
					State: apiv2beta1.PluginState_PLUGIN_RUNNING,
				},
			},
			wantErrContains: fmt.Sprintf(pluginErrPluginsOutputInvalidValue, "mlflow", "run_url"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits, err := common.GetPluginLimitsConfig()
			require.NoError(t, err)
			err = validatePluginsOutputLimits(tt.input, limits)
			if tt.wantErrContains != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErrContains)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidatePluginsInputLimitsUsesConfiguredOverrides(t *testing.T) {
	input := map[string]*structpb.Struct{
		"mlflow": {
			Fields: map[string]*structpb.Value{
				"k": structpb.NewStringValue("ok"),
			},
		},
		"other": {
			Fields: map[string]*structpb.Value{
				"k": structpb.NewStringValue("ok"),
			},
		},
	}

	setPluginLimitsConfigForTest(t, map[string]string{
		common.PluginMaxKeys: "1",
	})

	limits, err := common.GetPluginLimitsConfig()
	require.NoError(t, err)
	err = validatePluginsInputLimits(input, limits)
	require.Error(t, err)
	assert.ErrorContains(t, err, "exceeds maximum 1")
}

func TestValidatePluginsOutputLimitsUsesConfiguredOverrides(t *testing.T) {
	output := map[string]*apiv2beta1.PluginOutput{
		"mlflow": {
			Entries: map[string]*apiv2beta1.MetadataValue{
				"run_url": {
					Value:      structpb.NewStringValue(testPluginsURLBase),
					RenderType: apiv2beta1.MetadataValue_URL.Enum(),
				},
			},
			State: apiv2beta1.PluginState_PLUGIN_RUNNING,
		},
	}

	setPluginLimitsConfigForTest(t, map[string]string{
		common.PluginMaxPayloadBytes: "64",
	})

	limits, err := common.GetPluginLimitsConfig()
	require.NoError(t, err)
	err = validatePluginsOutputLimits(output, limits)
	require.Error(t, err)
	assert.ErrorContains(t, err, "exceeds maximum 64 bytes")
}

func TestValidatePluginsInputLimitsUsesNestingDepthOverride(t *testing.T) {
	input := map[string]*structpb.Struct{
		"mlflow": {
			Fields: map[string]*structpb.Value{
				"nested": makeDeepValue(3),
			},
		},
	}

	setPluginLimitsConfigForTest(t, map[string]string{
		common.PluginMaxNestingDepth: "2",
	})

	limits, err := common.GetPluginLimitsConfig()
	require.NoError(t, err)
	err = validatePluginsInputLimits(input, limits)
	require.Error(t, err)
	assert.ErrorContains(t, err, "nesting depth exceeds maximum 2")
}

func TestValidatePluginsOutputLimitsUsesTotalPayloadOverride(t *testing.T) {
	output := map[string]*apiv2beta1.PluginOutput{
		"mlflow": {
			Entries: map[string]*apiv2beta1.MetadataValue{
				"run_url": {
					Value:      structpb.NewStringValue("https://example.com/run1"),
					RenderType: apiv2beta1.MetadataValue_URL.Enum(),
				},
			},
			State: apiv2beta1.PluginState_PLUGIN_RUNNING,
		},
		"other": {
			Entries: map[string]*apiv2beta1.MetadataValue{
				"status": {
					Value: structpb.NewStringValue("https://example.com/run2"),
				},
			},
			State: apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
		},
	}

	setPluginLimitsConfigForTest(t, map[string]string{
		common.PluginMaxTotalPayloadBytes: "10",
		common.PluginMaxPayloadBytes:      "10",
	})

	limits, err := common.GetPluginLimitsConfig()
	require.NoError(t, err)
	err = validatePluginsOutputLimits(output, limits)
	require.Error(t, err)
	assert.ErrorContains(t, err, "exceeds maximum 10 bytes")
}

func makeDeepStruct(depth int) *structpb.Struct {
	current := structpb.NewStringValue("leaf")
	for range depth {
		current = structpb.NewStructValue(&structpb.Struct{
			Fields: map[string]*structpb.Value{"nested": current},
		})
	}
	return current.GetStructValue()
}

func makeDeepValue(depth int) *structpb.Value {
	current := structpb.NewStringValue("leaf")
	for range depth {
		current = structpb.NewStructValue(&structpb.Struct{
			Fields: map[string]*structpb.Value{"nested": current},
		})
	}
	return current
}

func TestToModelRunPluginsFields(t *testing.T) {
	pluginsInput := map[string]*structpb.Struct{
		"mlflow": {Fields: map[string]*structpb.Value{
			"experiment_name": structpb.NewStringValue(testPluginsExperimentName),
		}},
		"other": {Fields: map[string]*structpb.Value{
			"key": structpb.NewBoolValue(true),
		}},
	}
	pluginsOutput := map[string]*apiv2beta1.PluginOutput{
		"mlflow": {
			Entries: map[string]*apiv2beta1.MetadataValue{
				"root_run_id": {Value: structpb.NewStringValue("abc123")},
			},
			State:        apiv2beta1.PluginState_PLUGIN_SUCCEEDED,
			StateMessage: "ok",
		},
		"other": {
			State:        apiv2beta1.PluginState_PLUGIN_RUNNING,
			StateMessage: "in progress",
		},
	}

	t.Run("with plugins fields", func(t *testing.T) {
		run := &apiv2beta1.Run{
			RunId:       "run1",
			DisplayName: "test",
			PipelineSource: &apiv2beta1.Run_PipelineVersionReference{
				PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
					PipelineId: "p1", PipelineVersionId: "pv1",
				},
			},
			PluginsInput:  pluginsInput,
			PluginsOutput: pluginsOutput,
		}
		got, err := toModelRun(run)
		require.NoError(t, err)
		require.NotNil(t, got.PluginsInputString)
		require.NotNil(t, got.PluginsOutputString)

		parsedInput, err := jsonToPluginsInput(largeTextToString(got.PluginsInputString))
		require.NoError(t, err)
		assert.Equal(t, testPluginsExperimentName, parsedInput["mlflow"].Fields["experiment_name"].GetStringValue())

		parsedOutput, err := jsonToPluginsOutput(largeTextToString(got.PluginsOutputString))
		require.NoError(t, err)
		assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, parsedOutput["mlflow"].State)
		assert.Equal(t, "abc123", parsedOutput["mlflow"].Entries["root_run_id"].Value.GetStringValue())
	})

	t.Run("nil plugins fields", func(t *testing.T) {
		apiRun := &apiv2beta1.Run{
			RunId:       "run2",
			DisplayName: "test-nil",
			PipelineSource: &apiv2beta1.Run_PipelineVersionReference{
				PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
					PipelineId: "p1", PipelineVersionId: "pv1",
				},
			},
		}
		got, err := toModelRun(apiRun)
		require.NoError(t, err)
		assert.Nil(t, got.PluginsInputString)
		assert.Nil(t, got.PluginsOutputString)
	})

	t.Run("invalid plugins output URL scheme returns error", func(t *testing.T) {
		apiRun := &apiv2beta1.Run{
			RunId:       "run3",
			DisplayName: "test-invalid",
			PipelineSource: &apiv2beta1.Run_PipelineVersionReference{
				PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
					PipelineId: "p1", PipelineVersionId: "pv1",
				},
			},
			PluginsOutput: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value:      structpb.NewStringValue(testPluginsUnsafeJavaScriptURL),
							RenderType: apiv2beta1.MetadataValue_URL.Enum(),
						},
					},
				},
			},
		}

		_, err := toModelRun(apiRun)
		require.Error(t, err)
	})

	t.Run("plugins_input exceeding limits returns error", func(t *testing.T) {
		apiRun := &apiv2beta1.Run{
			RunId:       "run4",
			DisplayName: "test-too-large-input",
			PipelineSource: &apiv2beta1.Run_PipelineVersionReference{
				PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
					PipelineId: "p1", PipelineVersionId: "pv1",
				},
			},
			PluginsInput: map[string]*structpb.Struct{
				"mlflow": {
					Fields: map[string]*structpb.Value{
						"blob": structpb.NewStringValue(strings.Repeat("a", common.DefaultPluginMaxPayloadBytes*2)),
					},
				},
			},
		}

		_, err := toModelRun(apiRun)
		require.Error(t, err)
	})

	t.Run("plugins_output exceeding limits returns error", func(t *testing.T) {
		apiRun := &apiv2beta1.Run{
			RunId:       "run5",
			DisplayName: "test-too-large-output",
			PipelineSource: &apiv2beta1.Run_PipelineVersionReference{
				PipelineVersionReference: &apiv2beta1.PipelineVersionReference{
					PipelineId: "p1", PipelineVersionId: "pv1",
				},
			},
			PluginsOutput: map[string]*apiv2beta1.PluginOutput{
				"mlflow": {
					Entries: map[string]*apiv2beta1.MetadataValue{
						"run_url": {
							Value:      structpb.NewStringValue(testPluginsURLBase + strings.Repeat("a", common.DefaultPluginMaxPayloadBytes*2)),
							RenderType: apiv2beta1.MetadataValue_URL.Enum(),
						},
					},
					State: apiv2beta1.PluginState_PLUGIN_RUNNING,
				},
			},
		}

		_, err := toModelRun(apiRun)
		require.Error(t, err)
	})
}

func TestToApiRunPluginsFields(t *testing.T) {
	inputJSON := `{"mlflow":{"experiment_name":"` + testPluginsExperimentName + `"},"other":{"key":true}}`
	outputJSON := `{"mlflow":{"entries":{"root_run_id":{"value":"abc123"}},"state":"PLUGIN_SUCCEEDED","stateMessage":"ok"},"other":{"state":"PLUGIN_RUNNING","stateMessage":"in progress"}}`

	t.Run("with plugins fields", func(t *testing.T) {
		modelRun := &model.Run{
			UUID:        "run1",
			DisplayName: "test",
			PipelineSpec: model.PipelineSpec{
				PipelineVersionId: "pv1",
				PipelineId:        "p1",
			},
			RunDetails: model.RunDetails{
				PluginsInputString:  testLargeTextPtr(inputJSON),
				PluginsOutputString: testLargeTextPtr(outputJSON),
			},
		}
		got := toApiRun(modelRun)
		require.Len(t, got.PluginsInput, 2)
		require.Contains(t, got.PluginsInput, "mlflow")
		assert.Equal(t, testPluginsExperimentName, got.PluginsInput["mlflow"].Fields["experiment_name"].GetStringValue())
		require.Contains(t, got.PluginsInput, "other")
		assert.Equal(t, true, got.PluginsInput["other"].Fields["key"].GetBoolValue())

		require.Len(t, got.PluginsOutput, 2)
		require.Contains(t, got.PluginsOutput, "mlflow")
		assert.Equal(t, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, got.PluginsOutput["mlflow"].State)
		assert.Equal(t, "abc123", got.PluginsOutput["mlflow"].Entries["root_run_id"].Value.GetStringValue())
		require.Contains(t, got.PluginsOutput, "other")
		assert.Equal(t, apiv2beta1.PluginState_PLUGIN_RUNNING, got.PluginsOutput["other"].State)
	})

	t.Run("nil plugins fields", func(t *testing.T) {
		modelRun := &model.Run{
			UUID:        "run2",
			DisplayName: "test-nil",
			PipelineSpec: model.PipelineSpec{
				PipelineVersionId: "pv1",
				PipelineId:        "p1",
			},
			RunDetails: model.RunDetails{},
		}
		got := toApiRun(modelRun)
		assert.Nil(t, got.PluginsInput)
		assert.Nil(t, got.PluginsOutput)
	})

	t.Run("invalid plugins output URL in storage returns API error", func(t *testing.T) {
		modelRun := &model.Run{
			UUID:        "run3",
			DisplayName: "test-invalid",
			PipelineSpec: model.PipelineSpec{
				PipelineVersionId: "pv1",
				PipelineId:        "p1",
			},
			RunDetails: model.RunDetails{
				PluginsOutputString: testLargeTextPtr(`{"mlflow":{"entries":{"run_url":{"value":"` + testPluginsUnsafeJavaScriptURL + `","renderType":"URL"}}}}`),
			},
		}
		got := toApiRun(modelRun)
		require.NotNil(t, got.Error)
		assert.Nil(t, got.PluginsOutput)
	})
}

func TestToModelJobPluginsInput(t *testing.T) {
	pluginsInput := map[string]*structpb.Struct{
		"mlflow": {Fields: map[string]*structpb.Value{
			"experiment_name": structpb.NewStringValue(testPluginsRecurringExperimentName),
		}},
		"other": {Fields: map[string]*structpb.Value{
			"enabled": structpb.NewBoolValue(true),
		}},
	}

	t.Run("with plugins_input", func(t *testing.T) {
		apiJob := &apiv2beta1.RecurringRun{
			RecurringRunId: "job1",
			DisplayName:    testPluginsJobName,
			MaxConcurrency: 1,
			Mode:           apiv2beta1.RecurringRun_ENABLE,
			Trigger: &apiv2beta1.Trigger{
				Trigger: &apiv2beta1.Trigger_PeriodicSchedule{
					PeriodicSchedule: &apiv2beta1.PeriodicSchedule{IntervalSecond: 60},
				},
			},
			PluginsInput: pluginsInput,
		}
		got, err := toModelJob(apiJob)
		require.NoError(t, err)
		require.NotNil(t, got.PluginsInputString)

		parsedInput, err := jsonToPluginsInput(largeTextToString(got.PluginsInputString))
		require.NoError(t, err)
		require.Len(t, parsedInput, 2)
		require.Contains(t, parsedInput, "mlflow")
		assert.Equal(t, testPluginsRecurringExperimentName, parsedInput["mlflow"].Fields["experiment_name"].GetStringValue())
		require.Contains(t, parsedInput, "other")
		assert.Equal(t, true, parsedInput["other"].Fields["enabled"].GetBoolValue())
	})

	t.Run("nil plugins_input", func(t *testing.T) {
		apiJob := &apiv2beta1.RecurringRun{
			RecurringRunId: "job2",
			DisplayName:    "test-job-nil",
			MaxConcurrency: 1,
			Mode:           apiv2beta1.RecurringRun_ENABLE,
			Trigger: &apiv2beta1.Trigger{
				Trigger: &apiv2beta1.Trigger_PeriodicSchedule{
					PeriodicSchedule: &apiv2beta1.PeriodicSchedule{IntervalSecond: 60},
				},
			},
		}
		got, err := toModelJob(apiJob)
		require.NoError(t, err)
		assert.Nil(t, got.PluginsInputString)
	})

	t.Run("plugins_input exceeding limits returns error", func(t *testing.T) {
		apiJob := &apiv2beta1.RecurringRun{
			RecurringRunId: "job3",
			DisplayName:    "test-job-too-large",
			MaxConcurrency: 1,
			Mode:           apiv2beta1.RecurringRun_ENABLE,
			Trigger: &apiv2beta1.Trigger{
				Trigger: &apiv2beta1.Trigger_PeriodicSchedule{
					PeriodicSchedule: &apiv2beta1.PeriodicSchedule{IntervalSecond: 60},
				},
			},
			PluginsInput: map[string]*structpb.Struct{
				"mlflow": {
					Fields: map[string]*structpb.Value{
						"blob": structpb.NewStringValue(strings.Repeat("a", common.DefaultPluginMaxPayloadBytes*2)),
					},
				},
			},
		}

		_, err := toModelJob(apiJob)
		require.Error(t, err)
	})
}

func TestToApiRecurringRunPluginsInput(t *testing.T) {
	inputJSON := `{"mlflow":{"experiment_name":"` + testPluginsRecurringExperimentName + `"},"other":{"enabled":true}}`

	t.Run("with plugins_input", func(t *testing.T) {
		modelJob := &model.Job{
			UUID:               "job1",
			DisplayName:        testPluginsJobName,
			K8SName:            testPluginsJobName,
			Enabled:            true,
			Conditions:         "ENABLED",
			MaxConcurrency:     1,
			PluginsInputString: testLargeTextPtr(inputJSON),
			PipelineSpec: model.PipelineSpec{
				PipelineId:        "p1",
				PipelineVersionId: "pv1",
			},
		}
		got := toApiRecurringRun(modelJob)
		require.Len(t, got.PluginsInput, 2)
		require.Contains(t, got.PluginsInput, "mlflow")
		assert.Equal(t, testPluginsRecurringExperimentName, got.PluginsInput["mlflow"].Fields["experiment_name"].GetStringValue())
		require.Contains(t, got.PluginsInput, "other")
		assert.Equal(t, true, got.PluginsInput["other"].Fields["enabled"].GetBoolValue())
	})

	t.Run("empty plugins_input", func(t *testing.T) {
		modelJob := &model.Job{
			UUID:           "job2",
			DisplayName:    "test-job-empty",
			K8SName:        "test-job-empty",
			Enabled:        true,
			Conditions:     "ENABLED",
			MaxConcurrency: 1,
			PipelineSpec: model.PipelineSpec{
				PipelineId:        "p1",
				PipelineVersionId: "pv1",
			},
		}
		got := toApiRecurringRun(modelJob)
		assert.Nil(t, got.PluginsInput)
	})
}
