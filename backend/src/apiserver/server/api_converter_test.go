// Copyright 2018-2023 The Kubeflow Authors
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
	"strings"
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func TestToModelExperiment(t *testing.T) {
	tests := []struct {
		name                    string
		experiment              interface{}
		wantError               bool
		errorMessage            string
		expectedModelExperiment *model.Experiment
	}{
		{
			"API V1beta1: No resource references",
			&apiv1beta1.Experiment{
				Name:        "exp1",
				Description: "This is an experiment",
			},
			false,
			"",
			&model.Experiment{
				Name:        "exp1",
				Description: "This is an experiment",
				Namespace:   "",
			},
		},
		{
			"API V1beta1: Valid resource references",
			&apiv1beta1.Experiment{
				Name:        "exp1",
				Description: "This is an experiment",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					&apiv1beta1.ResourceReference{
						Key: &apiv1beta1.ResourceKey{
							Type: apiv1beta1.ResourceType_NAMESPACE,
							Id:   "ns1",
						},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			false,
			"",
			&model.Experiment{
				Name:        "exp1",
				Description: "This is an experiment",
				Namespace:   "ns1",
			},
		},
		{
			"API V1beta1: Invalid resource references",
			&apiv1beta1.Experiment{
				Name:        "exp1",
				Description: "This is an experiment",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					&apiv1beta1.ResourceReference{
						Key: &apiv1beta1.ResourceKey{
							Type: apiv1beta1.ResourceType_EXPERIMENT,
							Id:   "invalid",
						},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			true,
			"Invalid resource references for experiment",
			nil,
		},
		{
			"API V2beta1: Happy pass",
			&apiv2beta1.Experiment{
				DisplayName: "exp2",
				Description: "API V2beta1 test experiment",
				Namespace:   "ns2",
			},
			false,
			"",
			&model.Experiment{
				Name:        "exp2",
				Description: "API V2beta1 test experiment",
				Namespace:   "ns2",
			},
		},
		{
			"Wrong API type",
			&model.Experiment{
				Name:        "",
				Description: "API V2beta1 test experiment",
				Namespace:   "ns2",
			},
			true,
			"Invalid experiment type",
			nil,
		},
	}

	for _, tc := range tests {
		modelExperiment, err := toModelExperiment(tc.experiment)
		if tc.wantError {
			if err == nil {
				t.Errorf("TestToModelExperiment(%v) expect error but got nil", tc.name)
			} else if !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("TestToModelExperiment(%v) expect error containing: %v, but got: %v", tc.name, tc.errorMessage, err)
			}
		} else {
			if err != nil {
				t.Errorf("TestToModelExperiment(%v) expect no error but got %v", tc.name, err)
			} else if !cmp.Equal(tc.expectedModelExperiment, modelExperiment) {
				t.Errorf("TestToModelExperiment(%v) expect (%+v) but got (%+v)", tc.name, tc.expectedModelExperiment, modelExperiment)
			}
		}
	}
}

func TestToModelRunDetail(t *testing.T) {
	store, manager, experiment := initWithExperiment(t)
	defer store.Close()

	listParams := []interface{}{1, 2, 3}
	v2RuntimeListParams, _ := structpb.NewList(listParams)

	structParams := map[string]interface{}{"structParam1": "hello", "structParam2": 32}
	v2RuntimeStructParams, _ := structpb.NewStruct(structParams)

	// Test all parameters types converted to model.RuntimeConfig.Parameters, which is string type
	v2RuntimeParams := map[string]*structpb.Value{
		"param2": &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "world"}},
		"param3": &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: true}},
		"param4": &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: v2RuntimeListParams}},
		"param5": &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: 12}},
		"param6": &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: v2RuntimeStructParams}},
	}

	tests := []struct {
		name                   string
		apiRun                 *apiv1beta1.Run
		workflow               *util.Workflow
		manifest               string
		templateType           template.TemplateType
		expectedModelRunDetail *model.RunDetail
	}{
		{
			name: "v1",
			apiRun: &apiv1beta1.Run{
				Id:          "run1",
				Name:        "name1",
				Description: "this is a run",
				PipelineSpec: &apiv1beta1.PipelineSpec{
					Parameters: []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
				},
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
						Relationship: apiv1beta1.Relationship_OWNER}},
			},
			workflow: util.NewWorkflow(&v1alpha1.Workflow{
				ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "123"},
				Status:     v1alpha1.WorkflowStatus{Phase: "running"},
			}),
			manifest:     "workflow spec",
			templateType: template.V1,
			expectedModelRunDetail: &model.RunDetail{
				Run: model.Run{
					UUID:           "123",
					ExperimentUUID: experiment.UUID,
					Namespace:      "ns1",
					Name:           "name1",
					DisplayName:    "name1",
					Description:    "this is a run",
					PipelineSpec: model.PipelineSpec{
						WorkflowSpecManifest: "workflow spec",
						Parameters:           `[{"name":"param2","value":"world"}]`,
					},
					ResourceReferences: []*model.ResourceReference{
						{
							ResourceUUID:  "123",
							ResourceType:  model.RunResourceType,
							ReferenceUUID: experiment.UUID,
							ReferenceName: experiment.Name,
							ReferenceType: model.ExperimentResourceType,
							Relationship:  model.OwnerRelationship},
					},
				},
			},
		},
		{
			name: "v2",
			apiRun: &apiv1beta1.Run{
				Id:          "run1",
				Name:        "name1",
				Description: "this is a run",
				PipelineSpec: &apiv1beta1.PipelineSpec{
					RuntimeConfig: &apiv1beta1.PipelineSpec_RuntimeConfig{
						Parameters: v2RuntimeParams,
					}},
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
						Relationship: apiv1beta1.Relationship_OWNER}},
			},
			workflow: util.NewWorkflow(&v1alpha1.Workflow{
				ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "123"},
				Status:     v1alpha1.WorkflowStatus{Phase: "running"},
			}),
			manifest:     "pipeline spec",
			templateType: template.V2,
			expectedModelRunDetail: &model.RunDetail{
				Run: model.Run{
					UUID:           "123",
					ExperimentUUID: experiment.UUID,
					Namespace:      "ns1",
					Name:           "name1",
					DisplayName:    "name1",
					Description:    "this is a run",
					PipelineSpec: model.PipelineSpec{
						PipelineSpecManifest: "pipeline spec",
						RuntimeConfig: model.RuntimeConfig{
							// Note: for some versions of structpb.Value.MarshalJSON(), there is a trailing space after array items or struct items
							Parameters: "{\"param2\":\"world\",\"param3\":true,\"param4\":[1,2,3],\"param5\":12,\"param6\":{\"structParam1\":\"hello\",\"structParam2\":32}}",
						},
					},
					ResourceReferences: []*model.ResourceReference{
						{
							ResourceUUID:  "123",
							ResourceType:  model.RunResourceType,
							ReferenceUUID: experiment.UUID,
							ReferenceName: experiment.Name,
							ReferenceType: model.ExperimentResourceType,
							Relationship:  model.OwnerRelationship},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelRunDetail, err := manager.ToModelRunDetail(tt.apiRun, "123", 0, tt.manifest, tt.templateType)
			assert.Nil(t, err)
			assert.Equal(t, tt.expectedModelRunDetail, modelRunDetail)
		})
	}

}

func TestToModelJob(t *testing.T) {
	store, manager, experiment, pipeline, _ := initWithExperimentAndPipeline(t)
	defer store.Close()

	tests := []struct {
		name             string
		job              interface{}
		manifest         string
		templateType     template.TemplateType
		expectedModelJob *model.Job
	}{
		{
			name: "v1api v1template",
			job: &apiv1beta1.Job{
				Name:           "name1",
				Enabled:        true,
				MaxConcurrency: 1,
				NoCatchup:      true,
				Trigger: &apiv1beta1.Trigger{
					Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
						StartTime: &timestamp.Timestamp{Seconds: 1},
						Cron:      "1 * * * *",
					}}},
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: apiv1beta1.Relationship_OWNER},
				},
				PipelineSpec: &apiv1beta1.PipelineSpec{PipelineId: pipeline.UUID, Parameters: []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}}},
			},
			manifest:     "workflow spec",
			templateType: template.V1,
			expectedModelJob: &model.Job{
				DisplayName: "name1",
				Name:        "name1",
				Namespace:   "ns1",
				Enabled:     true,
				Trigger: model.Trigger{
					CronSchedule: model.CronSchedule{
						CronScheduleStartTimeInSec: util.Int64Pointer(1),
						Cron:                       util.StringPointer("1 * * * *"),
					},
				},
				MaxConcurrency: 1,
				NoCatchup:      true,
				PipelineSpec: model.PipelineSpec{
					PipelineId:           pipeline.UUID,
					PipelineName:         pipeline.Name,
					WorkflowSpecManifest: "workflow spec",
					Parameters:           `[{"name":"param2","value":"world"}]`,
				},
				ResourceReferences: []*model.ResourceReference{
					{
						ResourceType:  model.JobResourceType,
						ReferenceUUID: experiment.UUID,
						ReferenceName: experiment.Name,
						ReferenceType: model.ExperimentResourceType,
						Relationship:  model.OwnerRelationship},
				},
			},
		}, {
			name: "v1api v2template",
			job: &apiv1beta1.Job{
				Name:           "name1",
				Enabled:        true,
				MaxConcurrency: 1,
				NoCatchup:      true,
				Trigger: &apiv1beta1.Trigger{
					Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
						StartTime: &timestamp.Timestamp{Seconds: 1},
						Cron:      "1 * * * *",
					}}},
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: apiv1beta1.Relationship_OWNER},
				},
				PipelineSpec: &apiv1beta1.PipelineSpec{PipelineId: pipeline.UUID,
					RuntimeConfig: &apiv1beta1.PipelineSpec_RuntimeConfig{Parameters: map[string]*structpb.Value{
						"param2": &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "world"}},
					}}},
			},
			manifest:     "pipeline spec",
			templateType: template.V2,
			expectedModelJob: &model.Job{
				Name:        "name1",
				Namespace:   "ns1",
				DisplayName: "name1",
				Enabled:     true,
				Trigger: model.Trigger{
					CronSchedule: model.CronSchedule{
						CronScheduleStartTimeInSec: util.Int64Pointer(1),
						Cron:                       util.StringPointer("1 * * * *"),
					},
				},
				MaxConcurrency: 1,
				NoCatchup:      true,
				PipelineSpec: model.PipelineSpec{
					PipelineId:           pipeline.UUID,
					PipelineName:         pipeline.Name,
					PipelineSpecManifest: "pipeline spec",
					RuntimeConfig: model.RuntimeConfig{
						Parameters: "{\"param2\":\"world\"}",
					},
				},
				ResourceReferences: []*model.ResourceReference{
					{
						ResourceType:  model.JobResourceType,
						ReferenceUUID: experiment.UUID,
						ReferenceName: experiment.Name,
						ReferenceType: model.ExperimentResourceType,
						Relationship:  model.OwnerRelationship},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelJob, err := manager.ToModelJob(tt.job, tt.manifest, tt.templateType)
			assert.Nil(t, err)
			assert.Equal(t, tt.expectedModelJob, modelJob)
		})
	}
}

func TestToModelResourceReferences(t *testing.T) {
	store, manager, _ := initWithJob(t)
	defer store.Close()
	refs, err := manager.toModelResourceReferences("r1", model.RunResourceType, []*apiv1beta1.ResourceReference{
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: common.DefaultFakeUUID}, Relationship: apiv1beta1.Relationship_OWNER},
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: common.DefaultFakeUUID}, Relationship: apiv1beta1.Relationship_CREATOR},
	})
	assert.Nil(t, err)
	expectedRefs := []*model.ResourceReference{
		{ResourceUUID: "r1", ResourceType: model.RunResourceType,
			ReferenceUUID: common.DefaultFakeUUID, ReferenceName: "e1", ReferenceType: model.ExperimentResourceType, Relationship: model.OwnerRelationship},
		{ResourceUUID: "r1", ResourceType: model.RunResourceType,
			ReferenceUUID: common.DefaultFakeUUID, ReferenceName: "j1", ReferenceType: model.JobResourceType, Relationship: model.CreatorRelationship},
	}
	assert.Equal(t, expectedRefs, refs)
}

func TestToModelResourceReferences_UnknownRefType(t *testing.T) {
	store, manager, _ := initWithJob(t)
	defer store.Close()

	_, err := manager.toModelResourceReferences("r1", model.RunResourceType, []*apiv1beta1.ResourceReference{
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_UNKNOWN_RESOURCE_TYPE, Id: "e1"}, Relationship: apiv1beta1.Relationship_OWNER},
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "j1"}, Relationship: apiv1beta1.Relationship_CREATOR},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to convert reference type")
}

func TestToModelResourceReferences_NamespaceRef(t *testing.T) {
	store, manager, _ := initWithJob(t)
	defer store.Close()

	modelRefs, err := manager.toModelResourceReferences("r1", model.RunResourceType, []*apiv1beta1.ResourceReference{
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "e1"}, Relationship: apiv1beta1.Relationship_OWNER},
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(modelRefs))
}

func TestToModelResourceReferences_UnknownRelationship(t *testing.T) {
	store, manager, _ := initWithJob(t)
	defer store.Close()
	_, err := manager.toModelResourceReferences("r1", model.RunResourceType, []*apiv1beta1.ResourceReference{
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "e1"}, Relationship: apiv1beta1.Relationship_UNKNOWN_RELATIONSHIP},
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "j1"}, Relationship: apiv1beta1.Relationship_CREATOR},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to convert relationship")
}

func TestToModelResourceReferences_ReferredJobNotFound(t *testing.T) {
	store, manager, _ := initWithJob(t)
	defer store.Close()
	_, err := manager.toModelResourceReferences("r1", model.RunResourceType, []*apiv1beta1.ResourceReference{
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "e1"}, Relationship: apiv1beta1.Relationship_OWNER},
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "j2"}, Relationship: apiv1beta1.Relationship_CREATOR},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to find the referred resource")
}

func TestToModelResourceReferences_ReferredExperimentNotFound(t *testing.T) {
	store, manager, _ := initWithJob(t)
	defer store.Close()
	_, err := manager.toModelResourceReferences("r1", model.RunResourceType, []*apiv1beta1.ResourceReference{
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "e2"}, Relationship: apiv1beta1.Relationship_OWNER},
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "j1"}, Relationship: apiv1beta1.Relationship_CREATOR},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to find the referred resource")
}

func TestToModelRunMetric(t *testing.T) {
	apiRunMetric := &apiv1beta1.RunMetric{
		Name:   "metric-1",
		NodeId: "node-1",
		Value: &apiv1beta1.RunMetric_NumberValue{
			NumberValue: 0.88,
		},
		Format: apiv1beta1.RunMetric_RAW,
	}

	actualModelRunMetric := ToModelRunMetric(apiRunMetric, "run-1")

	expectedModelRunMetric := &model.RunMetric{
		RunUUID:     "run-1",
		Name:        "metric-1",
		NodeID:      "node-1",
		NumberValue: 0.88,
		Format:      "RAW",
	}
	assert.Equal(t, expectedModelRunMetric, actualModelRunMetric)
}

// Tests ToModelPipelineVersion
func TestToModelPipelineVersion(t *testing.T) {
	apiPipelineVersion := &apiv1beta1.PipelineVersion{
		Id:            "pipelineversion1",
		CreatedAt:     &timestamp.Timestamp{Seconds: 1},
		Parameters:    []*apiv1beta1.Parameter{},
		CodeSourceUrl: "http://repo/11111",
		ResourceReferences: []*apiv1beta1.ResourceReference{
			&apiv1beta1.ResourceReference{
				Key: &apiv1beta1.ResourceKey{
					Id:   "pipeline1",
					Type: apiv1beta1.ResourceType_PIPELINE,
				},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
		},
	}

	convertedModelPipelineVersion, _ := ToModelPipelineVersion(apiPipelineVersion)

	expectedModelPipelineVersion := model.PipelineVersion{
		UUID:           "pipelineversion1",
		CreatedAtInSec: 1,
		Parameters:     "[]",
		PipelineId:     "pipeline1",
		CodeSourceUrl:  "http://repo/11111",
	}

	assert.Equal(t, expectedModelPipelineVersion, convertedModelPipelineVersion)
}

// Tests ToApiPipelineV1
func TestToApiPipeline(t *testing.T) {
	modelPipeline := &model.Pipeline{
		UUID:           "pipeline1",
		CreatedAtInSec: 1,
	}
	modelVersion := &model.PipelineVersion{
		UUID:           "pipelineversion1",
		CreatedAtInSec: 1,
		Parameters:     "[]",
		PipelineId:     "pipeline1",
		Description:    "desc1",
		CodeSourceUrl:  "http://repo/22222",
	}
	apiPipeline := ToApiPipelineV1(modelPipeline, modelVersion)
	expectedApiPipeline := &apiv1beta1.Pipeline{
		Id:         "pipeline1",
		CreatedAt:  &timestamp.Timestamp{Seconds: 1},
		Parameters: []*apiv1beta1.Parameter{},
		DefaultVersion: &apiv1beta1.PipelineVersion{
			Id:            "pipelineversion1",
			CreatedAt:     &timestamp.Timestamp{Seconds: 1},
			Parameters:    []*apiv1beta1.Parameter{},
			Description:   "desc1",
			CodeSourceUrl: "http://repo/22222",
			ResourceReferences: []*apiv1beta1.ResourceReference{
				&apiv1beta1.ResourceReference{
					Key: &apiv1beta1.ResourceKey{
						Id:   "pipeline1",
						Type: apiv1beta1.ResourceType_PIPELINE,
					},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		},
	}
	assert.Equal(t, expectedApiPipeline, apiPipeline)
}

// Tests ToApiPipelineV1 (error parsing a field)
func TestToApiPipeline_ErrorParsingField(t *testing.T) {
	modelPipeline := &model.Pipeline{
		UUID:           "pipeline1",
		CreatedAtInSec: 1,
	}
	modelVersion := &model.PipelineVersion{
		Parameters: "wrong parameters",
	}
	apiPipeline := ToApiPipelineV1(modelPipeline, modelVersion)
	assert.Equal(t, "pipeline1", apiPipeline.Id)
	assert.Contains(t, apiPipeline.Error, "Parameter with wrong format is stored")
}

func TestToApiRunDetailV1_RuntimeParams(t *testing.T) {
	modelRun := &model.RunDetail{
		Run: model.Run{
			UUID:             "run123",
			Name:             "name123",
			StorageState:     apiv1beta1.Run_STORAGESTATE_AVAILABLE.String(),
			DisplayName:      "displayName123",
			Namespace:        "ns123",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			FinishedAtInSec:  1,
			Conditions:       "running",
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: "manifest",
				RuntimeConfig: model.RuntimeConfig{
					Parameters:   "{\"param2\":\"world\",\"param3\":true,\"param4\":[1,2,3],\"param5\":12,\"param6\":{\"structParam1\":\"hello\",\"structParam2\":32}}",
					PipelineRoot: "model-pipeline-root",
				},
			},
			ResourceReferences: []*model.ResourceReference{
				{ResourceUUID: "run123", ResourceType: model.RunResourceType, ReferenceUUID: "job123",
					ReferenceName: "j123", ReferenceType: model.JobResourceType, Relationship: model.CreatorRelationship},
			},
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow123"},
	}
	apiRun := ToApiRunDetailV1(modelRun)

	listParams := []interface{}{1, 2, 3}
	v2RuntimeListParams, _ := structpb.NewList(listParams)

	structParams := map[string]interface{}{"structParam1": "hello", "structParam2": 32}
	v2RuntimeStructParams, _ := structpb.NewStruct(structParams)

	// Test all parameters types converted to model.RuntimeConfig.Parameters, which is string type
	v2RuntimeParams := map[string]*structpb.Value{
		"param2": structpb.NewStringValue("world"),
		"param3": structpb.NewBoolValue(true),
		"param4": structpb.NewListValue(v2RuntimeListParams),
		"param5": structpb.NewNumberValue(12),
		"param6": structpb.NewStructValue(v2RuntimeStructParams),
	}

	expectedApiRun := &apiv1beta1.RunDetail{
		Run: &apiv1beta1.Run{
			Id:           "run123",
			Name:         "displayName123",
			StorageState: apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:    &timestamp.Timestamp{Seconds: 1},
			ScheduledAt:  &timestamp.Timestamp{Seconds: 1},
			FinishedAt:   &timestamp.Timestamp{Seconds: 1},
			Status:       "running",
			PipelineSpec: &apiv1beta1.PipelineSpec{
				WorkflowManifest: "manifest",
				RuntimeConfig: &apiv1beta1.PipelineSpec_RuntimeConfig{
					Parameters:   v2RuntimeParams,
					PipelineRoot: "model-pipeline-root",
				},
			},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "job123"},
					Name: "j123", Relationship: apiv1beta1.Relationship_CREATOR},
			},
		},
		PipelineRuntime: &apiv1beta1.PipelineRuntime{
			WorkflowManifest: "workflow123",
		},
	}
	// Compare the string representation of ApiRuns, since these structs have internal fields
	// used only by protobuff, and may be different. The .String() method marshal all
	// exported fields into string format.
	// See https://github.com/stretchr/testify/issues/758
	assert.Equal(t, expectedApiRun.String(), apiRun.String())
}

func TestToApiRunDetailV1_V1Params(t *testing.T) {
	modelRun := &model.RunDetail{
		Run: model.Run{
			UUID:             "run123",
			Name:             "name123",
			StorageState:     apiv1beta1.Run_STORAGESTATE_AVAILABLE.String(),
			DisplayName:      "displayName123",
			Namespace:        "ns123",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			FinishedAtInSec:  1,
			Conditions:       "running",
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: "manifest",
				Parameters:           `[{"name":"param2","value":"world"}]`,
			},
			ResourceReferences: []*model.ResourceReference{
				{ResourceUUID: "run123", ResourceType: model.RunResourceType, ReferenceUUID: "job123",
					ReferenceName: "j123", ReferenceType: model.JobResourceType, Relationship: model.CreatorRelationship},
			},
		},
		PipelineRuntime: model.PipelineRuntime{WorkflowRuntimeManifest: "workflow123"},
	}
	apiRun := ToApiRunDetail(modelRun)
	expectedApiRun := &apiv1beta1.RunDetail{
		Run: &apiv1beta1.Run{
			Id:           "run123",
			Name:         "displayName123",
			StorageState: apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:    &timestamp.Timestamp{Seconds: 1},
			ScheduledAt:  &timestamp.Timestamp{Seconds: 1},
			FinishedAt:   &timestamp.Timestamp{Seconds: 1},
			Status:       "running",
			PipelineSpec: &apiv1beta1.PipelineSpec{
				WorkflowManifest: "manifest",
				Parameters:       []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
			},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "job123"},
					Name: "j123", Relationship: apiv1beta1.Relationship_CREATOR},
			},
		},
		PipelineRuntime: &apiv1beta1.PipelineRuntime{
			WorkflowManifest: "workflow123",
		},
	}
	assert.Equal(t, expectedApiRun, apiRun)
}

func TesttoApiRunsV1(t *testing.T) {
	metric1 := &model.RunMetric{
		Name:        "metric-1",
		NodeID:      "node-1",
		NumberValue: 0.88,
		Format:      "RAW",
	}
	metric2 := &model.RunMetric{
		Name:        "metric-2",
		NodeID:      "node-2",
		NumberValue: 0.99,
		Format:      "PERCENTAGE",
	}
	apiMetric1 := &apiv1beta1.RunMetric{
		Name:   metric1.Name,
		NodeId: metric1.NodeID,
		Value:  &apiv1beta1.RunMetric_NumberValue{NumberValue: metric1.NumberValue},
		Format: apiv1beta1.RunMetric_RAW,
	}
	apiMetric2 := &apiv1beta1.RunMetric{
		Name:   metric2.Name,
		NodeId: metric2.NodeID,
		Value:  &apiv1beta1.RunMetric_NumberValue{NumberValue: metric2.NumberValue},
		Format: apiv1beta1.RunMetric_PERCENTAGE,
	}
	modelRun1 := model.Run{
		UUID:             "run1",
		Name:             "name1",
		StorageState:     apiv1beta1.Run_STORAGESTATE_AVAILABLE.String(),
		DisplayName:      "displayName1",
		Namespace:        "ns1",
		CreatedAtInSec:   1,
		ScheduledAtInSec: 1,
		Conditions:       "running",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: "manifest",
		},
		ResourceReferences: []*model.ResourceReference{
			{ResourceUUID: "run1", ResourceType: model.RunResourceType, ReferenceUUID: "job1",
				ReferenceName: "j1", ReferenceType: model.JobResourceType, Relationship: model.CreatorRelationship},
		},
		Metrics: []*model.RunMetric{metric1, metric2},
	}
	modelRun2 := model.Run{
		UUID:             "run2",
		Name:             "name2",
		StorageState:     apiv1beta1.Run_STORAGESTATE_AVAILABLE.String(),
		DisplayName:      "displayName2",
		Namespace:        "ns2",
		CreatedAtInSec:   2,
		ScheduledAtInSec: 2,
		Conditions:       "done",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: "manifest",
		},
		ResourceReferences: []*model.ResourceReference{
			{ResourceUUID: "run2", ResourceType: model.RunResourceType, ReferenceUUID: "job2",
				ReferenceName: "j2", ReferenceType: model.JobResourceType, Relationship: model.CreatorRelationship},
		},
		Metrics: []*model.RunMetric{metric2},
	}
	apiRuns := ToApiRuns([]*model.Run{&modelRun1, &modelRun2})
	expectedApiRun := []*apiv1beta1.Run{
		{
			Id:           "run1",
			Name:         "displayName1",
			StorageState: apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:    &timestamp.Timestamp{Seconds: 1},
			ScheduledAt:  &timestamp.Timestamp{Seconds: 1},
			FinishedAt:   &timestamp.Timestamp{},
			Status:       "running",
			PipelineSpec: &apiv1beta1.PipelineSpec{
				WorkflowManifest: "manifest",
			},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "job1"},
					Name: "j1", Relationship: apiv1beta1.Relationship_CREATOR},
			},
			Metrics: []*apiv1beta1.RunMetric{apiMetric1, apiMetric2},
		},
		{
			Id:           "run2",
			Name:         "displayName2",
			StorageState: apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:    &timestamp.Timestamp{Seconds: 2},
			ScheduledAt:  &timestamp.Timestamp{Seconds: 2},
			FinishedAt:   &timestamp.Timestamp{},
			Status:       "done",
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "job2"},
					Name: "j2", Relationship: apiv1beta1.Relationship_CREATOR},
			},
			PipelineSpec: &apiv1beta1.PipelineSpec{
				WorkflowManifest: "manifest",
			},
			Metrics: []*apiv1beta1.RunMetric{apiMetric2},
		},
	}
	assert.Equal(t, expectedApiRun, apiRuns)
}

func TestToApiTask(t *testing.T) {
	modelTask := &model.Task{
		UUID:              common.DefaultFakeUUID,
		Namespace:         "",
		PipelineName:      "pipeline/my-pipeline",
		RunUUID:           common.NonDefaultFakeUUID,
		MLMDExecutionID:   "1",
		CreatedTimestamp:  1,
		FinishedTimestamp: 2,
		Fingerprint:       "123",
	}
	apiTask := ToApiTask(modelTask)
	expectedApiTask := &apiv1beta1.Task{
		Id:              common.DefaultFakeUUID,
		Namespace:       "",
		PipelineName:    "pipeline/my-pipeline",
		RunId:           common.NonDefaultFakeUUID,
		MlmdExecutionID: "1",
		CreatedAt:       &timestamp.Timestamp{Seconds: 1},
		FinishedAt:      &timestamp.Timestamp{Seconds: 2},
		Fingerprint:     "123",
	}

	assert.Equal(t, expectedApiTask, apiTask)
}

func TestToApiTasks(t *testing.T) {
	modelTask1 := model.Task{
		UUID:              "123e4567-e89b-12d3-a456-426655440000",
		Namespace:         "ns1",
		PipelineName:      "namespace/ns1/pipeline/my-pipeline-1",
		RunUUID:           "123e4567-e89b-12d3-a456-426655440001",
		MLMDExecutionID:   "1",
		CreatedTimestamp:  1,
		FinishedTimestamp: 2,
		Fingerprint:       "123",
	}
	modelTask2 := model.Task{
		UUID:              "123e4567-e89b-12d3-a456-426655440002",
		Namespace:         "ns2",
		PipelineName:      "namespace/ns1/pipeline/my-pipeline-2",
		RunUUID:           "123e4567-e89b-12d3-a456-426655440003",
		MLMDExecutionID:   "2",
		CreatedTimestamp:  3,
		FinishedTimestamp: 4,
		Fingerprint:       "124",
	}

	apiTasks := ToApiTasks([]*model.Task{&modelTask1, &modelTask2})
	expectedApiTasks := []*apiv1beta1.Task{
		{
			Id:              "123e4567-e89b-12d3-a456-426655440000",
			Namespace:       "ns1",
			PipelineName:    "namespace/ns1/pipeline/my-pipeline-1",
			RunId:           "123e4567-e89b-12d3-a456-426655440001",
			MlmdExecutionID: "1",
			CreatedAt:       &timestamp.Timestamp{Seconds: 1},
			FinishedAt:      &timestamp.Timestamp{Seconds: 2},
			Fingerprint:     "123",
		},
		{
			Id:              "123e4567-e89b-12d3-a456-426655440002",
			Namespace:       "ns2",
			PipelineName:    "namespace/ns1/pipeline/my-pipeline-2",
			RunId:           "123e4567-e89b-12d3-a456-426655440003",
			MlmdExecutionID: "2",
			CreatedAt:       &timestamp.Timestamp{Seconds: 3},
			FinishedAt:      &timestamp.Timestamp{Seconds: 4},
			Fingerprint:     "124",
		},
	}
	assert.Equal(t, expectedApiTasks, apiTasks)
}

func TestCronScheduledJobToApiJob(t *testing.T) {
	modelJob := model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		Name:        "name1",
		Enabled:     true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				Cron:                       util.StringPointer("1 * *"),
			},
		},
		MaxConcurrency: 1,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			Parameters:   `[{"name":"param2","value":"world"}]`,
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		ResourceReferences: []*model.ResourceReference{
			{ResourceUUID: "job1", ResourceType: model.JobResourceType, ReferenceUUID: "experiment1", ReferenceName: "e1",
				ReferenceType: model.ExperimentResourceType, Relationship: model.OwnerRelationship},
		},
	}
	apiJob := ToApiJob(&modelJob)
	expectedJob := &apiv1beta1.Job{
		Id:             "job1",
		Name:           "name 1",
		Enabled:        true,
		CreatedAt:      &timestamp.Timestamp{Seconds: 1},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 1},
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * *",
			}}},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			Parameters:   []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
			PipelineId:   "1",
			PipelineName: "p1",
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "experiment1"},
				Name: "e1", Relationship: apiv1beta1.Relationship_OWNER},
		},
	}
	assert.Equal(t, expectedJob, apiJob)
}

func TestPeriodicScheduledJobToApiJob(t *testing.T) {
	modelJob := model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		Name:        "name1",
		Enabled:     true,
		Trigger: model.Trigger{
			PeriodicSchedule: model.PeriodicSchedule{
				PeriodicScheduleStartTimeInSec: util.Int64Pointer(1),
				IntervalSecond:                 util.Int64Pointer(3),
			},
		},
		MaxConcurrency: 1,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			Parameters:   `[{"name":"param2","value":"world"}]`,
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	apiJob := ToApiJob(&modelJob)
	expectedJob := &apiv1beta1.Job{
		Id:             "job1",
		Name:           "name 1",
		Enabled:        true,
		CreatedAt:      &timestamp.Timestamp{Seconds: 1},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 1},
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_PeriodicSchedule{PeriodicSchedule: &apiv1beta1.PeriodicSchedule{
				StartTime:      &timestamp.Timestamp{Seconds: 1},
				IntervalSecond: 3,
			}}},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			Parameters:   []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
			PipelineId:   "1",
			PipelineName: "p1",
		},
	}
	assert.Equal(t, expectedJob, apiJob)
}

func TestNonScheduledJobToApiJob(t *testing.T) {
	modelJob := model.Job{
		UUID:           "job1",
		DisplayName:    "name1",
		Enabled:        true,
		Trigger:        model.Trigger{},
		MaxConcurrency: 1,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			Parameters:   `[{"name":"param2","value":"world"}]`,
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	apiJob := ToApiJob(&modelJob)
	expectedJob := &apiv1beta1.Job{
		Id:             "job1",
		Name:           "name1",
		Enabled:        true,
		CreatedAt:      &timestamp.Timestamp{Seconds: 1},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 1},
		MaxConcurrency: 1,
		Trigger:        &apiv1beta1.Trigger{},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			Parameters:   []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
			PipelineId:   "1",
			PipelineName: "p1",
		},
	}
	assert.Equal(t, expectedJob, apiJob)
}

func TestToApiJob_ErrorParsingField(t *testing.T) {
	modelJob := &model.Job{
		UUID:           "job1",
		DisplayName:    "name1",
		Enabled:        true,
		Trigger:        model.Trigger{},
		MaxConcurrency: 1,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			Parameters:   `invalid parameter format`,
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}

	apiJob := ToApiJob(modelJob)
	assert.Equal(t, "job1", apiJob.Id)
	assert.Contains(t, apiJob.Error, "InternalServerError: Parameter with wrong format is stored")
}

func TestToApiJob_V2(t *testing.T) {
	modelJob := &model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		Name:        "name1",
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
	expectedJob := &apiv1beta1.Job{
		Id:             "job1",
		Name:           "name 1",
		Enabled:        true,
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 2},
		MaxConcurrency: 2,
		NoCatchup:      true,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 2},
				Cron:      "2 * *",
			}}},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			RuntimeConfig: &apiv1beta1.PipelineSpec_RuntimeConfig{
				Parameters: map[string]*structpb.Value{
					"param1": &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "world"}},
				},
				PipelineRoot: "job-1-root",
			},
		},
	}
	apiJob := ToApiJob(modelJob)
	// Compare the string representation of ApiRuns, since these structs have internal fields
	// used only by protobuff, and may be different. The .String() method marshal all
	// exported fields into string format.
	// See https://github.com/stretchr/testify/issues/758
	assert.Equal(t, expectedJob.String(), apiJob.String())
}

func TestToApiJobs(t *testing.T) {
	modelJob1 := model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		Name:        "name1",
		Enabled:     true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				Cron:                       util.StringPointer("1 * *"),
			},
		},
		MaxConcurrency: 1,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			Parameters:   `[{"name":"param1","value":"world"}]`,
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}
	modeljob2 := model.Job{
		UUID:        "job2",
		DisplayName: "name 2",
		Name:        "name2",
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
			PipelineId:   "2",
			PipelineName: "p2",
			Parameters:   `[{"name":"param1","value":"world"}]`,
		},
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
	}
	apiJobs := ToApiJobs([]*model.Job{&modelJob1, &modeljob2})
	expectedJobs := []*apiv1beta1.Job{
		{
			Id:             "job1",
			Name:           "name 1",
			Enabled:        true,
			CreatedAt:      &timestamp.Timestamp{Seconds: 1},
			UpdatedAt:      &timestamp.Timestamp{Seconds: 1},
			MaxConcurrency: 1,
			Trigger: &apiv1beta1.Trigger{
				Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
					StartTime: &timestamp.Timestamp{Seconds: 1},
					Cron:      "1 * *",
				}}},
			PipelineSpec: &apiv1beta1.PipelineSpec{
				Parameters:   []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
				PipelineId:   "1",
				PipelineName: "p1",
			},
		},
		{
			Id:             "job2",
			Name:           "name 2",
			Enabled:        true,
			CreatedAt:      &timestamp.Timestamp{Seconds: 2},
			UpdatedAt:      &timestamp.Timestamp{Seconds: 2},
			MaxConcurrency: 2,
			NoCatchup:      true,
			Trigger: &apiv1beta1.Trigger{
				Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
					StartTime: &timestamp.Timestamp{Seconds: 2},
					Cron:      "2 * *",
				}}},
			PipelineSpec: &apiv1beta1.PipelineSpec{
				Parameters:   []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
				PipelineId:   "2",
				PipelineName: "p2",
			},
		},
	}
	assert.Equal(t, expectedJobs, apiJobs)
}

func TestToApiRunMetric(t *testing.T) {
	modelRunMetric := &model.RunMetric{
		Name:        "metric-1",
		NodeID:      "node-1",
		NumberValue: 0.88,
		Format:      "RAW",
	}

	actualAPIRunMetric := ToApiRunMetric(modelRunMetric)

	expectedAPIRunMetric := &apiv1beta1.RunMetric{
		Name:   "metric-1",
		NodeId: "node-1",
		Value: &apiv1beta1.RunMetric_NumberValue{
			NumberValue: 0.88,
		},
		Format: apiv1beta1.RunMetric_RAW,
	}
	assert.Equal(t, expectedAPIRunMetric, actualAPIRunMetric)
}

func TestToApiRunMetric_UnknownFormat(t *testing.T) {
	// This can happen if we accidentally remove an existing format value from proto.
	modelRunMetric := &model.RunMetric{
		Name:        "metric-1",
		NodeID:      "node-1",
		NumberValue: 0.88,
		Format:      "NotExistValue",
	}

	actualAPIRunMetric := ToApiRunMetric(modelRunMetric)

	expectedAPIRunMetric := &apiv1beta1.RunMetric{
		Name:   "metric-1",
		NodeId: "node-1",
		Value: &apiv1beta1.RunMetric_NumberValue{
			NumberValue: 0.88,
		},
		// Expect return UNSPECIFIED for unknown format
		Format: apiv1beta1.RunMetric_UNSPECIFIED,
	}
	assert.Equal(t, expectedAPIRunMetric, actualAPIRunMetric)
}

func TestToApiResourceReferences(t *testing.T) {
	resourceReferences := []*model.ResourceReference{
		{ResourceUUID: "run1", ResourceType: model.RunResourceType, ReferenceUUID: "experiment1",
			ReferenceName: "e1", ReferenceType: model.ExperimentResourceType, Relationship: model.OwnerRelationship},
		{ResourceUUID: "run1", ResourceType: model.RunResourceType, ReferenceUUID: "job1",
			ReferenceName: "j1", ReferenceType: model.JobResourceType, Relationship: model.OwnerRelationship},
		{ResourceUUID: "run1", ResourceType: model.RunResourceType, ReferenceUUID: "pipelineversion1",
			ReferenceName: "k1", ReferenceType: model.PipelineVersionResourceType, Relationship: model.OwnerRelationship},
	}
	expectedApiResourceReferences := []*apiv1beta1.ResourceReference{
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "experiment1"},
			Name: "e1", Relationship: apiv1beta1.Relationship_OWNER},
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "job1"},
			Name: "j1", Relationship: apiv1beta1.Relationship_OWNER},
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_PIPELINE_VERSION, Id: "pipelineversion1"},
			Name: "k1", Relationship: apiv1beta1.Relationship_OWNER},
	}
	assert.Equal(t, expectedApiResourceReferences, toApiResourceReferences(resourceReferences))
}

func TestToApiExperimentsV1(t *testing.T) {
	exp1 := &model.Experiment{
		UUID:           "exp1",
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "experiment1 was created using V2 APIV1BETA1.",
		StorageState:   "AVAILABLE",
	}
	exp2 := &model.Experiment{
		UUID:           "exp2",
		CreatedAtInSec: 2,
		Name:           "experiment2",
		Description:    "experiment2 was created using V2 APIV1BETA1.",
		StorageState:   "ARCHIVED",
	}
	exp3 := &model.Experiment{
		UUID:           "exp3",
		CreatedAtInSec: 3,
		Name:           "experiment3",
		Description:    "experiment3 was created using V1 APIV1BETA1.",
		StorageState:   "STORAGESTATE_AVAILABLE",
	}
	exp4 := &model.Experiment{
		UUID:           "exp4",
		CreatedAtInSec: 4,
		Name:           "experiment4",
		Description:    "experiment4 was created using V1 APIV1BETA1.",
		StorageState:   "STORAGESTATE_ARCHIVED",
	}
	apiExps := ToApiExperimentsV1([]*model.Experiment{exp1, exp2, exp3, exp4})
	expectedApiExps := []*apiv1beta1.Experiment{
		{
			Id:           "exp1",
			Name:         "experiment1",
			Description:  "experiment1 was created using V2 APIV1BETA1.",
			CreatedAt:    &timestamp.Timestamp{Seconds: 1},
			StorageState: apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_AVAILABLE"]),
		},
		{
			Id:           "exp2",
			Name:         "experiment2",
			Description:  "experiment2 was created using V2 APIV1BETA1.",
			CreatedAt:    &timestamp.Timestamp{Seconds: 2},
			StorageState: apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_ARCHIVED"]),
		},
		{
			Id:           "exp3",
			Name:         "experiment3",
			Description:  "experiment3 was created using V1 APIV1BETA1.",
			CreatedAt:    &timestamp.Timestamp{Seconds: 3},
			StorageState: apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_AVAILABLE"]),
		},
		{
			Id:           "exp4",
			Name:         "experiment4",
			Description:  "experiment4 was created using V1 APIV1BETA1.",
			CreatedAt:    &timestamp.Timestamp{Seconds: 4},
			StorageState: apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_ARCHIVED"]),
		},
	}
	assert.Equal(t, expectedApiExps, apiExps)
}

func TestToApiExperiments(t *testing.T) {
	exp1 := &model.Experiment{
		UUID:           "exp1",
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "My name is experiment1",
		StorageState:   "AVAILABLE",
	}
	exp2 := &model.Experiment{
		UUID:           "exp2",
		CreatedAtInSec: 2,
		Name:           "experiment2",
		Description:    "My name is experiment2",
		StorageState:   "ARCHIVED",
	}
	exp3 := &model.Experiment{
		UUID:           "exp3",
		CreatedAtInSec: 1,
		Name:           "experiment3",
		Description:    "experiment3 was created using V1 APIV1BETA1.",
		StorageState:   "STORAGESTATE_AVAILABLE",
	}
	exp4 := &model.Experiment{
		UUID:           "exp4",
		CreatedAtInSec: 2,
		Name:           "experiment4",
		Description:    "experiment4 was created using V1 APIV1BETA1.",
		StorageState:   "STORAGESTATE_ARCHIVED",
	}
	apiExps := ToApiExperiments([]*model.Experiment{exp1, exp2, exp3, exp4})
	expectedApiExps := []*apiv2beta1.Experiment{
		{
			ExperimentId: "exp1",
			DisplayName:  "experiment1",
			Description:  "My name is experiment1",
			CreatedAt:    &timestamp.Timestamp{Seconds: 1},
			StorageState: apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["AVAILABLE"]),
		},
		{
			ExperimentId: "exp2",
			DisplayName:  "experiment2",
			Description:  "My name is experiment2",
			CreatedAt:    &timestamp.Timestamp{Seconds: 2},
			StorageState: apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["ARCHIVED"]),
		},
		{
			ExperimentId: "exp3",
			DisplayName:  "experiment3",
			Description:  "experiment3 was created using V1 APIV1BETA1.",
			CreatedAt:    &timestamp.Timestamp{Seconds: 1},
			StorageState: apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["AVAILABLE"]),
		},
		{
			ExperimentId: "exp4",
			DisplayName:  "experiment4",
			Description:  "experiment4 was created using V1 APIV1BETA1.",
			CreatedAt:    &timestamp.Timestamp{Seconds: 2},
			StorageState: apiv2beta1.Experiment_StorageState(apiv2beta1.Experiment_StorageState_value["ARCHIVED"]),
		},
	}
	assert.Equal(t, expectedApiExps, apiExps)
}

func TestToApiParameters(t *testing.T) {
	expectedApiParameters := []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}}
	modelParameters := `[{"name":"param2","value":"world"}]`
	actualApiParameters, err := toApiParameters(modelParameters)
	assert.Nil(t, err)
	assert.Equal(t, expectedApiParameters, actualApiParameters)
}

func TestToApiRuntimeConfigV1(t *testing.T) {
	listParams := []interface{}{1, 2, 3}
	v2RuntimeListParams, _ := structpb.NewList(listParams)

	structParams := map[string]interface{}{"structParam1": "hello", "structParam2": 32}
	v2RuntimeStructParams, _ := structpb.NewStruct(structParams)

	// Test all parameters types converted to model.RuntimeConfig.Parameters, which is string type
	runtimeParameters := map[string]*structpb.Value{
		"param2": structpb.NewStringValue("world"),
		"param3": structpb.NewBoolValue(true),
		"param4": structpb.NewListValue(v2RuntimeListParams),
		"param5": structpb.NewNumberValue(12),
		"param6": structpb.NewStructValue(v2RuntimeStructParams),
	}
	expectedRuntimeConfig := &apiv1beta1.PipelineSpec_RuntimeConfig{
		Parameters:   runtimeParameters,
		PipelineRoot: "model-pipeline-root",
	}
	modelRuntimeConfig := model.RuntimeConfig{
		Parameters:   "{\"param2\":\"world\",\"param3\":true,\"param4\":[1,2,3],\"param5\":12,\"param6\":{\"structParam1\":\"hello\",\"structParam2\":32}}",
		PipelineRoot: "model-pipeline-root",
	}
	actualRuntimeConfig, err := toApiRuntimeConfigV1(modelRuntimeConfig)
	assert.Nil(t, err)
	// Compare the string representation of ApiRuntimeConfig, since these structs have fields
	// used only by protobuff, and may be different. The .String() method marshal all
	// exported fields into string format.
	// See https://github.com/stretchr/testify/issues/758
	assert.Equal(t, expectedRuntimeConfig.String(), actualRuntimeConfig.String())
}

func TestToApiRecurringRun(t *testing.T) {
	modelJob := &model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		Name:        "name1",
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
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 2},
		MaxConcurrency: 2,
		NoCatchup:      true,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 2},
				Cron:      "2 * *",
			}}},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineId{PipelineId: "1"},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			Parameters: map[string]*structpb.Value{
				"param1": &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "world"}},
			},
			PipelineRoot: "job-1-root",
		},
	}
	apiRecurringRun := ToApiRecurringRun(modelJob)
	// Compare the string representation of ApiRuns, since these structs have internal fields
	// used only by protobuff, and may be different. The .String() method marshal all
	// exported fields into string format.
	// See https://github.com/stretchr/testify/issues/758
	assert.Equal(t, expectedRecurringRun.String(), apiRecurringRun.String())
}
