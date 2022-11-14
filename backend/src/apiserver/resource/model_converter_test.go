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

package resource

import (
	"strings"
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiV1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
)

func initResourceManager() (*FakeClientManager, *ResourceManager) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	return store, NewResourceManager(store)
}

func TestToModelExperiment(t *testing.T) {
	store, manager := initResourceManager()
	defer store.Close()

	tests := []struct {
		name                    string
		experiment              interface{}
		wantError               bool
		errorMessage            string
		expectedModelExperiment *model.Experiment
	}{
		{
			"API V1beta1: No resource references",
			&apiV1beta1.Experiment{
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
			&apiV1beta1.Experiment{
				Name:        "exp1",
				Description: "This is an experiment",
				ResourceReferences: []*apiV1beta1.ResourceReference{
					&apiV1beta1.ResourceReference{
						Key: &apiV1beta1.ResourceKey{
							Type: apiV1beta1.ResourceType_NAMESPACE,
							Id:   "ns1",
						},
						Relationship: apiV1beta1.Relationship_OWNER,
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
			&apiV1beta1.Experiment{
				Name:        "exp1",
				Description: "This is an experiment",
				ResourceReferences: []*apiV1beta1.ResourceReference{
					&apiV1beta1.ResourceReference{
						Key: &apiV1beta1.ResourceKey{
							Type: apiV1beta1.ResourceType_EXPERIMENT,
							Id:   "invalid",
						},
						Relationship: apiV1beta1.Relationship_OWNER,
					},
				},
			},
			true,
			"Invalid resource references for experiment",
			nil,
		},
		{
			"API V2beta1: Happy pass",
			&apiV2beta1.Experiment{
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
		modelExperiment, err := manager.ToModelExperiment(tc.experiment)
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

func TestToModelRunMetric(t *testing.T) {
	store, manager := initResourceManager()
	defer store.Close()
	apiRunMetric := &apiV1beta1.RunMetric{
		Name:   "metric-1",
		NodeId: "node-1",
		Value: &apiV1beta1.RunMetric_NumberValue{
			NumberValue: 0.88,
		},
		Format: apiV1beta1.RunMetric_RAW,
	}

	actualModelRunMetric := manager.ToModelRunMetric(apiRunMetric, "run-1")

	expectedModelRunMetric := &model.RunMetric{
		RunUUID:     "run-1",
		Name:        "metric-1",
		NodeID:      "node-1",
		NumberValue: 0.88,
		Format:      "RAW",
	}
	assert.Equal(t, expectedModelRunMetric, actualModelRunMetric)
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
		apiRun                 *apiV1beta1.Run
		workflow               *util.Workflow
		manifest               string
		templateType           template.TemplateType
		expectedModelRunDetail *model.RunDetail
	}{
		{
			name: "v1",
			apiRun: &apiV1beta1.Run{
				Id:          "run1",
				Name:        "name1",
				Description: "this is a run",
				PipelineSpec: &apiV1beta1.PipelineSpec{
					Parameters: []*apiV1beta1.Parameter{{Name: "param2", Value: "world"}},
				},
				ResourceReferences: []*apiV1beta1.ResourceReference{
					{
						Key:          &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
						Relationship: apiV1beta1.Relationship_OWNER}},
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
					DisplayName:    "name1",
					Name:           "workflow-name",
					Conditions:     "running",
					Description:    "this is a run",
					PipelineSpec: model.PipelineSpec{
						WorkflowSpecManifest: "workflow spec",
						Parameters:           `[{"name":"param2","value":"world"}]`,
					},
					ResourceReferences: []*model.ResourceReference{
						{
							ResourceUUID:  "123",
							ResourceType:  common.Run,
							ReferenceUUID: experiment.UUID,
							ReferenceName: experiment.Name,
							ReferenceType: common.Experiment,
							Relationship:  common.Owner},
					},
				},
				PipelineRuntime: model.PipelineRuntime{
					WorkflowRuntimeManifest: util.NewWorkflow(&v1alpha1.Workflow{
						ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "123"},
						Status:     v1alpha1.WorkflowStatus{Phase: "running"},
					}).ToStringForStore(),
				},
			},
		},
		{
			name: "v2",
			apiRun: &apiV1beta1.Run{
				Id:          "run1",
				Name:        "name1",
				Description: "this is a run",
				PipelineSpec: &apiV1beta1.PipelineSpec{
					RuntimeConfig: &apiV1beta1.PipelineSpec_RuntimeConfig{
						Parameters: v2RuntimeParams,
					}},
				ResourceReferences: []*apiV1beta1.ResourceReference{
					{
						Key:          &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID},
						Relationship: apiV1beta1.Relationship_OWNER}},
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
					DisplayName:    "name1",
					Name:           "workflow-name",
					Conditions:     "running",
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
							ResourceType:  common.Run,
							ReferenceUUID: experiment.UUID,
							ReferenceName: experiment.Name,
							ReferenceType: common.Experiment,
							Relationship:  common.Owner},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelRunDetail, err := manager.ToModelRunDetail(tt.apiRun, "123", tt.workflow, tt.manifest, tt.templateType)
			assert.Nil(t, err)
			assert.Equal(t, tt.expectedModelRunDetail, modelRunDetail)
		})
	}

}

func TestToModelJob(t *testing.T) {
	store, manager, experiment, pipeline := initWithExperimentAndPipeline(t)
	defer store.Close()

	tests := []struct {
		name             string
		apiJob           *apiV1beta1.Job
		swf              *util.ScheduledWorkflow
		manifest         string
		templateType     template.TemplateType
		expectedModelJob *model.Job
	}{
		{name: "v1",
			apiJob: &apiV1beta1.Job{
				Name:           "name1",
				Enabled:        true,
				MaxConcurrency: 1,
				NoCatchup:      true,
				Trigger: &apiV1beta1.Trigger{
					Trigger: &apiV1beta1.Trigger_CronSchedule{CronSchedule: &apiV1beta1.CronSchedule{
						StartTime: &timestamp.Timestamp{Seconds: 1},
						Cron:      "1 * * * *",
					}}},
				ResourceReferences: []*apiV1beta1.ResourceReference{
					{Key: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: apiV1beta1.Relationship_OWNER},
				},
				PipelineSpec: &apiV1beta1.PipelineSpec{PipelineId: pipeline.UUID, Parameters: []*apiV1beta1.Parameter{{Name: "param2", Value: "world"}}},
			},
			swf: util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
				ObjectMeta: v1.ObjectMeta{
					Name:      "swf_name",
					Namespace: "swf_namespace",
					UID:       "swf_123",
				},
				Status: swfapi.ScheduledWorkflowStatus{
					Conditions: []swfapi.ScheduledWorkflowCondition{{Type: swfapi.ScheduledWorkflowEnabled}}},
			}),
			manifest:     "workflow spec",
			templateType: template.V1,
			expectedModelJob: &model.Job{
				UUID:        "swf_123",
				Name:        "swf_name",
				Namespace:   "swf_namespace",
				Conditions:  "Enabled",
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
					WorkflowSpecManifest: "workflow spec",
					Parameters:           `[{"name":"param2","value":"world"}]`,
				},
				ResourceReferences: []*model.ResourceReference{
					{
						ResourceUUID:  "swf_123",
						ResourceType:  common.Job,
						ReferenceUUID: experiment.UUID,
						ReferenceName: experiment.Name,
						ReferenceType: common.Experiment,
						Relationship:  common.Owner},
				},
			},
		},
		{name: "v2",
			apiJob: &apiV1beta1.Job{
				Name:           "name1",
				Enabled:        true,
				MaxConcurrency: 1,
				NoCatchup:      true,
				Trigger: &apiV1beta1.Trigger{
					Trigger: &apiV1beta1.Trigger_CronSchedule{CronSchedule: &apiV1beta1.CronSchedule{
						StartTime: &timestamp.Timestamp{Seconds: 1},
						Cron:      "1 * * * *",
					}}},
				ResourceReferences: []*apiV1beta1.ResourceReference{
					{Key: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_EXPERIMENT, Id: experiment.UUID}, Relationship: apiV1beta1.Relationship_OWNER},
				},
				PipelineSpec: &apiV1beta1.PipelineSpec{PipelineId: pipeline.UUID, RuntimeConfig: &apiV1beta1.PipelineSpec_RuntimeConfig{Parameters: map[string]*structpb.Value{
					"param2": &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "world"}},
				}}},
			},
			swf: util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
				ObjectMeta: v1.ObjectMeta{
					Name:      "swf_name",
					Namespace: "swf_namespace",
					UID:       "swf_123",
				},
				Status: swfapi.ScheduledWorkflowStatus{
					Conditions: []swfapi.ScheduledWorkflowCondition{{Type: swfapi.ScheduledWorkflowEnabled}}},
			}),
			manifest:     "pipeline spec",
			templateType: template.V2,
			expectedModelJob: &model.Job{
				UUID:        "swf_123",
				Name:        "swf_name",
				Namespace:   "swf_namespace",
				Conditions:  "Enabled",
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
						ResourceUUID:  "swf_123",
						ResourceType:  common.Job,
						ReferenceUUID: experiment.UUID,
						ReferenceName: experiment.Name,
						ReferenceType: common.Experiment,
						Relationship:  common.Owner},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelJob, err := manager.ToModelJob(tt.apiJob, tt.swf, tt.manifest, tt.templateType)
			assert.Nil(t, err)
			assert.Equal(t, tt.expectedModelJob, modelJob)
		})
	}
}

func TestToModelResourceReferences(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	refs, err := manager.toModelResourceReferences("r1", common.Run, []*apiV1beta1.ResourceReference{
		{Key: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_EXPERIMENT, Id: DefaultFakeUUID}, Relationship: apiV1beta1.Relationship_OWNER},
		{Key: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_JOB, Id: job.UUID}, Relationship: apiV1beta1.Relationship_CREATOR},
	})
	assert.Nil(t, err)
	expectedRefs := []*model.ResourceReference{
		{ResourceUUID: "r1", ResourceType: common.Run,
			ReferenceUUID: DefaultFakeUUID, ReferenceName: "e1", ReferenceType: common.Experiment, Relationship: common.Owner},
		{ResourceUUID: "r1", ResourceType: common.Run,
			ReferenceUUID: job.UUID, ReferenceName: "j1", ReferenceType: common.Job, Relationship: common.Creator},
	}
	assert.Equal(t, expectedRefs, refs)
}

func TestToModelResourceReferences_UnknownRefType(t *testing.T) {
	store, manager, _ := initWithJob(t)
	defer store.Close()

	_, err := manager.toModelResourceReferences("r1", common.Run, []*apiV1beta1.ResourceReference{
		{Key: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_UNKNOWN_RESOURCE_TYPE, Id: "e1"}, Relationship: apiV1beta1.Relationship_OWNER},
		{Key: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_JOB, Id: "j1"}, Relationship: apiV1beta1.Relationship_CREATOR},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to convert reference type")
}

func TestToModelResourceReferences_NamespaceRef(t *testing.T) {
	store, manager, _ := initWithJob(t)
	defer store.Close()

	modelRefs, err := manager.toModelResourceReferences("r1", common.Run, []*apiV1beta1.ResourceReference{
		{Key: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_NAMESPACE, Id: "e1"}, Relationship: apiV1beta1.Relationship_OWNER},
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(modelRefs))
}

func TestToModelResourceReferences_UnknownRelationship(t *testing.T) {
	store, manager, _ := initWithJob(t)
	defer store.Close()
	_, err := manager.toModelResourceReferences("r1", common.Run, []*apiV1beta1.ResourceReference{
		{Key: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_EXPERIMENT, Id: "e1"}, Relationship: apiV1beta1.Relationship_UNKNOWN_RELATIONSHIP},
		{Key: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_JOB, Id: "j1"}, Relationship: apiV1beta1.Relationship_CREATOR},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to convert relationship")
}

func TestToModelResourceReferences_ReferredJobNotFound(t *testing.T) {
	store, manager, _ := initWithJob(t)
	defer store.Close()
	_, err := manager.toModelResourceReferences("r1", common.Run, []*apiV1beta1.ResourceReference{
		{Key: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_EXPERIMENT, Id: "e1"}, Relationship: apiV1beta1.Relationship_OWNER},
		{Key: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_JOB, Id: "j2"}, Relationship: apiV1beta1.Relationship_CREATOR},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to find the referred resource")
}

func TestToModelResourceReferences_ReferredExperimentNotFound(t *testing.T) {
	store, manager, _ := initWithJob(t)
	defer store.Close()
	_, err := manager.toModelResourceReferences("r1", common.Run, []*apiV1beta1.ResourceReference{
		{Key: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_EXPERIMENT, Id: "e2"}, Relationship: apiV1beta1.Relationship_OWNER},
		{Key: &apiV1beta1.ResourceKey{Type: apiV1beta1.ResourceType_JOB, Id: "j1"}, Relationship: apiV1beta1.Relationship_CREATOR},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to find the referred resource")
}

func TestToModelPipelineVersion(t *testing.T) {
	store, manager := initResourceManager()
	defer store.Close()
	apiPipelineVersion := &apiV1beta1.PipelineVersion{
		Id:            "pipelineversion1",
		CreatedAt:     &timestamp.Timestamp{Seconds: 1},
		Parameters:    []*apiV1beta1.Parameter{},
		CodeSourceUrl: "http://repo/11111",
		ResourceReferences: []*apiV1beta1.ResourceReference{
			&apiV1beta1.ResourceReference{
				Key: &apiV1beta1.ResourceKey{
					Id:   "pipeline1",
					Type: apiV1beta1.ResourceType_PIPELINE,
				},
				Relationship: apiV1beta1.Relationship_OWNER,
			},
		},
	}

	convertedModelPipelineVersion, _ := manager.ToModelPipelineVersion(
		apiPipelineVersion)

	expectedModelPipelineVersion := &model.PipelineVersion{
		UUID:           "pipelineversion1",
		CreatedAtInSec: 1,
		Parameters:     "",
		PipelineId:     "pipeline1",
		CodeSourceUrl:  "http://repo/11111",
	}

	assert.Equal(t, convertedModelPipelineVersion, expectedModelPipelineVersion)
}
