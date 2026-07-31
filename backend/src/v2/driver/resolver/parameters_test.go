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
	"google.golang.org/grpc/codes"
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
	assert.Equal(t, float64(codes.Unknown), resolved.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())
}

func TestResolveTaskFinalStatus_UsesOKCodeForSuccessfulTasks(t *testing.T) {
	parentTaskID := "parent"
	parentTask := &apiv2beta1.PipelineTask{TaskId: parentTaskID, Name: "parent"}
	producerTask := &apiv2beta1.PipelineTask{
		TaskId:       "producer-task",
		Name:         "produce",
		ParentTaskId: util.StringPointer(parentTaskID),
		Type:         apiv2beta1.PipelineTask_RUNTIME,
		State:        apiv2beta1.PipelineTask_SUCCEEDED,
	}
	opts := common.Options{
		ParentTask:     parentTask,
		Run:            &apiv2beta1.Run{Tasks: []*apiv2beta1.PipelineTask{producerTask}},
		RunName:        "run-name",
		IterationIndex: -1,
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
	assert.Equal(t, float64(codes.OK), resolved.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())
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

func TestResolveParameters_RejectsMissingRequiredInput(t *testing.T) {
	opts := common.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: "parent-task",
			Inputs: &apiv2beta1.PipelineTask_InputOutputs{},
		},
		Task: &pipelinespec.PipelineTaskSpec{
			Inputs: &pipelinespec.TaskInputsSpec{},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"required": {ParameterType: pipelinespec.ParameterType_STRING},
				},
			},
		},
	}

	_, err := resolveParameters(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-optional parameter")
}

func TestResolveParameters_DoesNotTreatOrdinaryLoopNamedInputAsIterator(t *testing.T) {
	opts := common.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: "parent-task",
			Inputs: &apiv2beta1.PipelineTask_InputOutputs{
				Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{
					{
						ParameterKey: "loop-item-user",
						Value:        structpb.NewStringValue("value"),
						Producer:     &apiv2beta1.IOProducer{TaskName: "upstream"},
					},
				},
			},
		},
		Task: &pipelinespec.PipelineTaskSpec{
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"loop-item-user": common.InputParamComponent("loop-item-user"),
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"loop-item-user": {ParameterType: pipelinespec.ParameterType_STRING},
				},
			},
		},
		IterationIndex: 1,
	}

	parameters, err := resolveParameters(opts)
	require.NoError(t, err)
	require.Len(t, parameters, 1)
	assert.Equal(t, "value", parameters[0].ParameterIO.GetValue().GetStringValue())
}

func TestResolveParameters_NestedLoopInputsUseCurrentIteratorOnly(t *testing.T) {
	opts := common.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: "parent-task",
			Inputs: &apiv2beta1.PipelineTask_InputOutputs{
				Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{
					{
						ParameterKey: "pipelinechannel--loop-item-param-3",
						Value:        structpb.NewStringValue("outer-0"),
						Producer: &apiv2beta1.IOProducer{
							TaskName:  "outer-loop",
							Iteration: util.Int64Pointer(0),
						},
					},
					{
						ParameterKey: "pipelinechannel--loop-item-param-5",
						Value:        structpb.NewStringValue("inner-0"),
						Producer: &apiv2beta1.IOProducer{
							TaskName:  "inner-loop",
							Iteration: util.Int64Pointer(0),
						},
					},
					{
						ParameterKey: "pipelinechannel--loop-item-param-5",
						Value:        structpb.NewStringValue("inner-1"),
						Producer: &apiv2beta1.IOProducer{
							TaskName:  "inner-loop",
							Iteration: util.Int64Pointer(1),
						},
					},
				},
			},
		},
		Task: &pipelinespec.PipelineTaskSpec{
			Inputs: &pipelinespec.TaskInputsSpec{
				Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
					"outer": common.InputParamComponent("pipelinechannel--loop-item-param-3"),
					"inner": common.InputParamComponent("pipelinechannel--loop-item-param-5"),
				},
			},
			Iterator: &pipelinespec.PipelineTaskSpec_ParameterIterator{
				ParameterIterator: &pipelinespec.ParameterIteratorSpec{
					ItemInput: "pipelinechannel--loop-item-param-5",
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
					"outer": {ParameterType: pipelinespec.ParameterType_STRING},
					"inner": {ParameterType: pipelinespec.ParameterType_STRING},
				},
			},
		},
		IterationIndex: 1,
	}

	parameters, err := resolveParameters(opts)
	require.NoError(t, err)
	require.Len(t, parameters, 2)
	valuesByKey := map[string]string{}
	for _, parameter := range parameters {
		valuesByKey[parameter.Key] = parameter.ParameterIO.GetValue().GetStringValue()
	}
	assert.Equal(t, "outer-0", valuesByKey["outer"])
	assert.Equal(t, "inner-1", valuesByKey["inner"])
}

func TestResolveTaskOutputParameter_EmptyLoopProducesEmptyCollection(t *testing.T) {
	parentTaskID := "dag-parent"
	parentTask := &apiv2beta1.PipelineTask{TaskId: parentTaskID, Name: "dag"}
	producerTask := &apiv2beta1.PipelineTask{
		TaskId:       "loop-task",
		Name:         "loop",
		ParentTaskId: util.StringPointer(parentTaskID),
		Type:         apiv2beta1.PipelineTask_LOOP,
		TypeAttributes: &apiv2beta1.PipelineTask_TypeAttributes{
			IterationCount: util.Int64Pointer(0),
		},
	}
	opts := common.Options{
		ParentTask:     parentTask,
		Run:            &apiv2beta1.Run{Tasks: []*apiv2beta1.PipelineTask{producerTask}},
		IterationIndex: -1,
	}

	resolved, err := resolveTaskOutputParameter(opts, common.InputParamTaskOutput("loop", "result"))
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, apiv2beta1.IOType_COLLECTED_INPUTS, resolved.GetType())
	assert.Empty(t, resolved.GetValue().GetListValue().GetValues())
}

func TestFindParameterByProducerKeyInList_OrdersIteratorOutputs(t *testing.T) {
	parameters := []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{
		iteratorParameter("result", 2, "two"),
		iteratorParameter("result", 0, "zero"),
		iteratorParameter("result", 1, "one"),
	}

	resolved, err := findParameterByProducerKeyInList("result", "producer", parameters)

	require.NoError(t, err)
	assert.Equal(t, apiv2beta1.IOType_COLLECTED_INPUTS, resolved.GetType())
	require.Len(t, resolved.GetValue().GetListValue().GetValues(), 3)
	assert.Equal(t, "zero", resolved.GetValue().GetListValue().GetValues()[0].GetStringValue())
	assert.Equal(t, "one", resolved.GetValue().GetListValue().GetValues()[1].GetStringValue())
	assert.Equal(t, "two", resolved.GetValue().GetListValue().GetValues()[2].GetStringValue())
}

func TestFindParameterByProducerKeyInList_WrapsSingletonIteratorOutput(t *testing.T) {
	resolved, err := findParameterByProducerKeyInList(
		"result",
		"producer",
		[]*apiv2beta1.PipelineTask_InputOutputs_IOParameter{
			iteratorParameter("result", 0, "only"),
		},
	)

	require.NoError(t, err)
	assert.Equal(t, apiv2beta1.IOType_COLLECTED_INPUTS, resolved.GetType())
	require.Len(t, resolved.GetValue().GetListValue().GetValues(), 1)
	assert.Equal(t, "only", resolved.GetValue().GetListValue().GetValues()[0].GetStringValue())
}

func TestFindParameterByProducerKeyInList_RejectsInvalidIteratorMetadata(t *testing.T) {
	tests := []struct {
		name       string
		parameters []*apiv2beta1.PipelineTask_InputOutputs_IOParameter
		errorText  string
	}{
		{
			name: "duplicate iteration",
			parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{
				iteratorParameter("result", 1, "first"),
				iteratorParameter("result", 1, "second"),
			},
			errorText: "duplicate iteration index 1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := findParameterByProducerKeyInList("result", "producer", test.parameters)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.errorText)
		})
	}
}

func TestFindParameterByProducerKeyInList_PreservesOrderWithoutIterationMetadata(t *testing.T) {
	parameters := []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{
		{
			ParameterKey: "result",
			Type:         apiv2beta1.IOType_ITERATOR_OUTPUT,
			Value:        structpb.NewStringValue("first"),
			Producer:     &apiv2beta1.IOProducer{TaskName: "producer"},
		},
		{
			ParameterKey: "result",
			Type:         apiv2beta1.IOType_ITERATOR_OUTPUT,
			Value:        structpb.NewStringValue("second"),
			Producer:     &apiv2beta1.IOProducer{TaskName: "producer"},
		},
	}

	resolved, err := findParameterByProducerKeyInList("result", "producer", parameters)

	require.NoError(t, err)
	require.Len(t, resolved.GetValue().GetListValue().GetValues(), 2)
	assert.Equal(t, "first", resolved.GetValue().GetListValue().GetValues()[0].GetStringValue())
	assert.Equal(t, "second", resolved.GetValue().GetListValue().GetValues()[1].GetStringValue())
}

func iteratorParameter(
	key string,
	iteration int64,
	value string,
) *apiv2beta1.PipelineTask_InputOutputs_IOParameter {
	return &apiv2beta1.PipelineTask_InputOutputs_IOParameter{
		ParameterKey: key,
		Type:         apiv2beta1.IOType_ITERATOR_OUTPUT,
		Value:        structpb.NewStringValue(value),
		Producer: &apiv2beta1.IOProducer{
			TaskName:  "producer",
			Iteration: util.Int64Pointer(iteration),
		},
	}
}
