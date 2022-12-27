// Copyright 2022-2022 The Kubeflow Authors
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
	"testing"

	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/ptypes/timestamp"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	commonApiRecurringRun = &apiv2beta1.RecurringRun{
		DisplayName:    "job1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: &structpb.Struct{}},
		ExperimentId:   "123e4567-e89b-12d3-a456-426655440000",
	}
)

func TestValidateApiRecurringRun(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRecurringRunServer(manager, &RecurringRunServerOptions{CollectMetrics: false})
	err := server.validateCreateRecurringRunRequest(&apiv2beta1.CreateRecurringRunRequest{RecurringRun: commonApiRecurringRun})
	assert.Nil(t, err)
}

func TestCreateRecurringRun(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRecurringRunServer(manager, &RecurringRunServerOptions{CollectMetrics: false})

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "recurring_run_1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		Namespace:      "ns1",
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
		ExperimentId: "123e4567-e89b-12d3-a456-426655440000",
	}

	expectedRecurringRun := &apiv2beta1.RecurringRun{
		RecurringRunId: "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "recurring_run_1",
		ServiceAccount: "pipeline-runner",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		Namespace:      "ns1",
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 2},
		Status:         apiv2beta1.RecurringRun_STATUS_UNSPECIFIED,
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
	}

	recurringRun, err := server.CreateRecurringRun(nil, &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)
	assert.Equal(t, expectedRecurringRun, recurringRun)

}

func TestGetRecurringRun(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRecurringRunServer(manager, &RecurringRunServerOptions{CollectMetrics: false})

	pipelineSpecStruct := &structpb.Struct{}
	yaml.Unmarshal([]byte(v2SpecHelloWorld), pipelineSpecStruct)

	apiRecurringRun := &apiv2beta1.RecurringRun{
		DisplayName:    "recurring_run_1",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		Namespace:      "ns1",
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
		ExperimentId: "123e4567-e89b-12d3-a456-426655440000",
	}

	expectedRecurringRun := &apiv2beta1.RecurringRun{
		RecurringRunId: "123e4567-e89b-12d3-a456-426655440000",
		DisplayName:    "recurring_run_1",
		ServiceAccount: "pipeline-runner",
		Mode:           apiv2beta1.RecurringRun_ENABLE,
		Namespace:      "ns1",
		MaxConcurrency: 1,
		Trigger: &apiv2beta1.Trigger{
			Trigger: &apiv2beta1.Trigger_CronSchedule{CronSchedule: &apiv2beta1.CronSchedule{
				StartTime: &timestamp.Timestamp{Seconds: 1},
				Cron:      "1 * * * *",
			}}},
		CreatedAt:      &timestamp.Timestamp{Seconds: 2},
		UpdatedAt:      &timestamp.Timestamp{Seconds: 2},
		Status:         apiv2beta1.RecurringRun_STATUS_UNSPECIFIED,
		PipelineSource: &apiv2beta1.RecurringRun_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig: &apiv2beta1.RuntimeConfig{
			PipelineRoot: "model-pipeline-root",
		},
	}

	createdRecurringRun, err := server.CreateRecurringRun(nil, &apiv2beta1.CreateRecurringRunRequest{RecurringRun: apiRecurringRun})
	assert.Nil(t, err)

	recurringRun, err := server.GetRecurringRun(nil, &apiv2beta1.GetRecurringRunRequest{RecurringRunId: createdRecurringRun.RecurringRunId})
	assert.Nil(t, err)
	assert.Equal(t, expectedRecurringRun, recurringRun)

}
