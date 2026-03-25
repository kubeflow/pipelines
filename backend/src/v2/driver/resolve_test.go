// Copyright 2025 The Kubeflow Authors
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

package driver

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestResolvePipelineJobCreateTimePlaceholder(t *testing.T) {
	ctx := context.Background()

	paramSpec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
			RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue("{{$.pipeline_job_create_time_utc}}"),
				},
			},
		},
	}

	opts := Options{
		PipelineJobCreateTimeUTC: "2026-01-09T12:34:56Z",
	}

	val, err := resolveInputParameter(
		ctx,
		nil, // task
		nil, // component spec
		opts,
		nil, // runtime config
		paramSpec,
		map[string]*structpb.Value{},
	)

	assert.NoError(t, err)
	assert.Equal(t, "2026-01-09T12:34:56Z", val.GetStringValue())
}

func TestResolvePipelineJobScheduleTimePlaceholder(t *testing.T) {
	ctx := context.Background()

	paramSpec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
			RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue("{{$.pipeline_job_schedule_time_utc}}"),
				},
			},
		},
	}

	opts := Options{
		PipelineJobScheduleTimeUTC: "2026-01-09T13:00:00Z",
	}

	val, err := resolveInputParameter(
		ctx,
		nil,
		nil,
		opts,
		nil,
		paramSpec,
		map[string]*structpb.Value{},
	)

	assert.NoError(t, err)
	assert.Equal(t, "2026-01-09T13:00:00Z", val.GetStringValue())
}

func Test_InferIndexedTaskName(t *testing.T) {
	tests := []struct {
		name             string
		producerTaskName string
		dag              *metadata.Execution
		expected         string
	}{
		{
			name:             "DAG with iteration_index returns indexed task name",
			producerTaskName: "my-task",
			dag: metadata.NewExecution(&pb.Execution{
				CustomProperties: map[string]*pb.Value{
					"iteration_index": {Value: &pb.Value_IntValue{IntValue: 3}},
				},
			}),
			expected: "my-task_idx_3",
		},
		{
			name:             "DAG with iteration_index 0 returns indexed task name",
			producerTaskName: "my-task",
			dag: metadata.NewExecution(&pb.Execution{
				CustomProperties: map[string]*pb.Value{
					"iteration_index": {Value: &pb.Value_IntValue{IntValue: 0}},
				},
			}),
			expected: "my-task_idx_0",
		},
		{
			name:             "DAG without iteration_index returns original task name",
			producerTaskName: "my-task",
			dag: metadata.NewExecution(&pb.Execution{
				CustomProperties: map[string]*pb.Value{},
			}),
			expected: "my-task",
		},
		{
			name:             "DAG with nil custom properties returns original task name",
			producerTaskName: "my-task",
			dag:              metadata.NewExecution(&pb.Execution{}),
			expected:         "my-task",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := InferIndexedTaskName(test.producerTaskName, test.dag)
			assert.Equal(t, test.expected, result)
		})
	}
}
