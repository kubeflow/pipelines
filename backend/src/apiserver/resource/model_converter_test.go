// Copyright 2018 Google LLC
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

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/apiserver/common"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	swfapi "github.com/googleprivate/ml/backend/src/crd/pkg/apis/scheduledworkflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToModelRunMetric(t *testing.T) {
	apiRunMetric := &api.RunMetric{
		Name:   "metric-1",
		NodeId: "node-1",
		Value: &api.RunMetric_NumberValue{
			NumberValue: 0.88,
		},
		Format: api.RunMetric_RAW,
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

func TestToModelRunDetail(t *testing.T) {
	apiRun := &api.Run{
		Id:          "run1",
		Name:        "name1",
		Description: "this is a run",
		PipelineSpec: &api.PipelineSpec{
			Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
		},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: "exp1"}, Relationship: api.Relationship_OWNER},
		},
	}
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name", UID: "123"},
		Status:     v1alpha1.WorkflowStatus{Phase: "running"},
	})
	modelRunDetail, err := ToModelRunDetail(apiRun, workflow, "workflow spec")
	assert.Nil(t, err)

	expectedModelRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:        "123",
			DisplayName: "name1",
			Name:        "workflow-name",
			Conditions:  "running",
			Description: "this is a run",
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: "workflow spec",
				Parameters:           `[{"name":"param2","value":"world"}]`,
			},
			ResourceReferences: []*model.ResourceReference{
				{
					ResourceUUID:  "123",
					ResourceType:  common.Run,
					ReferenceUUID: "exp1",
					ReferenceType: common.Experiment,
					Relationship:  common.Owner},
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: workflow.ToStringForStore(),
		},
	}
	assert.Equal(t, expectedModelRunDetail, modelRunDetail)
}

func TestToModelJob(t *testing.T) {
	apiJob := &api.Job{
		Name:           "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: "exp1"}, Relationship: api.Relationship_OWNER},
		},
		PipelineSpec: &api.PipelineSpec{PipelineId: "1", Parameters: []*api.Parameter{{Name: "param2", Value: "world"}}},
	}
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "swf_name",
			Namespace: "swf_namespace",
			UID:       "swf_123",
		},
		Status: swfapi.ScheduledWorkflowStatus{
			Conditions: []swfapi.ScheduledWorkflowCondition{{Type: swfapi.ScheduledWorkflowEnabled}}},
	})
	modelJob, err := ToModelJob(apiJob, swf, "workflow spec")
	assert.Nil(t, err)

	expectedModelJob := &model.Job{
		UUID:        "swf_123",
		Name:        "swf_name",
		Namespace:   "swf_namespace",
		Conditions:  "Enabled",
		DisplayName: "name1",
		Enabled:     true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				Cron: util.StringPointer("1 * * * *"),
			},
		},
		MaxConcurrency: 1,
		PipelineSpec: model.PipelineSpec{
			PipelineId:           "1",
			WorkflowSpecManifest: "workflow spec",
			Parameters:           `[{"name":"param2","value":"world"}]`,
		},
		ResourceReferences: []*model.ResourceReference{
			{
				ResourceUUID:  "swf_123",
				ResourceType:  common.Job,
				ReferenceUUID: "exp1",
				ReferenceType: common.Experiment,
				Relationship:  common.Owner},
		},
	}
	assert.Equal(t, expectedModelJob, modelJob)
}

func TestToModelResourceReferences(t *testing.T) {
	refs, err := toModelResourceReferences("r1", common.Run, []*api.ResourceReference{
		{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: "e1"}, Relationship: api.Relationship_OWNER},
		{Key: &api.ResourceKey{Type: api.ResourceType_JOB, Id: "j1"}, Relationship: api.Relationship_CREATOR},
	})
	assert.Nil(t, err)
	expectedRefs := []*model.ResourceReference{
		{ResourceUUID: "r1", ResourceType: common.Run,
			ReferenceUUID: "e1", ReferenceType: common.Experiment, Relationship: common.Owner},
		{ResourceUUID: "r1", ResourceType: common.Run,
			ReferenceUUID: "j1", ReferenceType: common.Job, Relationship: common.Creator},
	}
	assert.Equal(t, expectedRefs, refs)
}

func TestToModelResourceReferences_UnknownRefType(t *testing.T) {
	_, err := toModelResourceReferences("r1", common.Run, []*api.ResourceReference{
		{Key: &api.ResourceKey{Type: api.ResourceType_UNKNOWN_RESOURCE_TYPE, Id: "e1"}, Relationship: api.Relationship_OWNER},
		{Key: &api.ResourceKey{Type: api.ResourceType_JOB, Id: "j1"}, Relationship: api.Relationship_CREATOR},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to convert reference type")
}

func TestToModelResourceReferences_UnknownRelationship(t *testing.T) {
	_, err := toModelResourceReferences("r1", common.Run, []*api.ResourceReference{
		{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: "e1"}, Relationship: api.Relationship_UNKNOWN_RELATIONSHIP},
		{Key: &api.ResourceKey{Type: api.ResourceType_JOB, Id: "j1"}, Relationship: api.Relationship_CREATOR},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to convert relationship")
}
