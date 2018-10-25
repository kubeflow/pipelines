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

	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/stretchr/testify/assert"
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

func TestToModelRun(t *testing.T) {
	apiRun := &api.Run{
		Id:          "run1",
		Name:        "name1",
		Description: "this is a run",
		PipelineSpec: &api.PipelineSpec{
			Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
		},
	}
	modelRun, err := ToModelRun(apiRun, "workflow spec")
	assert.Nil(t, err)

	expectedModelRun := &model.Run{
		DisplayName: "name1",
		Description: "this is a run",
		PipelineSpec: model.PipelineSpec{
			WorkflowSpecManifest: "workflow spec",
			Parameters:           `[{"name":"param2","value":"world"}]`,
		},
	}
	assert.Equal(t, expectedModelRun, modelRun)
}

func TestToModelJob(t *testing.T) {
	apiJob := &api.Job{
		Id:             "job1",
		Name:           "name1",
		PipelineId:     "1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: &api.Trigger{
			Trigger: &api.Trigger_CronSchedule{CronSchedule: &api.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		Parameters: []*api.Parameter{{Name: "param2", Value: "world"}},
	}
	modelJob, err := ToModelJob(apiJob)
	assert.Nil(t, err)

	expectedModelJob := &model.Job{
		DisplayName: "name1",
		Enabled:     true,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				Cron:                       util.StringPointer("1 * * * *"),
			},
		},
		MaxConcurrency: 1,
		PipelineSpec: model.PipelineSpec{
			PipelineId: "1",
			Parameters: `[{"name":"param2","value":"world"}]`,
		},
	}
	assert.Equal(t, expectedModelJob, modelJob)
}
