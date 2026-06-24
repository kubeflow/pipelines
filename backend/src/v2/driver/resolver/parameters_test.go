// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resolver

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestResolveTaskOutputParameter_FindsIterationScopedNonRuntimeProducer(t *testing.T) {
	tests := []struct {
		name         string
		producerType apiv2beta1.PipelineTask_TaskType
	}{
		{
			name:         "dag producer",
			producerType: apiv2beta1.PipelineTask_DAG,
		},
		{
			name:         "importer producer",
			producerType: apiv2beta1.PipelineTask_IMPORTER,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parentTaskID := "loop-parent"
			parentTask := &apiv2beta1.PipelineTask{
				TaskId: parentTaskID,
				Name:   "loop-body",
				Type:   apiv2beta1.PipelineTask_LOOP,
			}

			outputValue := structpb.NewStringValue("resolved")
			producerTask := &apiv2beta1.PipelineTask{
				TaskId:       "producer-task",
				Name:         "produce",
				ParentTaskId: util.StringPointer(parentTaskID),
				Type:         test.producerType,
				TypeAttributes: &apiv2beta1.PipelineTask_TypeAttributes{
					IterationIndex: util.Int64Pointer(0),
				},
				Outputs: &apiv2beta1.PipelineTask_InputOutputs{
					Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{
						{
							ParameterKey: "result",
							Value:        outputValue,
							Producer: &apiv2beta1.IOProducer{
								TaskName: "produce",
							},
						},
					},
				},
			}

			opts := common.Options{
				ParentTask:     parentTask,
				Run:            &apiv2beta1.Run{Tasks: []*apiv2beta1.PipelineTask{producerTask}},
				IterationIndex: 0,
			}

			paramSpec := common.InputParamTaskOutput("produce", "result")

			resolved, err := resolveTaskOutputParameter(opts, paramSpec)
			require.NoError(t, err)
			require.NotNil(t, resolved)
			assert.Equal(t, "resolved", resolved.GetValue().GetStringValue())
		})
	}
}

func TestResolveTaskFinalStatus_FindsIterationScopedProducer(t *testing.T) {
	parentTaskID := "loop-parent"
	parentTask := &apiv2beta1.PipelineTask{
		TaskId: parentTaskID,
		Name:   "loop-body",
		Type:   apiv2beta1.PipelineTask_LOOP,
	}

	producerTask := &apiv2beta1.PipelineTask{
		TaskId:       "producer-task",
		Name:         "produce",
		ParentTaskId: util.StringPointer(parentTaskID),
		Type:         apiv2beta1.PipelineTask_RUNTIME,
		State:        apiv2beta1.PipelineTask_FAILED,
		TypeAttributes: &apiv2beta1.PipelineTask_TypeAttributes{
			IterationIndex: util.Int64Pointer(1),
		},
		StatusMetadata: &apiv2beta1.PipelineTask_StatusMetadata{
			Message: "boom",
		},
	}

	opts := common.Options{
		ParentTask:     parentTask,
		Run:            &apiv2beta1.Run{Tasks: []*apiv2beta1.PipelineTask{producerTask}},
		RunName:        "run-name",
		IterationIndex: 1,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "cleanup"},
			DependentTasks: []string{"produce"},
		},
	}

	paramSpec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{
				ProducerTask: "produce",
			},
		},
	}

	resolved, err := resolveTaskFinalStatus(opts, paramSpec)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "FAILED", resolved.GetStructValue().GetFields()["state"].GetStringValue())
	assert.Equal(t, "produce", resolved.GetStructValue().GetFields()["pipelineTaskName"].GetStringValue())
}

func TestResolveParameters_AppliesSelectorAndStringCoercion(t *testing.T) {
	inputValue, err := structpb.NewValue(map[string]interface{}{"name": 42})
	require.NoError(t, err)

	parentTask := &apiv2beta1.PipelineTask{
		TaskId: "parent-task",
		Inputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{
				{
					ParameterKey: "item",
					Value:        inputValue,
					Producer:     &apiv2beta1.IOProducer{TaskName: "upstream"},
				},
			},
		},
	}

	opts := common.Options{
		ParentTask: parentTask,
		Task: &pipelinespec.PipelineTaskSpec{
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"name": {
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
							ComponentInputParameter: "item",
						},
						ParameterExpressionSelector: "struct_value.name",
					},
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"name": {ParameterType: pipelinespec.ParameterType_STRING},
				},
			},
		},
	}

	parameters, err := resolveParameters(opts)
	require.NoError(t, err)
	require.Len(t, parameters, 1)
	assert.Equal(t, "42", parameters[0].ParameterIO.GetValue().GetStringValue())
}

func TestResolveParameters_ValidatesLiterals(t *testing.T) {
	parentTask := &apiv2beta1.PipelineTask{
		TaskId: "parent-task",
		Inputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{
				{
					ParameterKey: "mode",
					Value:        structpb.NewStringValue("invalid"),
					Producer:     &apiv2beta1.IOProducer{TaskName: "upstream"},
				},
			},
		},
	}

	opts := common.Options{
		ParentTask: parentTask,
		Task: &pipelinespec.PipelineTaskSpec{
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"mode": common.InputParamComponent("mode"),
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"mode": {
						ParameterType: pipelinespec.ParameterType_STRING,
						Literals: []*structpb.Value{
							structpb.NewStringValue("train"),
							structpb.NewStringValue("eval"),
						},
					},
				},
			},
		},
	}

	_, err := resolveParameters(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validating parameter")
}
