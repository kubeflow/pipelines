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
	"strings"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/google/go-cmp/cmp"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			"No resource references v1",
			&apiv1beta1.Experiment{
				Name:        "exp1",
				Description: "This is an experiment",
			},
			false,
			"",
			&model.Experiment{
				Name:         "exp1",
				Description:  "This is an experiment",
				Namespace:    "",
				StorageState: model.StorageStateAvailable,
			},
		},
		{
			"Valid resource references v1",
			&apiv1beta1.Experiment{
				Name:        "exp1",
				Description: "This is an experiment",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
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
				Name:         "exp1",
				Description:  "This is an experiment",
				Namespace:    "ns1",
				StorageState: model.StorageStateAvailable,
			},
		},
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
			"Wrong API type",
			&model.Experiment{
				Name:        "test",
				Description: "API V2beta1 test experiment",
				Namespace:   "ns2",
			},
			true,
			"UnknownApiVersionError: Error using Experiment with *model.Experiment",
			nil,
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
		{
			"missing name v1",
			&apiv1beta1.Experiment{
				Name:        "",
				Description: "API V2beta1 test experiment",
			},
			true,
			"Experiment must have a non-empty name",
			nil,
		},
		{
			"name too long v1",
			&apiv1beta1.Experiment{
				Name:        strings.Repeat("a", 129), // Max is 128
				Description: "This is an experiment with a very long name",
			},
			true,
			"Experiment.Name length cannot exceed 128",
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
		pipeline              interface{}
		wantError             bool
		errorMessage          string
		expectedModelPipeline *model.Pipeline
	}{
		{
			"No resource references v1",
			&apiv1beta1.Pipeline{
				Name:        "p1",
				Description: "This is a pipeline1",
			},
			false,
			"",
			&model.Pipeline{
				Name:        "p1",
				DisplayName: "p1",
				Description: "This is a pipeline1",
				Status:      model.PipelineCreating,
			},
		},
		{
			"Invalid resource reference v1",
			&apiv1beta1.Pipeline{
				Name:        "p2",
				Description: "This is a pipeline2",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "exp1"},
						Relationship: apiv1beta1.Relationship_CREATOR,
					},
				},
			},
			false,
			"",
			&model.Pipeline{
				Name:        "p2",
				DisplayName: "p2",
				Description: "This is a pipeline2",
				Status:      model.PipelineCreating,
			},
		},
		{
			"Invalid relationship reference v1",
			&apiv1beta1.Pipeline{
				Name:        "p3",
				Description: "This is a pipeline3",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: apiv1beta1.Relationship_CREATOR,
					},
				},
			},
			false,
			"",
			&model.Pipeline{
				Name:        "p3",
				DisplayName: "p3",
				Description: "This is a pipeline3",
				Status:      model.PipelineCreating,
				Namespace:   "ns1",
			},
		},
		{
			"Valid reference v1",
			&apiv1beta1.Pipeline{
				Name:        "p4",
				Description: "This is a pipeline4",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			false,
			"",
			&model.Pipeline{
				Name:        "p4",
				DisplayName: "p4",
				Description: "This is a pipeline4",
				Status:      model.PipelineCreating,
				Namespace:   "ns1",
			},
		},
		{
			"Empty valid reference v1",
			&apiv1beta1.Pipeline{
				Name:        "p5",
				Description: "This is a pipeline5",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: ""},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			false,
			"",
			&model.Pipeline{
				Name:        "p5",
				DisplayName: "p5",
				Description: "This is a pipeline5",
				Status:      model.PipelineCreating,
				Namespace:   "",
			},
		},
		{
			name: "name too long v1",
			pipeline: &apiv1beta1.Pipeline{
				Name:        strings.Repeat("a", 129), // Max 128
				Description: "desc",
			},
			wantError:             true,
			errorMessage:          "Pipeline.Name length cannot exceed 128",
			expectedModelPipeline: nil,
		},
		{
			name: "namespace too long v1",
			pipeline: &apiv1beta1.Pipeline{
				Name:        "p1",
				Description: "desc",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key: &apiv1beta1.ResourceKey{
							Type: apiv1beta1.ResourceType_NAMESPACE,
							Id:   strings.Repeat("n", 64), // Max 63
						},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			wantError:             true,
			errorMessage:          "Pipeline.Namespace length cannot exceed 63",
			expectedModelPipeline: nil,
		},
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

func TestToModelRunDetail(t *testing.T) {
	listParams := []interface{}{1, 2, 3}
	v2RuntimeListParams, _ := structpb.NewList(listParams)

	structParams := map[string]interface{}{"structParam1": "hello", "structParam2": 32}
	v2RuntimeStructParams, _ := structpb.NewStruct(structParams)

	// Test all parameters types converted to model.RuntimeConfig.Parameters, which is string type
	v2RuntimeParams := map[string]*structpb.Value{
		"param2": {Kind: &structpb.Value_StringValue{StringValue: "world"}},
		"param3": {Kind: &structpb.Value_BoolValue{BoolValue: true}},
		"param4": {Kind: &structpb.Value_ListValue{ListValue: v2RuntimeListParams}},
		"param5": {Kind: &structpb.Value_NumberValue{NumberValue: 12}},
		"param6": {Kind: &structpb.Value_StructValue{StructValue: v2RuntimeStructParams}},
	}

	tests := []struct {
		name                   string
		apiRun                 *apiv1beta1.Run
		workflow               *util.Workflow
		manifest               string
		templateType           template.TemplateType
		expectedModelRunDetail *model.Run
		wantErr                bool
		errMsg                 string
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
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "exp1"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_PIPELINE_VERSION, Id: "pv1"},
						Relationship: apiv1beta1.Relationship_CREATOR,
					},
				},
			},
			workflow: util.NewWorkflow(&v1alpha1.Workflow{
				ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "123"},
				Status:     v1alpha1.WorkflowStatus{Phase: "running"},
			}),
			manifest:     "workflow spec",
			templateType: template.V1,
			expectedModelRunDetail: &model.Run{
				UUID:         "run1",
				ExperimentId: "exp1",
				Namespace:    "",
				K8SName:      "",
				DisplayName:  "name1",
				Description:  "this is a run",
				RunDetails: model.RunDetails{
					State: model.RuntimeStateUnspecified,
				},
				PipelineSpec: model.PipelineSpec{
					Parameters: `[{"name":"param2","value":"world"}]`,
					RuntimeConfig: model.RuntimeConfig{
						Parameters: "",
					},
					PipelineVersionId: "pv1",
					PipelineName:      "pipelines/pv1",
				},
				StorageState: model.StorageStateAvailable,
			},
			wantErr: false,
			errMsg:  "",
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
					},
					PipelineId: "p1",
				},
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "exp1"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			workflow: util.NewWorkflow(&v1alpha1.Workflow{
				ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "123"},
				Status:     v1alpha1.WorkflowStatus{Phase: "running"},
			}),
			manifest:     "pipeline spec",
			templateType: template.V2,
			expectedModelRunDetail: &model.Run{
				UUID:         "run1",
				ExperimentId: "exp1",
				Namespace:    "",
				K8SName:      "",
				RunDetails: model.RunDetails{
					State: model.RuntimeStateUnspecified,
				},
				DisplayName: "name1",
				Description: "this is a run",
				PipelineSpec: model.PipelineSpec{
					RuntimeConfig: model.RuntimeConfig{
						// Note: for some versions of structpb.Value.MarshalJSON(), there is a trailing space after array items or struct items
						Parameters: "{\"param2\":\"world\",\"param3\":true,\"param4\":[1,2,3],\"param5\":12,\"param6\":{\"structParam1\":\"hello\",\"structParam2\":32}}",
					},
					PipelineId: "p1",
				},
				StorageState: model.StorageStateAvailable,
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "v1 namespace too long",
			apiRun: &apiv1beta1.Run{
				Id:          "run1",
				Name:        "name1",
				Description: "desc",
				PipelineSpec: &apiv1beta1.PipelineSpec{
					Parameters: []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
				},
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: strings.Repeat("n", 64)},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "exp1"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_PIPELINE_VERSION, Id: "pv1"},
						Relationship: apiv1beta1.Relationship_CREATOR,
					},
				},
			},
			workflow: util.NewWorkflow(&v1alpha1.Workflow{
				ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "123"},
				Status:     v1alpha1.WorkflowStatus{Phase: "running"},
			}),
			manifest:               "workflow spec",
			templateType:           template.V1,
			expectedModelRunDetail: nil,
			wantErr:                true,
			errMsg:                 "Run.Namespace length cannot exceed 63",
		},
		{
			name: "v1 experimentId too long",
			apiRun: &apiv1beta1.Run{
				Id:          "run1",
				Name:        "name1",
				Description: "desc",
				PipelineSpec: &apiv1beta1.PipelineSpec{
					Parameters: []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
				},
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: strings.Repeat("e", 65)},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
					{
						Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_PIPELINE_VERSION, Id: "pv1"},
						Relationship: apiv1beta1.Relationship_CREATOR,
					},
				},
			},
			workflow: util.NewWorkflow(&v1alpha1.Workflow{
				ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "123"},
				Status:     v1alpha1.WorkflowStatus{Phase: "running"},
			}),
			manifest:               "workflow spec",
			templateType:           template.V1,
			expectedModelRunDetail: nil,
			wantErr:                true,
			errMsg:                 "Run.ExperimentId length cannot exceed 64",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelRunDetail, err := toModelRun(tt.apiRun)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedModelRunDetail, modelRunDetail)
			}
		})
	}
}

func TestToModelJob(t *testing.T) {
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
						StartTime: timestamppb.New(time.Unix(1, 0)),
						Cron:      "1 * * * *",
					}},
				},
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "exp1"}, Relationship: apiv1beta1.Relationship_OWNER},
				},
				PipelineSpec: &apiv1beta1.PipelineSpec{PipelineId: "p1", Parameters: []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}}},
			},
			manifest:     "workflow spec",
			templateType: template.V1,
			expectedModelJob: &model.Job{
				DisplayName:  "name1",
				K8SName:      "name1",
				Enabled:      true,
				Conditions:   "ENABLED",
				ExperimentId: "exp1",
				Trigger: model.Trigger{
					CronSchedule: model.CronSchedule{
						CronScheduleStartTimeInSec: util.Int64Pointer(1),
						Cron:                       util.StringPointer("1 * * * *"),
					},
				},
				MaxConcurrency: 1,
				NoCatchup:      true,
				PipelineSpec: model.PipelineSpec{
					PipelineId: "p1",
					Parameters: `[{"name":"param2","value":"world"}]`,
				},
				ResourceReferences: make([]*model.ResourceReference, 0),
			},
		},
		{
			name: "v1api v2template",
			job: &apiv1beta1.Job{
				Name:           "name1",
				Enabled:        true,
				MaxConcurrency: 1,
				NoCatchup:      true,
				Trigger: &apiv1beta1.Trigger{
					Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
						StartTime: timestamppb.New(time.Unix(1, 0)),
						Cron:      "1 * * * *",
					}},
				},
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "exp1"}, Relationship: apiv1beta1.Relationship_OWNER},
				},
				PipelineSpec: &apiv1beta1.PipelineSpec{
					PipelineId: "p1",
					RuntimeConfig: &apiv1beta1.PipelineSpec_RuntimeConfig{Parameters: map[string]*structpb.Value{
						"param2": {Kind: &structpb.Value_StringValue{StringValue: "world"}},
					}},
				},
			},
			manifest:     "pipeline spec",
			templateType: template.V2,
			expectedModelJob: &model.Job{
				K8SName:      "name1",
				DisplayName:  "name1",
				Enabled:      true,
				Conditions:   "ENABLED",
				ExperimentId: "exp1",
				Trigger: model.Trigger{
					CronSchedule: model.CronSchedule{
						CronScheduleStartTimeInSec: util.Int64Pointer(1),
						Cron:                       util.StringPointer("1 * * * *"),
					},
				},
				ResourceReferences: make([]*model.ResourceReference, 0),
				MaxConcurrency:     1,
				NoCatchup:          true,
				PipelineSpec: model.PipelineSpec{
					PipelineId: "p1",
					RuntimeConfig: model.RuntimeConfig{
						Parameters: "{\"param2\":\"world\"}",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelJob, err := toModelJob(tt.job)
			assert.Nil(t, err)
			assert.Equal(t, tt.expectedModelJob, modelJob)
		})
	}
}

func TestToModelResourceReferencesV1(t *testing.T) {
	refs, err := toModelResourceReferencesV1(
		[]*apiv1beta1.ResourceReference{
			{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: DefaultFakeUUID}, Relationship: apiv1beta1.Relationship_OWNER},
			{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: DefaultFakeUUID}, Relationship: apiv1beta1.Relationship_OWNER},
		}, "r1", apiv1beta1.ResourceType_JOB,
	)
	assert.Nil(t, err)
	expectedRefs := []*model.ResourceReference{
		{
			ResourceUUID: "r1", ResourceType: model.JobResourceType,
			ReferenceUUID: DefaultFakeUUID, ReferenceType: model.ExperimentResourceType, Relationship: model.OwnerRelationship,
		},
		{
			ResourceUUID: "r1", ResourceType: model.JobResourceType,
			ReferenceUUID: DefaultFakeUUID, ReferenceType: model.NamespaceResourceType, Relationship: model.OwnerRelationship,
		},
	}
	assert.Equal(t, expectedRefs, refs)
}

func TestToModelResourceReferences_NamespaceRef(t *testing.T) {
	modelRefs, err := toModelResourceReferencesV1([]*apiv1beta1.ResourceReference{
		{
			Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "e1"},
			Relationship: apiv1beta1.Relationship_OWNER,
		},
	}, "r1", apiv1beta1.ResourceType_JOB)
	assert.Nil(t, err)
	expectedRefs := []*model.ResourceReference{
		{
			ResourceUUID:  "r1",
			ResourceType:  model.JobResourceType,
			ReferenceUUID: "e1",
			ReferenceType: model.NamespaceResourceType,
			Relationship:  model.OwnerRelationship,
		},
	}
	assert.Equal(t, 1, len(modelRefs))
	assert.Equal(t, expectedRefs, modelRefs)
}

func TestToModelResourceReferences_UnknownRefType(t *testing.T) {
	_, err := toModelResourceReferencesV1([]*apiv1beta1.ResourceReference{
		{
			Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_UNKNOWN_RESOURCE_TYPE, Id: "e1"},
			Relationship: apiv1beta1.Relationship_OWNER,
		},
		{
			Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "j1"},
			Relationship: apiv1beta1.Relationship_CREATOR,
		},
	}, "e1", apiv1beta1.ResourceType_EXPERIMENT)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to convert unsupported v1beta1 API resource type UNKNOWN_RESOURCE_TYPE")
}

func TestToModelResourceReferences_UnknownRelationship(t *testing.T) {
	_, err := toModelResourceReferencesV1([]*apiv1beta1.ResourceReference{
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "e1"}, Relationship: apiv1beta1.Relationship_UNKNOWN_RELATIONSHIP},
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "j1"}, Relationship: apiv1beta1.Relationship_OWNER},
	}, "r1", apiv1beta1.ResourceType_JOB,
	)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "an error in reference relationship")
}

func TestToModelResourceReferences_ImpossibleRelationship(t *testing.T) {
	_, err := toModelResourceReferencesV1([]*apiv1beta1.ResourceReference{
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "e1"}, Relationship: apiv1beta1.Relationship_CREATOR},
		{Key: &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "j1"}, Relationship: apiv1beta1.Relationship_OWNER},
	}, "r1", apiv1beta1.ResourceType_JOB,
	)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Invalid resource-reference relationship")
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

	actualModelRunMetric, err := toModelRunMetric(apiRunMetric, "run-1")
	assert.Nil(t, err)
	expectedModelRunMetric := &model.RunMetric{
		RunUUID:     "run-1",
		Name:        "metric-1",
		NodeID:      "node-1",
		NumberValue: 0.88,
		Format:      "RAW",
	}
	assert.Equal(t, expectedModelRunMetric, actualModelRunMetric)

	// Test Name length overflow
	{
		longName := strings.Repeat("a", 192)
		apiRunMetric := &apiv1beta1.RunMetric{
			Name:   longName,
			NodeId: "node-1",
			Value:  &apiv1beta1.RunMetric_NumberValue{NumberValue: 0.88},
			Format: apiv1beta1.RunMetric_RAW,
		}
		_, err := toModelRunMetric(apiRunMetric, "run-1")
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "RunMetric.Name length cannot exceed 191")
	}

	// Test NodeID length overflow
	{
		longNodeID := strings.Repeat("a", 192)
		apiRunMetric := &apiv1beta1.RunMetric{
			Name:   "metric-1",
			NodeId: longNodeID,
			Value:  &apiv1beta1.RunMetric_NumberValue{NumberValue: 0.88},
			Format: apiv1beta1.RunMetric_RAW,
		}
		_, err := toModelRunMetric(apiRunMetric, "run-1")
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "RunMetric.NodeID length cannot exceed 191")
	}
}

func TestToModelPipelineVersion(t *testing.T) {
	tests := []struct {
		name                    string
		pipeline                interface{}
		expectedPipelineVersion *model.PipelineVersion
		isError                 bool
		errMsg                  string
	}{
		{
			"happy version v1",
			&apiv1beta1.PipelineVersion{
				PackageUrl: &apiv1beta1.Url{
					PipelineUrl: "http://package/11111",
				},
				CodeSourceUrl: "http://repo/11111",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key: &apiv1beta1.ResourceKey{
							Id:   "pipeline1",
							Type: apiv1beta1.ResourceType_PIPELINE,
						},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			&model.PipelineVersion{
				PipelineId:      "pipeline1",
				PipelineSpecURI: "http://package/11111",
				CodeSourceUrl:   "http://repo/11111",
				Status:          model.PipelineVersionCreating,
			},
			false,
			"",
		},
		{
			"missing pipeline url v1",
			&apiv1beta1.PipelineVersion{
				PackageUrl: &apiv1beta1.Url{
					PipelineUrl: "",
				},
				CodeSourceUrl: "http://repo/11111",
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key: &apiv1beta1.ResourceKey{
							Id:   "pipeline1",
							Type: apiv1beta1.ResourceType_PIPELINE,
						},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
			nil,
			true,
			"Invalid input error: Failed to convert v1beta1 API pipeline version to its internal representation due to missing pipeline URL",
		},
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

// Tests toApiPipelineV1
func TestToApiPipelineV1(t *testing.T) {
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
	apiPipeline := toApiPipelineV1(modelPipeline, modelVersion)
	expectedApiPipeline := &apiv1beta1.Pipeline{
		Id:        "pipeline1",
		CreatedAt: timestamppb.New(time.Unix(1, 0)),
		Url:       &apiv1beta1.Url{PipelineUrl: "http://repo/22222"},
		DefaultVersion: &apiv1beta1.PipelineVersion{
			Id:            "pipelineversion1",
			CreatedAt:     timestamppb.New(time.Unix(1, 0)),
			Description:   "desc1",
			CodeSourceUrl: "http://repo/22222",
			PackageUrl:    &apiv1beta1.Url{PipelineUrl: "http://repo/22222"},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
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

// Tests toApiPipelineV1 (error parsing a field)
func TestToApiPipelineV1_ErrorParsingField(t *testing.T) {
	modelPipeline := &model.Pipeline{
		UUID:           "pipeline1",
		CreatedAtInSec: 1,
	}
	modelVersion := &model.PipelineVersion{
		Parameters: "super wrong parameters",
	}
	apiPipeline := toApiPipelineV1(modelPipeline, modelVersion)
	assert.Equal(t, "pipeline1", apiPipeline.Id)
	assert.Contains(t, apiPipeline.Error, "Failed to convert parameters: super wrong parameters")
}

func TestToApiPipelinesV1(t *testing.T) {
	modelPipelines := []*model.Pipeline{
		{
			UUID:           "pipeline1",
			CreatedAtInSec: 1,
		},
		nil,
		{
			UUID:           "pipeline1",
			CreatedAtInSec: 1,
		},
	}
	modelPipelineVersions := []*model.PipelineVersion{
		{
			UUID:           "pipelineversion1",
			CreatedAtInSec: 1,
			Parameters:     "[]",
			PipelineId:     "pipeline1",
			Description:    "desc1",
			CodeSourceUrl:  "http://repo/22222",
		},
		{
			UUID:           "pipelineversion1",
			CreatedAtInSec: 1,
			Parameters:     "[]",
			PipelineId:     "pipeline1",
			Description:    "desc1",
		},
		{
			Parameters: "super wrong parameters",
		},
	}
	apiPipelines := toApiPipelinesV1(modelPipelines, modelPipelineVersions)
	expectedPipelines := []*apiv1beta1.Pipeline{
		{
			Id:        "pipeline1",
			CreatedAt: timestamppb.New(time.Unix(1, 0)),
			Url:       &apiv1beta1.Url{PipelineUrl: "http://repo/22222"},
			DefaultVersion: &apiv1beta1.PipelineVersion{
				Id:            "pipelineversion1",
				CreatedAt:     timestamppb.New(time.Unix(1, 0)),
				Description:   "desc1",
				CodeSourceUrl: "http://repo/22222",
				PackageUrl:    &apiv1beta1.Url{PipelineUrl: "http://repo/22222"},
				ResourceReferences: []*apiv1beta1.ResourceReference{
					{
						Key: &apiv1beta1.ResourceKey{
							Id:   "pipeline1",
							Type: apiv1beta1.ResourceType_PIPELINE,
						},
						Relationship: apiv1beta1.Relationship_OWNER,
					},
				},
			},
		},
		{
			Id:    "",
			Error: "InternalServerError: Failed to convert a model pipeline to v1beta1 API pipeline: Invalid input error: Pipeline cannot be nil",
		},
		{
			Id:    "pipeline1",
			Error: "InternalServerError: Failed to convert a model pipeline to v1beta1 API pipeline: Invalid input error: Failed to convert parameters: super wrong parameters",
		},
	}
	assert.Equal(t, expectedPipelines, apiPipelines)

	modelPipelines2 := make([]*model.Pipeline, 0)
	modelPipelineVersions2 := make([]*model.PipelineVersion, 0)
	apiPipelines2 := toApiPipelinesV1(modelPipelines2, modelPipelineVersions2)
	expectedPipelines2 := make([]*apiv1beta1.Pipeline, 0)
	assert.Equal(t, expectedPipelines2, apiPipelines2)
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

func TestToApiRunDetailV1_RuntimeParams(t *testing.T) {
	modelRun := &model.Run{
		UUID:           "run123",
		K8SName:        "name123",
		StorageState:   model.StorageStateAvailable,
		DisplayName:    "displayName123",
		Namespace:      "ns123",
		RecurringRunId: "job123",
		ExperimentId:   "exp123",
		RunDetails: model.RunDetails{
			CreatedAtInSec:          1,
			ScheduledAtInSec:        1,
			FinishedAtInSec:         1,
			Conditions:              "running",
			WorkflowRuntimeManifest: "workflow123",
		},
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: "manifest",
			RuntimeConfig: model.RuntimeConfig{
				Parameters:   "{\"param2\":\"world\",\"param3\":true,\"param4\":[1,2,3],\"param5\":12,\"param6\":{\"structParam1\":\"hello\",\"structParam2\":32}}",
				PipelineRoot: "model-pipeline-root",
			},
		},
	}
	apiRun := toApiRunDetailV1(modelRun)

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
			CreatedAt:    timestamppb.New(time.Unix(1, 0)),
			ScheduledAt:  timestamppb.New(time.Unix(1, 0)),
			FinishedAt:   timestamppb.New(time.Unix(1, 0)),
			Status:       "Running",
			PipelineSpec: &apiv1beta1.PipelineSpec{
				WorkflowManifest: "manifest",
				RuntimeConfig: &apiv1beta1.PipelineSpec_RuntimeConfig{
					Parameters:   v2RuntimeParams,
					PipelineRoot: "model-pipeline-root",
				},
			},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "exp123"},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns123"},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "job123"},
					Relationship: apiv1beta1.Relationship_CREATOR,
				},
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
	modelRun := &model.Run{
		UUID:         "run123",
		K8SName:      "name123",
		StorageState: model.StorageStateAvailable,
		DisplayName:  "displayName123",
		Namespace:    "ns123",
		RunDetails: model.RunDetails{
			CreatedAtInSec:          1,
			ScheduledAtInSec:        1,
			FinishedAtInSec:         1,
			Conditions:              "running",
			WorkflowRuntimeManifest: "workflow123",
		},
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: "manifest",
			Parameters:           `[{"name":"param2","value":"world"}]`,
		},
		RecurringRunId: "job123",
	}
	apiRun := toApiRunDetailV1(modelRun)
	expectedApiRun := &apiv1beta1.RunDetail{
		Run: &apiv1beta1.Run{
			Id:           "run123",
			Name:         "displayName123",
			StorageState: apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:    timestamppb.New(time.Unix(1, 0)),
			ScheduledAt:  timestamppb.New(time.Unix(1, 0)),
			FinishedAt:   timestamppb.New(time.Unix(1, 0)),
			Status:       "Running",
			PipelineSpec: &apiv1beta1.PipelineSpec{
				WorkflowManifest: "manifest",
				Parameters:       []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
			},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns123"},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "job123"},
					Relationship: apiv1beta1.Relationship_CREATOR,
				},
			},
		},
		PipelineRuntime: &apiv1beta1.PipelineRuntime{
			WorkflowManifest: "workflow123",
		},
	}
	assert.Equal(t, expectedApiRun, apiRun)
}

func TestToApiRunsV1(t *testing.T) {
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
		UUID:         "run1",
		K8SName:      "name1",
		StorageState: model.StorageStateAvailable,
		DisplayName:  "displayName1",
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
		},
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: "manifest",
		},
		RecurringRunId: "job1",
		Metrics:        []*model.RunMetric{metric1, metric2},
	}
	modelRun2 := model.Run{
		UUID:         "run2",
		K8SName:      "name2",
		StorageState: model.StorageStateAvailable,
		DisplayName:  "displayName2",
		Namespace:    "ns2",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "done",
		},
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: "manifest",
		},
		RecurringRunId: "job2",
		Metrics:        []*model.RunMetric{metric2},
	}
	apiRuns := toApiRunsV1([]*model.Run{&modelRun1, &modelRun2})
	expectedApiRun := []*apiv1beta1.Run{
		{
			Id:           "run1",
			Name:         "displayName1",
			StorageState: apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:    timestamppb.New(time.Unix(1, 0)),
			ScheduledAt:  timestamppb.New(time.Unix(1, 0)),
			FinishedAt:   &timestamppb.Timestamp{Seconds: 0, Nanos: 0},
			Status:       "Running",
			PipelineSpec: &apiv1beta1.PipelineSpec{
				WorkflowManifest: "manifest",
			},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns1"},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "job1"},
					Relationship: apiv1beta1.Relationship_CREATOR,
				},
			},
			Metrics: []*apiv1beta1.RunMetric{apiMetric1, apiMetric2},
		},
		{
			Id:           "run2",
			Name:         "displayName2",
			StorageState: apiv1beta1.Run_STORAGESTATE_AVAILABLE,
			CreatedAt:    timestamppb.New(time.Unix(2, 0)),
			ScheduledAt:  timestamppb.New(time.Unix(2, 0)),
			FinishedAt:   &timestamppb.Timestamp{Seconds: 0, Nanos: 0},
			Status:       "Succeeded",
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "ns2"},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "job2"},
					Relationship: apiv1beta1.Relationship_CREATOR,
				},
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
		UUID:              DefaultFakeUUID,
		Namespace:         "",
		PipelineName:      "pipeline/my-pipeline",
		RunId:             NonDefaultFakeUUID,
		MLMDExecutionID:   "1",
		CreatedTimestamp:  1,
		FinishedTimestamp: 2,
		Fingerprint:       "123",
	}
	apiTask := toApiTaskV1(modelTask)
	expectedApiTask := &apiv1beta1.Task{
		Id:              DefaultFakeUUID,
		Namespace:       "",
		PipelineName:    "pipeline/my-pipeline",
		RunId:           NonDefaultFakeUUID,
		MlmdExecutionID: "1",
		CreatedAt:       timestamppb.New(time.Unix(1, 0)),
		FinishedAt:      timestamppb.New(time.Unix(2, 0)),
		Fingerprint:     "123",
	}

	assert.Equal(t, expectedApiTask, apiTask)
}

func TestToApiTasks(t *testing.T) {
	modelTask1 := model.Task{
		UUID:              "123e4567-e89b-12d3-a456-426655440000",
		Namespace:         "ns1",
		PipelineName:      "namespace/ns1/pipeline/my-pipeline-1",
		RunId:             "123e4567-e89b-12d3-a456-426655440001",
		MLMDExecutionID:   "1",
		CreatedTimestamp:  1,
		FinishedTimestamp: 2,
		Fingerprint:       "123",
	}
	modelTask2 := model.Task{
		UUID:              "123e4567-e89b-12d3-a456-426655440002",
		Namespace:         "ns2",
		PipelineName:      "namespace/ns1/pipeline/my-pipeline-2",
		RunId:             "123e4567-e89b-12d3-a456-426655440003",
		MLMDExecutionID:   "2",
		CreatedTimestamp:  3,
		FinishedTimestamp: 4,
		Fingerprint:       "124",
	}

	apiTasks := toApiTasksV1([]*model.Task{&modelTask1, &modelTask2})
	expectedApiTasks := []*apiv1beta1.Task{
		{
			Id:              "123e4567-e89b-12d3-a456-426655440000",
			Namespace:       "ns1",
			PipelineName:    "namespace/ns1/pipeline/my-pipeline-1",
			RunId:           "123e4567-e89b-12d3-a456-426655440001",
			MlmdExecutionID: "1",
			CreatedAt:       timestamppb.New(time.Unix(1, 0)),
			FinishedAt:      timestamppb.New(time.Unix(2, 0)),
			Fingerprint:     "123",
		},
		{
			Id:              "123e4567-e89b-12d3-a456-426655440002",
			Namespace:       "ns2",
			PipelineName:    "namespace/ns1/pipeline/my-pipeline-2",
			RunId:           "123e4567-e89b-12d3-a456-426655440003",
			MlmdExecutionID: "2",
			CreatedAt:       &timestamppb.Timestamp{Seconds: 3, Nanos: 0},
			FinishedAt:      &timestamppb.Timestamp{Seconds: 4, Nanos: 0},
			Fingerprint:     "124",
		},
	}
	assert.Equal(t, expectedApiTasks, apiTasks)
}

func TestCronScheduledJobtoApiJob(t *testing.T) {
	modelJob := model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		K8SName:     "name1",
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
			{
				ResourceUUID: "job1", ResourceType: model.JobResourceType, ReferenceUUID: "experiment1", ReferenceName: "e1",
				ReferenceType: model.ExperimentResourceType, Relationship: model.OwnerRelationship,
			},
		},
	}
	apiJob := toApiJobV1(modelJob.ToV2())
	expectedJob := &apiv1beta1.Job{
		Id:             "job1",
		Name:           "name 1",
		Enabled:        true,
		CreatedAt:      timestamppb.New(time.Unix(1, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(1, 0)),
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(1, 0)),
				Cron:      "1 * *",
			}},
		},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			Parameters:   []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
			PipelineId:   "1",
			PipelineName: "p1",
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "experiment1"},
				Relationship: apiv1beta1.Relationship_OWNER,
			},
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_PIPELINE, Id: "1"},
				Relationship: apiv1beta1.Relationship_CREATOR,
			},
		},
		Status: "STATUS_UNSPECIFIED",
	}
	assert.Equal(t, expectedJob, apiJob)
}

func TestPeriodicScheduledJobtoApiJob(t *testing.T) {
	modelJob := model.Job{
		UUID:        "job1",
		DisplayName: "name 1",
		K8SName:     "name1",
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
	apiJob := toApiJobV1(&modelJob)
	expectedJob := &apiv1beta1.Job{
		Id:             "job1",
		Name:           "name 1",
		Enabled:        true,
		CreatedAt:      timestamppb.New(time.Unix(1, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(1, 0)),
		MaxConcurrency: 1,
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_PeriodicSchedule{PeriodicSchedule: &apiv1beta1.PeriodicSchedule{
				StartTime:      timestamppb.New(time.Unix(1, 0)),
				IntervalSecond: 3,
			}},
		},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			Parameters:   []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
			PipelineId:   "1",
			PipelineName: "p1",
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_PIPELINE, Id: "1"},
				Relationship: apiv1beta1.Relationship_CREATOR,
			},
		},
		Status: "STATUS_UNSPECIFIED",
	}
	assert.Equal(t, expectedJob, apiJob)
}

func TestNonScheduledJobtoApiJob(t *testing.T) {
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
	apiJob := toApiJobV1(&modelJob)
	expectedJob := &apiv1beta1.Job{
		Id:             "job1",
		Name:           "name1",
		Enabled:        true,
		CreatedAt:      timestamppb.New(time.Unix(1, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(1, 0)),
		MaxConcurrency: 1,
		PipelineSpec: &apiv1beta1.PipelineSpec{
			Parameters:   []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}},
			PipelineId:   "1",
			PipelineName: "p1",
		},
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_PIPELINE, Id: "1"},
				Relationship: apiv1beta1.Relationship_CREATOR,
			},
		},
		Status: "STATUS_UNSPECIFIED",
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
	modelJob2 := &model.Job{
		UUID:           "job2",
		DisplayName:    "name1",
		Enabled:        true,
		Trigger:        model.Trigger{},
		MaxConcurrency: 1,
		PipelineSpec: model.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "wrong cong params",
			},
		},
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
	}

	apiJob := toApiJobV1(modelJob)
	assert.Equal(t, "job1", apiJob.Id)
	assert.Contains(t, apiJob.Error, "Pipeline v1 parameters were not parsed correctly")

	apiJob2 := toApiJobV1(modelJob2)
	assert.Equal(t, "job2", apiJob2.Id)
	assert.Contains(t, apiJob2.Error, "Runtime config was not parsed correctly")
}

func TestToApiJob_V2(t *testing.T) {
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
	expectedJob := &apiv1beta1.Job{
		Id:             "job1",
		Name:           "name 1",
		Enabled:        true,
		CreatedAt:      timestamppb.New(time.Unix(2, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(2, 0)),
		MaxConcurrency: 2,
		NoCatchup:      true,
		Status:         "STATUS_UNSPECIFIED",
		ResourceReferences: []*apiv1beta1.ResourceReference{
			{
				Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_PIPELINE, Id: "1"},
				Relationship: apiv1beta1.Relationship_CREATOR,
			},
		},
		Trigger: &apiv1beta1.Trigger{
			Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
				StartTime: timestamppb.New(time.Unix(2, 0)),
				Cron:      "2 * *",
			}},
		},
		PipelineSpec: &apiv1beta1.PipelineSpec{
			PipelineId:   "1",
			PipelineName: "p1",
			RuntimeConfig: &apiv1beta1.PipelineSpec_RuntimeConfig{
				Parameters: map[string]*structpb.Value{
					"param1": {Kind: &structpb.Value_StringValue{StringValue: "world"}},
				},
				PipelineRoot: "job-1-root",
			},
		},
	}
	apiJob := toApiJobV1(modelJob)
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
		K8SName:     "name1",
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
		K8SName:     "name2",
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
	apiJobs := toApiJobsV1([]*model.Job{&modelJob1, &modeljob2})
	expectedJobs := []*apiv1beta1.Job{
		{
			Id:             "job1",
			Name:           "name 1",
			Enabled:        true,
			CreatedAt:      timestamppb.New(time.Unix(1, 0)),
			UpdatedAt:      timestamppb.New(time.Unix(1, 0)),
			MaxConcurrency: 1,
			Status:         "STATUS_UNSPECIFIED",
			Trigger: &apiv1beta1.Trigger{
				Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
					StartTime: timestamppb.New(time.Unix(1, 0)),
					Cron:      "1 * *",
				}},
			},
			PipelineSpec: &apiv1beta1.PipelineSpec{
				Parameters:   []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
				PipelineId:   "1",
				PipelineName: "p1",
			},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_PIPELINE, Id: "1"},
					Relationship: apiv1beta1.Relationship_CREATOR,
				},
			},
		},
		{
			Id:             "job2",
			Name:           "name 2",
			Enabled:        true,
			CreatedAt:      timestamppb.New(time.Unix(2, 0)),
			UpdatedAt:      timestamppb.New(time.Unix(2, 0)),
			MaxConcurrency: 2,
			NoCatchup:      true,
			Status:         "STATUS_UNSPECIFIED",
			Trigger: &apiv1beta1.Trigger{
				Trigger: &apiv1beta1.Trigger_CronSchedule{CronSchedule: &apiv1beta1.CronSchedule{
					StartTime: timestamppb.New(time.Unix(2, 0)),
					Cron:      "2 * *",
				}},
			},
			PipelineSpec: &apiv1beta1.PipelineSpec{
				Parameters:   []*apiv1beta1.Parameter{{Name: "param1", Value: "world"}},
				PipelineId:   "2",
				PipelineName: "p2",
			},
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_PIPELINE, Id: "2"},
					Relationship: apiv1beta1.Relationship_CREATOR,
				},
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

	actualAPIRunMetric := toApiRunMetricV1(modelRunMetric)

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

	actualAPIRunMetric := toApiRunMetricV1(modelRunMetric)

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
		{
			ResourceUUID: "run1", ResourceType: model.RunResourceType, ReferenceUUID: "experiment1",
			ReferenceName: "e1", ReferenceType: model.ExperimentResourceType, Relationship: model.OwnerRelationship,
		},
		{
			ResourceUUID: "run1", ResourceType: model.RunResourceType, ReferenceUUID: "job1",
			ReferenceName: "j1", ReferenceType: model.JobResourceType, Relationship: model.OwnerRelationship,
		},
		{
			ResourceUUID: "run1", ResourceType: model.RunResourceType, ReferenceUUID: "pipelineversion1",
			ReferenceName: "k1", ReferenceType: model.PipelineVersionResourceType, Relationship: model.OwnerRelationship,
		},
	}
	expectedApiResourceReferences := []*apiv1beta1.ResourceReference{
		{
			Key:  &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_EXPERIMENT, Id: "experiment1"},
			Name: "e1", Relationship: apiv1beta1.Relationship_OWNER,
		},
		{
			Key:  &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_JOB, Id: "job1"},
			Name: "j1", Relationship: apiv1beta1.Relationship_OWNER,
		},
		{
			Key:  &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_PIPELINE_VERSION, Id: "pipelineversion1"},
			Name: "k1", Relationship: apiv1beta1.Relationship_OWNER,
		},
	}
	assert.Equal(t, expectedApiResourceReferences, toApiResourceReferencesV1(resourceReferences))
}

func TestToApiExperimentsV1(t *testing.T) {
	exp1 := &model.Experiment{
		UUID:           "exp1",
		CreatedAtInSec: 1,
		Name:           "experiment1",
		Description:    "experiment1 was created using V2 APIV1BETA1",
		StorageState:   "AVAILABLE",
		Namespace:      "default",
	}
	exp2 := &model.Experiment{
		UUID:           "exp2",
		CreatedAtInSec: 2,
		Name:           "experiment2",
		Description:    "experiment2 was created using V2 APIV1BETA1",
		Namespace:      "default",
		StorageState:   "ARCHIVED",
	}
	exp3 := &model.Experiment{
		UUID:           "exp3",
		CreatedAtInSec: 3,
		Name:           "experiment3",
		Description:    "experiment3 was created using V1 APIV1BETA1",
		Namespace:      "default",
		StorageState:   "STORAGESTATE_AVAILABLE",
	}
	exp4 := &model.Experiment{
		UUID:           "exp4",
		CreatedAtInSec: 4,
		Name:           "experiment4",
		Description:    "experiment4 was created using V1 APIV1BETA1",
		Namespace:      "default",
		StorageState:   "STORAGESTATE_ARCHIVED",
	}
	exp5 := &model.Experiment{
		UUID:           "exp5",
		CreatedAtInSec: 1,
		Name:           "experiment5",
		Description:    "experiment5 was created using V2 APIV1BETA1",
		StorageState:   "this is invalid value",
		Namespace:      "default",
	}
	apiExps := toApiExperimentsV1([]*model.Experiment{exp1, exp2, exp3, exp4, nil, exp5})
	expectedApiExps := []*apiv1beta1.Experiment{
		{
			Id:           "exp1",
			Name:         "experiment1",
			Description:  "experiment1 was created using V2 APIV1BETA1",
			CreatedAt:    timestamppb.New(time.Unix(1, 0)),
			StorageState: apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_AVAILABLE"]),
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "default"},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		},
		{
			Id:           "exp2",
			Name:         "experiment2",
			Description:  "experiment2 was created using V2 APIV1BETA1",
			CreatedAt:    timestamppb.New(time.Unix(2, 0)),
			StorageState: apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_ARCHIVED"]),
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "default"},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		},
		{
			Id:           "exp3",
			Name:         "experiment3",
			Description:  "experiment3 was created using V1 APIV1BETA1",
			CreatedAt:    &timestamppb.Timestamp{Seconds: 3, Nanos: 0},
			StorageState: apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_AVAILABLE"]),
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "default"},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		},
		{
			Id:           "exp4",
			Name:         "experiment4",
			Description:  "experiment4 was created using V1 APIV1BETA1",
			CreatedAt:    &timestamppb.Timestamp{Seconds: 4, Nanos: 0},
			StorageState: apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_ARCHIVED"]),
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "default"},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		},
		{},
		{
			Id:           "exp5",
			Name:         "experiment5",
			Description:  "experiment5 was created using V2 APIV1BETA1",
			CreatedAt:    timestamppb.New(time.Unix(1, 0)),
			StorageState: apiv1beta1.Experiment_StorageState(apiv1beta1.Experiment_StorageState_value["STORAGESTATE_UNSPECIFIED"]),
			ResourceReferences: []*apiv1beta1.ResourceReference{
				{
					Key:          &apiv1beta1.ResourceKey{Type: apiv1beta1.ResourceType_NAMESPACE, Id: "default"},
					Relationship: apiv1beta1.Relationship_OWNER,
				},
			},
		},
	}
	assert.Equal(t, expectedApiExps, apiExps)
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

func TestToApiParameters(t *testing.T) {
	expectedApiParameters := []*apiv1beta1.Parameter{{Name: "param2", Value: "world"}}
	modelParameters := `[{"name":"param2","value":"world"}]`
	actualApiParameters := toApiParametersV1(modelParameters)
	assert.Equal(t, expectedApiParameters, actualApiParameters)

	expectedApiParameters = []*apiv1beta1.Parameter{
		{
			Name:  "pipeline-root",
			Value: "gs://my-bucket/tfx_taxi_simple/{{workflow.uid}}",
		},
		{
			Name:  "version",
			Value: "2",
		},
	}
	modelParameters = `[{"name":"pipeline-root","value":"gs://my-bucket/tfx_taxi_simple/{{workflow.uid}}"},{"name":"version","value":"2"}]`
	actualApiParameters = toApiParametersV1(modelParameters)
	assert.Equal(t, expectedApiParameters, actualApiParameters)

}

func TestToMapProtoStructParameters(t *testing.T) {
	expectedApiParameters := map[string]*structpb.Value{
		"param2": structpb.NewStringValue("world"),
	}
	modelParameters := `[{"name":"param2","value":"world"}]`
	actualApiParameters := toMapProtoStructParameters(modelParameters)
	assert.Equal(t, expectedApiParameters, actualApiParameters)

	expectedApiParameters = map[string]*structpb.Value{
		"pipeline-root": structpb.NewStringValue("gs://my-bucket/tfx_taxi_simple/{{workflow.uid}}"),
		"version":       structpb.NewStringValue("2"),
	}
	modelParameters = `[{"name":"pipeline-root","value":"gs://my-bucket/tfx_taxi_simple/{{workflow.uid}}"},{"name":"version","value":"2"}]`
	actualApiParameters = toMapProtoStructParameters(modelParameters)
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
	actualRuntimeConfig := toApiRuntimeConfigV1(modelRuntimeConfig)
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
		{
			"Invalid run type",
			&apiv1beta1.Run{},
			"",
			"",
			true,
			"Error using RuntimeState with *go_client.Run",
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
				assert.Equal(t, tt.wantV1, got.ToV1())
				assert.Equal(t, tt.wantV2, got.ToV2())
				assert.Equal(t, string(tt.wantV2), got.ToString())
			}
		})
	}
}

func Test_toApiRuntimeStateV1(t *testing.T) {

	tests := []struct {
		name       string
		modelState model.RuntimeState
		want       string
	}{
		{
			"v1 Error",
			model.RuntimeStateErrorV1,
			"Failed",
		},
		{
			"v1 NO_STATUS",
			model.RuntimeState(model.LegacyStateNoStatus),
			"Unknown",
		},
		{
			"v1 succeeded",
			model.RuntimeStateSucceededV1,
			"Succeeded",
		},
		{
			"v2 succeeded",
			model.RuntimeStateSucceeded,
			"Succeeded",
		},
		{
			"v2 cancelling",
			model.RuntimeStateCancelling,
			"Terminating",
		},
		{
			"v2 paused",
			model.RuntimeStatePaused,
			"Pending",
		},
		{
			"v2 Unspecified",
			model.RuntimeStateUnspecified,
			"Unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toApiRuntimeStateV1(&tt.modelState); got != tt.want {
				t.Errorf("toApiRuntimeStateV1() = %v, want %v", tt.want, got)
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
			"v2 cancelling",
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

func Test_toModelTask(t *testing.T) {
	tests := []struct {
		name    string
		apiTask interface{}
		want    *model.Task
		wantErr bool
		errMsg  string
	}{
		{
			"V1 full spec",
			&apiv1beta1.Task{
				Id:              "1",
				Namespace:       "ns1",
				PipelineName:    "namespaces/ns1/pipelines/p1",
				RunId:           "2",
				MlmdExecutionID: "3",
				CreatedAt:       &timestamppb.Timestamp{Seconds: 4},
				FinishedAt:      &timestamppb.Timestamp{Seconds: 5},
				Fingerprint:     "6",
			},
			&model.Task{
				UUID:              "1",
				Namespace:         "ns1",
				PipelineName:      "namespaces/ns1/pipelines/p1",
				RunId:             "2",
				MLMDExecutionID:   "3",
				CreatedTimestamp:  4,
				StartedTimestamp:  4,
				FinishedTimestamp: 5,
				Fingerprint:       "6",
				Name:              "",
				ParentTaskId:      "",
				State:             model.RuntimeStateUnspecified,
				StateHistory:      nil,
				MLMDInputs:        "",
				MLMDOutputs:       "",
				ChildrenPods:      nil,
			},
			false,
			"",
		},
		{
			"V2 full spec",
			&apiv2beta1.PipelineTaskDetail{
				RunId:       "2",
				TaskId:      "1",
				DisplayName: "task",
				CreateTime:  &timestamppb.Timestamp{Seconds: 4},
				StartTime:   &timestamppb.Timestamp{Seconds: 5},
				EndTime:     &timestamppb.Timestamp{Seconds: 6},
				State:       apiv2beta1.RuntimeState_CANCELING,
				ExecutionId: 7,
				Inputs: map[string]*apiv2beta1.ArtifactList{
					"a1": {
						ArtifactIds: []int64{1, 2, 3},
					},
				},
				Outputs: map[string]*apiv2beta1.ArtifactList{
					"b2": {
						ArtifactIds: []int64{4, 5, 6},
					},
				},
				ParentTaskId: "8",
				StateHistory: []*apiv2beta1.RuntimeStatus{
					{
						UpdateTime: &timestamppb.Timestamp{Seconds: 9},
						State:      apiv2beta1.RuntimeState_PAUSED,
					},
				},
				ChildTasks: []*apiv2beta1.PipelineTaskDetail_ChildTask{
					{
						ChildTask: &apiv2beta1.PipelineTaskDetail_ChildTask_PodName{PodName: "9"},
					},
					{
						ChildTask: &apiv2beta1.PipelineTaskDetail_ChildTask_PodName{PodName: "10"},
					},
				},
			},
			&model.Task{
				UUID:              "1",
				Namespace:         "",
				PipelineName:      "",
				RunId:             "2",
				MLMDExecutionID:   "7",
				CreatedTimestamp:  4,
				StartedTimestamp:  5,
				FinishedTimestamp: 6,
				Fingerprint:       "",
				Name:              "task",
				ParentTaskId:      "8",
				State:             model.RuntimeStateCancelling,
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 9,
						State:           model.RuntimeStatePaused,
					},
				},
				MLMDInputs:   `{"a1":{"artifact_ids":[1,2,3]}}`,
				MLMDOutputs:  `{"b2":{"artifact_ids":[4,5,6]}}`,
				ChildrenPods: []string{"9", "10"},
			},
			false,
			"",
		},
		{
			"argo node status",
			util.NodeStatus{
				ID:          "1",
				DisplayName: "node_1",
				State:       "Pending",
				Children:    []string{"node3", "node4"},
				StartTime:   4,
				CreateTime:  4,
				FinishTime:  5,
			},
			&model.Task{
				PodName:           "1",
				CreatedTimestamp:  4,
				StartedTimestamp:  4,
				FinishedTimestamp: 5,
				Name:              "node_1",
				State:             model.RuntimeStatePending,
				ChildrenPods:      []string{"node3", "node4"},
			},
			false,
			"",
		},
		{
			"invalid type",
			apiv2beta1.Run{},
			nil,
			true,
			"UnknownApiVersionError: Error using Task with go_client.Run",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toModelTask(tt.apiTask)
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

func Test_toModelTasks_v2(t *testing.T) {
	argV2 := []*apiv2beta1.PipelineTaskDetail{
		{
			RunId:       "2",
			TaskId:      "1",
			DisplayName: "task",
			CreateTime:  &timestamppb.Timestamp{Seconds: 4},
			StartTime:   &timestamppb.Timestamp{Seconds: 5},
			EndTime:     &timestamppb.Timestamp{Seconds: 6},
			State:       apiv2beta1.RuntimeState_FAILED,
			ExecutionId: 7,
			Inputs: map[string]*apiv2beta1.ArtifactList{
				"a1": {
					ArtifactIds: []int64{1, 2, 3},
				},
			},
			Outputs: map[string]*apiv2beta1.ArtifactList{
				"b2": {
					ArtifactIds: []int64{4, 5, 6},
				},
			},
			ParentTaskId: "8",
			StateHistory: []*apiv2beta1.RuntimeStatus{
				{
					UpdateTime: &timestamppb.Timestamp{Seconds: 9},
					State:      apiv2beta1.RuntimeState_FAILED,
					Error:      util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid")),
				},
			},
			ChildTasks: []*apiv2beta1.PipelineTaskDetail_ChildTask{
				{
					ChildTask: &apiv2beta1.PipelineTaskDetail_ChildTask_PodName{PodName: "9"},
				},
				{
					ChildTask: &apiv2beta1.PipelineTaskDetail_ChildTask_PodName{PodName: "10"},
				},
			},
		},
	}
	expectedV2 := []*model.Task{
		{
			UUID:              "1",
			Namespace:         "",
			PipelineName:      "",
			RunId:             "2",
			MLMDExecutionID:   "7",
			CreatedTimestamp:  4,
			StartedTimestamp:  5,
			FinishedTimestamp: 6,
			Fingerprint:       "",
			Name:              "task",
			ParentTaskId:      "8",
			State:             model.RuntimeStateFailed,
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 9,
					State:           model.RuntimeStateFailed,
					Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid"))),
				},
			},
			MLMDInputs:   `{"a1":{"artifact_ids":[1,2,3]}}`,
			MLMDOutputs:  `{"b2":{"artifact_ids":[4,5,6]}}`,
			ChildrenPods: []string{"9", "10"},
		},
	}
	gotV2, err := toModelTasks(argV2)
	assert.Nil(t, err)
	assert.Equal(t, expectedV2, gotV2)
}

func Test_toModelTasks_wf(t *testing.T) {
	expectedWf := []*model.Task{
		{
			PodName:           "run1-file-passing-pipelines-node0",
			Namespace:         "kubeflow",
			RunId:             "run1_uid_true",
			CreatedTimestamp:  -62135596800,
			StartedTimestamp:  1675734919,
			FinishedTimestamp: 1675735118,
			Name:              "boudary_exec_id",
			State:             model.RuntimeStateSucceeded,
			ChildrenPods:      []string{"boudary_exec_id-node1"},
		},
		{
			PodName:           "run1-print-text-node1",
			Namespace:         "kubeflow",
			RunId:             "run1_uid_true",
			CreatedTimestamp:  -62135596800,
			StartedTimestamp:  1675735015,
			FinishedTimestamp: 1675735041,
			Name:              "print-text",
			State:             model.RuntimeStateSucceeded,
		},
	}
	argWf, err := util.NewWorkflowFromBytes([]byte(`{  "kind": "Workflow",  "apiVersion": "argoproj.io/v1alpha1",  "metadata": {    "name": "run1",    "namespace": "kubeflow",    "uid": "run1_uid",	"labels": {	  "pipeline/runid": "run1_uid_true"	 }  },  "status": {    "phase": "Succeeded",    "startedAt": "2023-02-07T01:55:19Z",    "finishedAt": "2023-02-07T01:58:38Z",    "progress": "9/9",    "nodes": {      "boudary_exec_id-node0": {        "id": "boudary_exec_id-node0",        "name": "boudary_exec_id",        "displayName": "boudary_exec_id",        "type": "DAG",        "templateName": "file-passing-pipelines",        "templateScope": "local/boudary_exec_id",        "phase": "Succeeded",        "startedAt": "2023-02-07T01:55:19Z",        "finishedAt": "2023-02-07T01:58:38Z",        "progress": "9/9",        "resourcesDuration": {"cpu": 53,"memory": 19},        "children": ["boudary_exec_id-node1"],        "outboundNodes": ["boudary_exec_id-node1"]      },      "boudary_exec_id-node1": {        "id": "boudary_exec_id-node1",        "name": "boudary_exec_id.print-text",        "displayName": "print-text",        "type": "Pod",        "templateName": "print-text",        "templateScope": "local/boudary_exec_id",        "phase": "Succeeded",        "boundaryID": "boudary_exec_id",        "startedAt": "2023-02-07T01:56:55Z",        "finishedAt": "2023-02-07T01:57:21Z",        "progress": "1/1",        "resourcesDuration": {"cpu": 15,"memory": 7},        "inputs": {"artifacts": [{"name": "repeat-line-output_text",              "path": "/tmp/inputs/text/data",              "s3": {"key": "art1.tgz"}}]},        "outputs": {"artifacts": [{"name": "main-logs",              "s3": {"key": "art1.log"}}],          "exitCode": "0"},        "hostNodeName": "gke-kfp-node1"      }    }  }}`))
	assert.Nil(t, err)

	gotWf, err := toModelTasks(argWf)
	assert.Nil(t, err)
	if !cmp.Equal(expectedWf, gotWf) {
		t.Errorf("toModelTasks() diff: %v", cmp.Diff(gotWf, expectedWf))
	}
}

func Test_toApiTaskV1(t *testing.T) {
	tests := []struct {
		name string
		args *model.Task
		want *apiv1beta1.Task
	}{
		{
			"v1 spec",
			&model.Task{
				UUID:              "1",
				Namespace:         "ns1",
				PipelineName:      "namespaces/ns1/pipelines/p1",
				RunId:             "2",
				MLMDExecutionID:   "3",
				CreatedTimestamp:  4,
				StartedTimestamp:  4,
				FinishedTimestamp: 5,
				Fingerprint:       "6",
				Name:              "",
				ParentTaskId:      "",
				State:             model.RuntimeStateUnspecified,
				StateHistory:      nil,
				MLMDInputs:        "",
				MLMDOutputs:       "",
				ChildrenPods:      nil,
			},
			&apiv1beta1.Task{
				Id:              "1",
				Namespace:       "ns1",
				PipelineName:    "namespaces/ns1/pipelines/p1",
				RunId:           "2",
				MlmdExecutionID: "3",
				CreatedAt:       &timestamppb.Timestamp{Seconds: 4},
				FinishedAt:      &timestamppb.Timestamp{Seconds: 5},
				Fingerprint:     "6",
			},
		},
		{
			"v2 spec",
			&model.Task{
				UUID:              "1",
				Namespace:         "ns1",
				PipelineName:      "namespaces/ns1/pipelines/p1",
				RunId:             "2",
				MLMDExecutionID:   "7",
				CreatedTimestamp:  4,
				StartedTimestamp:  5,
				FinishedTimestamp: 6,
				Fingerprint:       "fp",
				Name:              "task",
				ParentTaskId:      "8",
				State:             model.RuntimeStateCancelling,
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 9,
						State:           model.RuntimeStatePaused,
						Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Sample error2"))),
					},
				},
				MLMDInputs:   `{"a1":{"artifact_ids":[1,2,3]}}`,
				MLMDOutputs:  `{"b2":{"artifact_ids":[4,5,6]}}`,
				ChildrenPods: []string{"9", "10"},
			},
			&apiv1beta1.Task{
				Id:              "1",
				Namespace:       "ns1",
				PipelineName:    "namespaces/ns1/pipelines/p1",
				RunId:           "2",
				MlmdExecutionID: "7",
				CreatedAt:       &timestamppb.Timestamp{Seconds: 4},
				FinishedAt:      &timestamppb.Timestamp{Seconds: 6},
				Fingerprint:     "fp",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toApiTaskV1(tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_toApiTasksV1(t *testing.T) {
	arg := []*model.Task{
		{
			UUID:              "1",
			Namespace:         "ns1",
			PipelineName:      "namespaces/ns1/pipelines/p1",
			RunId:             "2",
			MLMDExecutionID:   "3",
			CreatedTimestamp:  4,
			StartedTimestamp:  4,
			FinishedTimestamp: 5,
			Fingerprint:       "6",
			Name:              "",
			ParentTaskId:      "",
			State:             model.RuntimeStateUnspecified,
			StateHistory:      nil,
			MLMDInputs:        "",
			MLMDOutputs:       "",
			ChildrenPods:      nil,
		},
		{
			UUID:              "1",
			Namespace:         "ns1",
			PipelineName:      "namespaces/ns1/pipelines/p1",
			RunId:             "2",
			MLMDExecutionID:   "7",
			CreatedTimestamp:  4,
			StartedTimestamp:  5,
			FinishedTimestamp: 6,
			Fingerprint:       "fp",
			Name:              "task",
			ParentTaskId:      "8",
			State:             model.RuntimeStateCancelling,
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 9,
					State:           model.RuntimeStatePaused,
					Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Sample error2"))),
				},
			},
			MLMDInputs:   `{"a1":{"artifact_ids":[1,2,3]}}`,
			MLMDOutputs:  `{"b2":{"artifact_ids":[4,5,6]}}`,
			ChildrenPods: []string{"9", "10"},
		},
	}
	expected := []*apiv1beta1.Task{
		{
			Id:              "1",
			Namespace:       "ns1",
			PipelineName:    "namespaces/ns1/pipelines/p1",
			RunId:           "2",
			MlmdExecutionID: "3",
			CreatedAt:       &timestamppb.Timestamp{Seconds: 4},
			FinishedAt:      &timestamppb.Timestamp{Seconds: 5},
			Fingerprint:     "6",
		},
		{
			Id:              "1",
			Namespace:       "ns1",
			PipelineName:    "namespaces/ns1/pipelines/p1",
			RunId:           "2",
			MlmdExecutionID: "7",
			CreatedAt:       &timestamppb.Timestamp{Seconds: 4},
			FinishedAt:      &timestamppb.Timestamp{Seconds: 6},
			Fingerprint:     "fp",
		},
	}
	got := toApiTasksV1(arg)
	assert.Equal(t, expected, got)
}

func Test_toApiPipelineTaskDetail(t *testing.T) {
	tests := []struct {
		name    string
		args    *model.Task
		want    *apiv2beta1.PipelineTaskDetail
		wantErr bool
		errMsg  string
	}{
		{
			"v1 spec",
			&model.Task{
				UUID:              "1",
				Namespace:         "ns1",
				PipelineName:      "namespaces/ns1/pipelines/p1",
				RunId:             "2",
				MLMDExecutionID:   "3",
				CreatedTimestamp:  4,
				StartedTimestamp:  4,
				FinishedTimestamp: 5,
				Fingerprint:       "6",
				State:             model.RuntimeStateUnspecified,
			},
			&apiv2beta1.PipelineTaskDetail{
				RunId:       "2",
				TaskId:      "1",
				DisplayName: "",
				CreateTime:  &timestamppb.Timestamp{Seconds: 4},
				StartTime:   &timestamppb.Timestamp{Seconds: 4},
				EndTime:     &timestamppb.Timestamp{Seconds: 5},
				State:       apiv2beta1.RuntimeState_RUNTIME_STATE_UNSPECIFIED,
				ExecutionId: 3,
			},
			false,
			"",
		},
		{
			"v2 spec",
			&model.Task{
				UUID:              "1",
				Namespace:         "ns1",
				PipelineName:      "namespaces/ns1/pipelines/p1",
				RunId:             "2",
				MLMDExecutionID:   "7",
				CreatedTimestamp:  4,
				StartedTimestamp:  5,
				FinishedTimestamp: 6,
				Fingerprint:       "fp",
				Name:              "task",
				ParentTaskId:      "8",
				State:             model.RuntimeStateCancelling,
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 9,
						State:           model.RuntimeStatePaused,
						Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Sample error2"))),
					},
				},
				MLMDInputs:   `{"a1":{"artifact_ids":[1,2,3]}}`,
				MLMDOutputs:  `{"b2":{"artifact_ids":[4,5,6]}}`,
				ChildrenPods: []string{"9", "10"},
			},
			&apiv2beta1.PipelineTaskDetail{
				RunId:       "2",
				TaskId:      "1",
				DisplayName: "task",
				CreateTime:  &timestamppb.Timestamp{Seconds: 4},
				StartTime:   &timestamppb.Timestamp{Seconds: 5},
				EndTime:     &timestamppb.Timestamp{Seconds: 6},
				State:       apiv2beta1.RuntimeState_CANCELING,
				ExecutionId: 7,
				Inputs: map[string]*apiv2beta1.ArtifactList{
					"a1": {
						ArtifactIds: []int64{1, 2, 3},
					},
				},
				Outputs: map[string]*apiv2beta1.ArtifactList{
					"b2": {
						ArtifactIds: []int64{4, 5, 6},
					},
				},
				ParentTaskId: "8",
				StateHistory: []*apiv2beta1.RuntimeStatus{
					{
						UpdateTime: &timestamppb.Timestamp{Seconds: 9},
						State:      apiv2beta1.RuntimeState_PAUSED,
						Error:      util.ToRpcStatus(util.NewInvalidInputError("Sample error2")),
					},
				},
				ChildTasks: []*apiv2beta1.PipelineTaskDetail_ChildTask{
					{
						ChildTask: &apiv2beta1.PipelineTaskDetail_ChildTask_PodName{PodName: "9"},
					},
					{
						ChildTask: &apiv2beta1.PipelineTaskDetail_ChildTask_PodName{PodName: "10"},
					},
				},
			},
			false,
			"",
		},
		{
			"v2 wrong inputs",
			&model.Task{
				UUID:              "1",
				Namespace:         "ns1",
				PipelineName:      "namespaces/ns1/pipelines/p1",
				RunId:             "2",
				MLMDExecutionID:   "7",
				CreatedTimestamp:  4,
				StartedTimestamp:  5,
				FinishedTimestamp: 6,
				Fingerprint:       "fp",
				Name:              "task",
				ParentTaskId:      "8",
				State:             model.RuntimeStateCancelling,
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 9,
						State:           model.RuntimeStatePaused,
						Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Sample error2"))),
					},
				},
				MLMDInputs:   `{"a1":{"artifact_ids":[1,2,3]}`,
				MLMDOutputs:  `{"b2":{"artifact_ids":[4,5,6]}}`,
				ChildrenPods: []string{"9", "10"},
			},
			&apiv2beta1.PipelineTaskDetail{
				RunId:  "2",
				TaskId: "1",
			},
			true,
			"Failed to convert task's internal representation to its API counterpart due to error parsing inputs",
		},
		{
			"v2 wrong outputs",
			&model.Task{
				UUID:              "1",
				Namespace:         "ns1",
				PipelineName:      "namespaces/ns1/pipelines/p1",
				RunId:             "2",
				MLMDExecutionID:   "7",
				CreatedTimestamp:  4,
				StartedTimestamp:  5,
				FinishedTimestamp: 6,
				Fingerprint:       "fp",
				Name:              "task",
				ParentTaskId:      "8",
				State:             model.RuntimeStateCancelling,
				StateHistory: []*model.RuntimeStatus{
					{
						UpdateTimeInSec: 9,
						State:           model.RuntimeStatePaused,
						Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Sample error2"))),
					},
				},
				MLMDInputs:   `{"a1":{"artifact_ids":[1,2,3]}}`,
				MLMDOutputs:  `{"b2":{"artifact_ids":[4,5,6]}`,
				ChildrenPods: []string{"9", "10"},
			},
			&apiv2beta1.PipelineTaskDetail{
				RunId:  "2",
				TaskId: "1",
			},
			true,
			"Failed to convert task's internal representation to its API counterpart due to error parsing outputs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toApiPipelineTaskDetail(tt.args)
			if tt.wantErr {
				assert.Equal(t, tt.want.GetRunId(), got.GetRunId())
				assert.Equal(t, tt.want.GetTaskId(), got.GetTaskId())
				assert.Contains(t, got.Error.Message, tt.errMsg)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_toApiPipelineTaskDetails(t *testing.T) {
	args := []*model.Task{
		{
			UUID:              "1",
			Namespace:         "ns1",
			PipelineName:      "namespaces/ns1/pipelines/p1",
			RunId:             "2",
			MLMDExecutionID:   "3",
			CreatedTimestamp:  4,
			StartedTimestamp:  4,
			FinishedTimestamp: 5,
			Fingerprint:       "6",
			State:             model.RuntimeStateUnspecified,
		},
		{
			UUID:              "1",
			Namespace:         "ns1",
			PipelineName:      "namespaces/ns1/pipelines/p1",
			RunId:             "2",
			MLMDExecutionID:   "7",
			CreatedTimestamp:  4,
			StartedTimestamp:  5,
			FinishedTimestamp: 6,
			Fingerprint:       "fp",
			Name:              "task",
			ParentTaskId:      "8",
			State:             model.RuntimeStateCancelling,
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 9,
					State:           model.RuntimeStatePaused,
					Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Sample error2"))),
				},
			},
			MLMDInputs:   `{"a1":{"artifact_ids":[1,2,3]}}`,
			MLMDOutputs:  `{"b2":{"artifact_ids":[4,5,6]}}`,
			ChildrenPods: []string{"9", "10"},
		},
	}
	expected := []*apiv2beta1.PipelineTaskDetail{
		{
			RunId:       "2",
			TaskId:      "1",
			DisplayName: "",
			CreateTime:  &timestamppb.Timestamp{Seconds: 4},
			StartTime:   &timestamppb.Timestamp{Seconds: 4},
			EndTime:     &timestamppb.Timestamp{Seconds: 5},
			State:       apiv2beta1.RuntimeState_RUNTIME_STATE_UNSPECIFIED,
			ExecutionId: 3,
		},
		{
			RunId:       "2",
			TaskId:      "1",
			DisplayName: "task",
			CreateTime:  &timestamppb.Timestamp{Seconds: 4},
			StartTime:   &timestamppb.Timestamp{Seconds: 5},
			EndTime:     &timestamppb.Timestamp{Seconds: 6},
			State:       apiv2beta1.RuntimeState_CANCELING,
			ExecutionId: 7,
			Inputs: map[string]*apiv2beta1.ArtifactList{
				"a1": {
					ArtifactIds: []int64{1, 2, 3},
				},
			},
			Outputs: map[string]*apiv2beta1.ArtifactList{
				"b2": {
					ArtifactIds: []int64{4, 5, 6},
				},
			},
			ParentTaskId: "8",
			StateHistory: []*apiv2beta1.RuntimeStatus{
				{
					UpdateTime: &timestamppb.Timestamp{Seconds: 9},
					State:      apiv2beta1.RuntimeState_PAUSED,
					Error:      util.ToRpcStatus(util.NewInvalidInputError("Sample error2")),
				},
			},
			ChildTasks: []*apiv2beta1.PipelineTaskDetail_ChildTask{
				{
					ChildTask: &apiv2beta1.PipelineTaskDetail_ChildTask_PodName{PodName: "9"},
				},
				{
					ChildTask: &apiv2beta1.PipelineTaskDetail_ChildTask_PodName{PodName: "10"},
				},
			},
		},
	}
	got := toApiPipelineTaskDetails(args)
	assert.Equal(t, expected, got)

	args2 := []*model.Task{
		{
			UUID:              "1",
			Namespace:         "ns1",
			PipelineName:      "namespaces/ns1/pipelines/p1",
			RunId:             "2",
			MLMDExecutionID:   "7",
			CreatedTimestamp:  4,
			StartedTimestamp:  5,
			FinishedTimestamp: 6,
			Fingerprint:       "fp",
			Name:              "task",
			ParentTaskId:      "8",
			State:             model.RuntimeStateCancelling,
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 9,
					State:           model.RuntimeStatePaused,
					Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Sample error2"))),
				},
			},
			MLMDInputs:   `{"a1":{"artifact_ids":[1,2,3]}`,
			MLMDOutputs:  `{"b2":{"artifact_ids":[4,5,6]}}`,
			ChildrenPods: []string{"9", "10"},
		},
		{
			UUID:              "1",
			Namespace:         "ns1",
			PipelineName:      "namespaces/ns1/pipelines/p1",
			RunId:             "2",
			MLMDExecutionID:   "7",
			CreatedTimestamp:  4,
			StartedTimestamp:  5,
			FinishedTimestamp: 6,
			Fingerprint:       "fp",
			Name:              "task",
			ParentTaskId:      "8",
			State:             model.RuntimeStateCancelling,
			StateHistory: []*model.RuntimeStatus{
				{
					UpdateTimeInSec: 9,
					State:           model.RuntimeStatePaused,
					Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Sample error2"))),
				},
			},
			MLMDInputs:   `{"a1":{"artifact_ids":[1,2,3]}}`,
			MLMDOutputs:  `{"b2":{"artifact_ids":[4,5,6]}`,
			ChildrenPods: []string{"9", "10"},
		},
	}
	got2 := toApiPipelineTaskDetails(args2)
	assert.Contains(t, got2[0].Error.Message, "Failed to convert task's internal representation to its API counterpart due to error parsing inputs")
	assert.Contains(t, got2[1].Error.Message, "Failed to convert task's internal representation to its API counterpart due to error parsing outputs")
	expected2 := &apiv2beta1.PipelineTaskDetail{
		RunId:  "2",
		TaskId: "1",
	}
	expected2.Error = got2[0].GetError()
	assert.Equal(t, expected2, got2[0])
	expected2.Error = got2[1].GetError()
	assert.Equal(t, expected2, got2[1])
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
					TaskDetails: []*apiv2beta1.PipelineTaskDetail{
						{
							RunId:          "run1",
							TaskId:         "task1",
							DisplayName:    "this is task",
							CreateTime:     timestamppb.New(time.Unix(11, 0)),
							StartTime:      timestamppb.New(time.Unix(12, 0)),
							EndTime:        timestamppb.New(time.Unix(13, 0)),
							ExecutorDetail: nil,
							State:          apiv2beta1.RuntimeState_FAILED,
							ExecutionId:    14,
							Inputs: map[string]*apiv2beta1.ArtifactList{
								"a1": {ArtifactIds: []int64{1, 2, 3}},
							},
							Outputs: map[string]*apiv2beta1.ArtifactList{
								"b2": {ArtifactIds: []int64{4, 5, 6}},
							},
							StateHistory: []*apiv2beta1.RuntimeStatus{
								{
									UpdateTime: &timestamppb.Timestamp{Seconds: 15},
									State:      apiv2beta1.RuntimeState_FAILED,
									Error:      util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid")),
								},
							},
							ChildTasks: []*apiv2beta1.PipelineTaskDetail_ChildTask{
								{
									ChildTask: &apiv2beta1.PipelineTaskDetail_ChildTask_PodName{PodName: "task2"},
								},
							},
						},
						{
							RunId:          "run1",
							TaskId:         "task2",
							DisplayName:    "this is task 2",
							CreateTime:     timestamppb.New(time.Unix(11, 0)),
							StartTime:      timestamppb.New(time.Unix(12, 0)),
							EndTime:        timestamppb.New(time.Unix(13, 0)),
							ExecutorDetail: nil,
							State:          apiv2beta1.RuntimeState_CANCELED,
							ExecutionId:    14,
							Inputs: map[string]*apiv2beta1.ArtifactList{
								"a1": {ArtifactIds: []int64{1, 2, 3}},
							},
							Outputs: map[string]*apiv2beta1.ArtifactList{
								"b2": {ArtifactIds: []int64{4, 5, 6}},
							},
							ParentTaskId: "task1",
							StateHistory: []*apiv2beta1.RuntimeStatus{
								{
									UpdateTime: &timestamppb.Timestamp{Seconds: 15},
									State:      apiv2beta1.RuntimeState_CANCELED,
								},
							},
						},
					},
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
					TaskDetails: []*model.Task{
						{
							UUID:              "task1",
							Namespace:         "",
							PipelineName:      "",
							RunId:             "run1",
							MLMDExecutionID:   "14",
							CreatedTimestamp:  11,
							StartedTimestamp:  12,
							FinishedTimestamp: 13,
							Fingerprint:       "",
							Name:              "this is task",
							State:             model.RuntimeStateFailed,
							MLMDInputs:        `{"a1":{"artifact_ids":[1,2,3]}}`,
							MLMDOutputs:       `{"b2":{"artifact_ids":[4,5,6]}}`,
							StateHistory: []*model.RuntimeStatus{
								{
									UpdateTimeInSec: 15,
									State:           model.RuntimeStateFailed,
									Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid"))),
								},
							},
							ChildrenPods: []string{"task2"},
						},
						{
							UUID:              "task2",
							Namespace:         "",
							PipelineName:      "",
							RunId:             "run1",
							MLMDExecutionID:   "14",
							CreatedTimestamp:  11,
							StartedTimestamp:  12,
							FinishedTimestamp: 13,
							Fingerprint:       "",
							Name:              "this is task 2",
							ParentTaskId:      "task1",
							State:             model.RuntimeStateCanceled,
							MLMDInputs:        `{"a1":{"artifact_ids":[1,2,3]}}`,
							MLMDOutputs:       `{"b2":{"artifact_ids":[4,5,6]}}`,
							StateHistory: []*model.RuntimeStatus{
								{
									UpdateTimeInSec: 15,
									State:           model.RuntimeStateCanceled,
								},
							},
						},
					},
				},
				ResourceReferences: nil,
				Metrics:            nil,
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
					TaskDetails: []*apiv2beta1.PipelineTaskDetail{
						{
							RunId:          "run2",
							TaskId:         "task1",
							DisplayName:    "this is task",
							CreateTime:     timestamppb.New(time.Unix(11, 0)),
							StartTime:      timestamppb.New(time.Unix(12, 0)),
							EndTime:        timestamppb.New(time.Unix(13, 0)),
							ExecutorDetail: nil,
							State:          apiv2beta1.RuntimeState_RUNNING,
							ExecutionId:    14,
							Inputs: map[string]*apiv2beta1.ArtifactList{
								"a1": {ArtifactIds: []int64{1, 2, 3}},
							},
							Outputs: map[string]*apiv2beta1.ArtifactList{
								"b2": {ArtifactIds: []int64{4, 5, 6}},
							},
							StateHistory: []*apiv2beta1.RuntimeStatus{
								{
									UpdateTime: &timestamppb.Timestamp{Seconds: 15},
									State:      apiv2beta1.RuntimeState_RUNNING,
								},
							},
						},
					},
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
					TaskDetails: []*model.Task{
						{
							UUID:              "task1",
							Namespace:         "",
							PipelineName:      "",
							RunId:             "run2",
							MLMDExecutionID:   "14",
							CreatedTimestamp:  11,
							StartedTimestamp:  12,
							FinishedTimestamp: 13,
							Fingerprint:       "",
							Name:              "this is task",
							State:             model.RuntimeStateRunning,
							MLMDInputs:        `{"a1":{"artifact_ids":[1,2,3]}}`,
							MLMDOutputs:       `{"b2":{"artifact_ids":[4,5,6]}}`,
							StateHistory: []*model.RuntimeStatus{
								{
									UpdateTimeInSec: 15,
									State:           model.RuntimeStateRunning,
								},
							},
						},
					},
				},
				ResourceReferences: nil,
				Metrics:            nil,
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
					TaskDetails: []*apiv2beta1.PipelineTaskDetail{
						{
							RunId:          "run1",
							TaskId:         "task1",
							DisplayName:    "this is task",
							CreateTime:     timestamppb.New(time.Unix(11, 0)),
							StartTime:      timestamppb.New(time.Unix(12, 0)),
							EndTime:        timestamppb.New(time.Unix(13, 0)),
							ExecutorDetail: nil,
							State:          apiv2beta1.RuntimeState_FAILED,
							ExecutionId:    14,
							Inputs: map[string]*apiv2beta1.ArtifactList{
								"a1": {ArtifactIds: []int64{1, 2, 3}},
							},
							Outputs: map[string]*apiv2beta1.ArtifactList{
								"b2": {ArtifactIds: []int64{4, 5, 6}},
							},
							StateHistory: []*apiv2beta1.RuntimeStatus{
								{
									UpdateTime: &timestamppb.Timestamp{Seconds: 15},
									State:      apiv2beta1.RuntimeState_FAILED,
									Error:      util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid")),
								},
							},
							ChildTasks: []*apiv2beta1.PipelineTaskDetail_ChildTask{
								{
									ChildTask: &apiv2beta1.PipelineTaskDetail_ChildTask_PodName{PodName: "task2"},
								},
							},
						},
						{
							RunId:          "run1",
							TaskId:         "task2",
							DisplayName:    "this is task 2",
							CreateTime:     timestamppb.New(time.Unix(11, 0)),
							StartTime:      timestamppb.New(time.Unix(12, 0)),
							EndTime:        timestamppb.New(time.Unix(13, 0)),
							ExecutorDetail: nil,
							State:          apiv2beta1.RuntimeState_CANCELED,
							ExecutionId:    14,
							Inputs: map[string]*apiv2beta1.ArtifactList{
								"a1": {ArtifactIds: []int64{1, 2, 3}},
							},
							Outputs: map[string]*apiv2beta1.ArtifactList{
								"b2": {ArtifactIds: []int64{4, 5, 6}},
							},
							ParentTaskId: "task1",
							StateHistory: []*apiv2beta1.RuntimeStatus{
								{
									UpdateTime: &timestamppb.Timestamp{Seconds: 15},
									State:      apiv2beta1.RuntimeState_CANCELED,
								},
							},
						},
					},
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
					TaskDetails: []*model.Task{
						{
							UUID:              "task1",
							Namespace:         "",
							PipelineName:      "",
							RunId:             "run2",
							MLMDExecutionID:   "14",
							CreatedTimestamp:  11,
							StartedTimestamp:  12,
							FinishedTimestamp: 13,
							Fingerprint:       "",
							Name:              "this is task",
							State:             model.RuntimeStateFailed,
							MLMDInputs:        `{"a1":{"artifact_ids":[1,2,3]}}`,
							MLMDOutputs:       `{"b2":{"artifact_ids":[4,5,6]}}`,
							StateHistory: []*model.RuntimeStatus{
								{
									UpdateTimeInSec: 15,
									State:           model.RuntimeStateFailed,
									Error:           util.ToError(util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid"))),
								},
							},
						},
					},
				},
				ResourceReferences: nil,
				Metrics:            nil,
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
					TaskDetails: []*apiv2beta1.PipelineTaskDetail{
						{
							RunId:          "run2",
							TaskId:         "task1",
							DisplayName:    "this is task",
							CreateTime:     timestamppb.New(time.Unix(11, 0)),
							StartTime:      timestamppb.New(time.Unix(12, 0)),
							EndTime:        timestamppb.New(time.Unix(13, 0)),
							ExecutorDetail: nil,
							State:          apiv2beta1.RuntimeState_FAILED,
							ExecutionId:    14,
							Inputs: map[string]*apiv2beta1.ArtifactList{
								"a1": {ArtifactIds: []int64{1, 2, 3}},
							},
							Outputs: map[string]*apiv2beta1.ArtifactList{
								"b2": {ArtifactIds: []int64{4, 5, 6}},
							},
							StateHistory: []*apiv2beta1.RuntimeStatus{
								{
									UpdateTime: &timestamppb.Timestamp{Seconds: 15},
									State:      apiv2beta1.RuntimeState_FAILED,
									Error:      util.ToRpcStatus(util.NewInvalidInputError("Input argument is invalid")),
								},
							},
						},
					},
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
					TaskDetails: []*model.Task{
						{
							UUID:              "task1",
							Namespace:         "",
							PipelineName:      "",
							RunId:             "run2",
							MLMDExecutionID:   "14",
							CreatedTimestamp:  11,
							StartedTimestamp:  12,
							FinishedTimestamp: 13,
							Fingerprint:       "",
							Name:              "this is task",
							State:             model.RuntimeStateCancelling,
							MLMDInputs:        `{"a1":{"artifact_ids":[1,2,3]}}`,
							MLMDOutputs:       `{"b2":{"artifact_ids":[4,5,6]}}`,
							StateHistory: []*model.RuntimeStatus{
								{
									UpdateTimeInSec: 15,
									State:           model.RuntimeStateCancelling,
								},
							},
							ChildrenPods: []string{"task3", "task4"},
						},
					},
				},
				ResourceReferences: nil,
				Metrics:            nil,
				Namespace:          "",
				K8SName:            "",
			},
			nil,
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
					TaskDetails: []*model.Task{
						{
							UUID:              "task1",
							Namespace:         "",
							PipelineName:      "",
							RunId:             "run2",
							MLMDExecutionID:   "14",
							CreatedTimestamp:  11,
							StartedTimestamp:  12,
							FinishedTimestamp: 13,
							Fingerprint:       "",
							Name:              "this is task",
							State:             model.RuntimeStatePaused,
							MLMDInputs:        `{"a1":{"artifact_ids":[1,2,3]}}`,
							MLMDOutputs:       `{"b2":{"artifact_ids":[4,5,6]}}`,
							StateHistory: []*model.RuntimeStatus{
								{
									UpdateTimeInSec: 15,
									State:           model.RuntimeStatePaused,
								},
							},
							ChildrenPods: []string{"task3", "task4"},
						},
					},
				},
				ResourceReferences: nil,
				Metrics:            nil,
				Namespace:          "",
				K8SName:            "",
			},
			nil,
			true,
			"Failed to convert internal run representation to its API counterpart due to missing pipeline source",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toApiRun(tt.arg)
			if tt.wantErr {
				assert.Contains(t, got.Error.Message, tt.errMsg)
				assert.Equal(t, &apiv2beta1.Run{RunId: "run1", ExperimentId: "exp1", Error: got.GetError()}, got)
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
