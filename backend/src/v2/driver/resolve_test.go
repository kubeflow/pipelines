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
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGetProducerTask_SelectsSuccessfulArtifactOneOfBranch(t *testing.T) {
	outputArtifacts := map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{
		"pipelinechannel--condition-branches-1-oneof-1": {
			ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{
				{
					ProducerSubtask:   "condition-2",
					OutputArtifactKey: "pipelinechannel--param-to-artifact-a",
				},
				{
					ProducerSubtask:   "condition-3",
					OutputArtifactKey: "pipelinechannel--param-to-artifact-2-a",
				},
			},
		},
	}
	serializedOutputArtifacts, err := json.Marshal(outputArtifacts)
	require.NoError(t, err)

	parentDagID := int64(515)
	parentTask := metadata.NewExecution(&pb.Execution{
		Id: &parentDagID,
		CustomProperties: map[string]*pb.Value{
			"task_name": {
				Value: &pb.Value_StringValue{
					StringValue: "condition-branches-1",
				},
			},
			"artifact_producer_task": {
				Value: &pb.Value_StringValue{
					StringValue: string(serializedOutputArtifacts),
				},
			},
		},
	})

	tasks := map[string]*metadata.Execution{
		metadata.GetTaskNameWithDagID("condition-2", parentDagID): metadata.NewExecution(&pb.Execution{
			LastKnownState: pb.Execution_COMPLETE.Enum(),
		}),
		metadata.GetTaskNameWithDagID("condition-3", parentDagID): metadata.NewExecution(&pb.Execution{
			LastKnownState: pb.Execution_CANCELED.Enum(),
		}),
	}

	producerTaskName, outputArtifactKey, err := GetProducerTask(
		parentTask,
		tasks,
		"condition-branches-1",
		"pipelinechannel--condition-branches-1-oneof-1",
		true,
	)
	require.NoError(t, err)
	assert.Equal(t, metadata.GetTaskNameWithDagID("condition-2", parentDagID), producerTaskName)
	assert.Equal(t, "pipelinechannel--param-to-artifact-a", outputArtifactKey)
}

func TestGetProducerTask_SelectsLaterSuccessfulArtifactOneOfBranch(t *testing.T) {
	outputArtifacts := map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{
		"pipelinechannel--condition-branches-1-oneof-1": {
			ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{
				{
					ProducerSubtask:   "condition-2",
					OutputArtifactKey: "pipelinechannel--param-to-artifact-a",
				},
				{
					ProducerSubtask:   "condition-3",
					OutputArtifactKey: "pipelinechannel--param-to-artifact-2-a",
				},
			},
		},
	}
	serializedOutputArtifacts, err := json.Marshal(outputArtifacts)
	require.NoError(t, err)

	parentDagID := int64(515)
	parentTask := metadata.NewExecution(&pb.Execution{
		Id: &parentDagID,
		CustomProperties: map[string]*pb.Value{
			"task_name": {
				Value: &pb.Value_StringValue{
					StringValue: "condition-branches-1",
				},
			},
			"artifact_producer_task": {
				Value: &pb.Value_StringValue{
					StringValue: string(serializedOutputArtifacts),
				},
			},
		},
	})

	tasks := map[string]*metadata.Execution{
		metadata.GetTaskNameWithDagID("condition-2", parentDagID): metadata.NewExecution(&pb.Execution{
			LastKnownState: pb.Execution_CANCELED.Enum(),
		}),
		metadata.GetTaskNameWithDagID("condition-3", parentDagID): metadata.NewExecution(&pb.Execution{
			LastKnownState: pb.Execution_COMPLETE.Enum(),
		}),
	}

	producerTaskName, outputArtifactKey, err := GetProducerTask(
		parentTask,
		tasks,
		"condition-branches-1",
		"pipelinechannel--condition-branches-1-oneof-1",
		true,
	)
	require.NoError(t, err)
	assert.Equal(t, metadata.GetTaskNameWithDagID("condition-3", parentDagID), producerTaskName)
	assert.Equal(t, "pipelinechannel--param-to-artifact-2-a", outputArtifactKey)
}

func TestGetProducerTask_ArtifactOneOfReturnsErrorWhenNoBranchSucceeds(t *testing.T) {
	outputArtifacts := map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{
		"pipelinechannel--condition-branches-1-oneof-1": {
			ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{
				{
					ProducerSubtask:   "condition-2",
					OutputArtifactKey: "pipelinechannel--param-to-artifact-a",
				},
				{
					ProducerSubtask:   "condition-3",
					OutputArtifactKey: "pipelinechannel--param-to-artifact-2-a",
				},
			},
		},
	}
	serializedOutputArtifacts, err := json.Marshal(outputArtifacts)
	require.NoError(t, err)

	parentDagID := int64(515)
	parentTask := metadata.NewExecution(&pb.Execution{
		Id: &parentDagID,
		CustomProperties: map[string]*pb.Value{
			"task_name": {
				Value: &pb.Value_StringValue{
					StringValue: "condition-branches-1",
				},
			},
			"artifact_producer_task": {
				Value: &pb.Value_StringValue{
					StringValue: string(serializedOutputArtifacts),
				},
			},
		},
	})

	tasks := map[string]*metadata.Execution{
		metadata.GetTaskNameWithDagID("condition-2", parentDagID): metadata.NewExecution(&pb.Execution{
			LastKnownState: pb.Execution_CANCELED.Enum(),
		}),
		metadata.GetTaskNameWithDagID("condition-3", parentDagID): metadata.NewExecution(&pb.Execution{
			LastKnownState: pb.Execution_FAILED.Enum(),
		}),
	}

	_, _, err = GetProducerTask(
		parentTask,
		tasks,
		"condition-branches-1",
		"pipelinechannel--condition-branches-1-oneof-1",
		true,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "processing OneOf: No successful task found")
	assert.Contains(t, err.Error(), "pipelinechannel--condition-branches-1-oneof-1")
	assert.Contains(t, err.Error(), "condition-branches-1")
}

func TestGetProducerTask_SelectsCachedArtifactOneOfBranch(t *testing.T) {
	outputArtifacts := map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{
		"pipelinechannel--condition-branches-1-oneof-1": {
			ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{
				{
					ProducerSubtask:   "condition-2",
					OutputArtifactKey: "pipelinechannel--param-to-artifact-a",
				},
				{
					ProducerSubtask:   "condition-3",
					OutputArtifactKey: "pipelinechannel--param-to-artifact-2-a",
				},
			},
		},
	}
	serializedOutputArtifacts, err := json.Marshal(outputArtifacts)
	require.NoError(t, err)

	parentDagID := int64(515)
	parentTask := metadata.NewExecution(&pb.Execution{
		Id: &parentDagID,
		CustomProperties: map[string]*pb.Value{
			"task_name": {
				Value: &pb.Value_StringValue{
					StringValue: "condition-branches-1",
				},
			},
			"artifact_producer_task": {
				Value: &pb.Value_StringValue{
					StringValue: string(serializedOutputArtifacts),
				},
			},
		},
	})

	tasks := map[string]*metadata.Execution{
		metadata.GetTaskNameWithDagID("condition-2", parentDagID): metadata.NewExecution(&pb.Execution{
			LastKnownState: pb.Execution_CACHED.Enum(),
		}),
		metadata.GetTaskNameWithDagID("condition-3", parentDagID): metadata.NewExecution(&pb.Execution{
			LastKnownState: pb.Execution_CANCELED.Enum(),
		}),
	}

	producerTaskName, outputArtifactKey, err := GetProducerTask(
		parentTask,
		tasks,
		"condition-branches-1",
		"pipelinechannel--condition-branches-1-oneof-1",
		true,
	)
	require.NoError(t, err)
	assert.Equal(t, metadata.GetTaskNameWithDagID("condition-2", parentDagID), producerTaskName)
	assert.Equal(t, "pipelinechannel--param-to-artifact-a", outputArtifactKey)
}

func TestGetProducerTask_ArtifactOneOfReturnsErrorWhenSubtasksAreMissing(t *testing.T) {
	outputArtifacts := map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{
		"pipelinechannel--condition-branches-1-oneof-1": {
			ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{
				{
					ProducerSubtask:   "condition-2",
					OutputArtifactKey: "pipelinechannel--param-to-artifact-a",
				},
				{
					ProducerSubtask:   "condition-3",
					OutputArtifactKey: "pipelinechannel--param-to-artifact-2-a",
				},
			},
		},
	}
	serializedOutputArtifacts, err := json.Marshal(outputArtifacts)
	require.NoError(t, err)

	parentDagID := int64(515)
	parentTask := metadata.NewExecution(&pb.Execution{
		Id: &parentDagID,
		CustomProperties: map[string]*pb.Value{
			"task_name": {
				Value: &pb.Value_StringValue{
					StringValue: "condition-branches-1",
				},
			},
			"artifact_producer_task": {
				Value: &pb.Value_StringValue{
					StringValue: string(serializedOutputArtifacts),
				},
			},
		},
	})

	tasks := map[string]*metadata.Execution{}

	_, _, err = GetProducerTask(
		parentTask,
		tasks,
		"condition-branches-1",
		"pipelinechannel--condition-branches-1-oneof-1",
		true,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "processing OneOf: No successful task found")
}

func TestGetProducerTask_ArtifactOutputKeyMissingReturnsError(t *testing.T) {
	outputArtifacts := map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{
		"some-other-output-key": {
			ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{
				{
					ProducerSubtask:   "condition-2",
					OutputArtifactKey: "pipelinechannel--param-to-artifact-a",
				},
			},
		},
	}
	serializedOutputArtifacts, err := json.Marshal(outputArtifacts)
	require.NoError(t, err)

	parentDagID := int64(515)
	parentTask := metadata.NewExecution(&pb.Execution{
		Id: &parentDagID,
		CustomProperties: map[string]*pb.Value{
			"task_name": {
				Value: &pb.Value_StringValue{
					StringValue: "condition-branches-1",
				},
			},
			"artifact_producer_task": {
				Value: &pb.Value_StringValue{
					StringValue: string(serializedOutputArtifacts),
				},
			},
		},
	})

	_, _, err = GetProducerTask(
		parentTask,
		map[string]*metadata.Execution{},
		"condition-branches-1",
		"pipelinechannel--condition-branches-1-oneof-1",
		true,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "artifact output key")
	assert.Contains(t, err.Error(), "pipelinechannel--condition-branches-1-oneof-1")
	assert.Contains(t, err.Error(), "condition-branches-1")
}

func TestValidateExecutorInputs(t *testing.T) {
	tests := []struct {
		name      string
		inputs    *pipelinespec.ExecutorInput_Inputs
		component *pipelinespec.ComponentSpec
		wantErr   bool
		errStr    string
	}{
		{
			name: "Valid STRING parameter",
			inputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{
					"param1": structpb.NewStringValue("hello"),
				},
			},
			component: &pipelinespec.ComponentSpec{
				InputDefinitions: &pipelinespec.ComponentInputsSpec{
					Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
						"param1": {
							ParameterType: pipelinespec.ParameterType_STRING,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid STRING parameter type mismatch",
			inputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{
					"param1": structpb.NewNumberValue(123),
				},
			},
			component: &pipelinespec.ComponentSpec{
				InputDefinitions: &pipelinespec.ComponentInputsSpec{
					Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
						"param1": {
							ParameterType: pipelinespec.ParameterType_STRING,
						},
					},
				},
			},
			wantErr: true,
			errStr:  "expected STRING, got *structpb.Value_NumberValue",
		},
		{
			name: "Missing required parameter",
			inputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{},
			},
			component: &pipelinespec.ComponentSpec{
				InputDefinitions: &pipelinespec.ComponentInputsSpec{
					Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
						"param1": {
							ParameterType: pipelinespec.ParameterType_STRING,
							IsOptional:    false,
						},
					},
				},
			},
			wantErr: true,
			errStr:  "missing required parameter: \"param1\"",
		},
		{
			name: "Missing optional parameter",
			inputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{},
			},
			component: &pipelinespec.ComponentSpec{
				InputDefinitions: &pipelinespec.ComponentInputsSpec{
					Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
						"param1": {
							ParameterType: pipelinespec.ParameterType_STRING,
							IsOptional:    true,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing required parameter but has DefaultValue",
			inputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{},
			},
			component: &pipelinespec.ComponentSpec{
				InputDefinitions: &pipelinespec.ComponentInputsSpec{
					Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
						"param1": {
							ParameterType: pipelinespec.ParameterType_STRING,
							IsOptional:    false,
							DefaultValue:  structpb.NewStringValue("default"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Optional parameter provided as NullValue",
			inputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{
					"param1": structpb.NewNullValue(),
				},
			},
			component: &pipelinespec.ComponentSpec{
				InputDefinitions: &pipelinespec.ComponentInputsSpec{
					Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
						"param1": {
							ParameterType: pipelinespec.ParameterType_STRING,
							IsOptional:    true,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "NUMBER_INTEGER with float value",
			inputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{
					"param1": structpb.NewNumberValue(123.45),
				},
			},
			component: &pipelinespec.ComponentSpec{
				InputDefinitions: &pipelinespec.ComponentInputsSpec{
					Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
						"param1": {
							ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER,
						},
					},
				},
			},
			wantErr: true,
			errStr:  "expected NUMBER_INTEGER",
		},
		{
			name: "NUMBER_INTEGER with integer value",
			inputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{
					"param1": structpb.NewNumberValue(123.0),
				},
			},
			component: &pipelinespec.ComponentSpec{
				InputDefinitions: &pipelinespec.ComponentInputsSpec{
					Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
						"param1": {
							ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER,
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateExecutorInputs(tc.inputs, tc.component)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errStr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
