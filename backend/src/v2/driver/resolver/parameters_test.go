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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestResolveInputParameter_RuntimePipelineJobTimes(t *testing.T) {
	createTime := "2026-08-10T18:10:00Z"
	scheduleTime := "2026-08-10T18:40:00Z"
	parentTask := &apiv2.PipelineTask{
		TaskId: "parent-task",
		Name:   "parent",
	}
	opts := common.Options{
		ParentTask:                 parentTask,
		Run:                        &apiv2.Run{RunId: "run-1"},
		PipelineJobCreateTimeUTC:   createTime,
		PipelineJobScheduleTimeUTC: scheduleTime,
	}

	tests := []struct {
		name        string
		placeholder string
		want        string
	}{
		{
			name:        "create time placeholder",
			placeholder: "{{$.pipeline_job_create_time_utc}}",
			want:        createTime,
		},
		{
			name:        "schedule time placeholder",
			placeholder: "{{$.pipeline_job_schedule_time_utc}}",
			want:        scheduleTime,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolved, ioType, err := ResolveInputParameter(opts, common.InputParamConstant(test.placeholder), nil)
			require.NoError(t, err)
			require.NotNil(t, resolved)
			assert.Equal(t, apiv2.IOType_RUNTIME_VALUE_INPUT, ioType)
			assert.Equal(t, test.want, resolved.GetValue().GetStringValue())
			assert.Equal(t, parentTask.GetName(), resolved.GetProducer().GetTaskName())
		})
	}
}

func TestResolveTaskOutputParameter_FindsIterationScopedNonRuntimeProducer(t *testing.T) {
	tests := []struct {
		name         string
		producerType apiv2.PipelineTask_TaskType
	}{
		{
			name:         "dag producer",
			producerType: apiv2.PipelineTask_DAG,
		},
		{
			name:         "importer producer",
			producerType: apiv2.PipelineTask_IMPORTER,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parentTaskID := "loop-parent"
			parentTask := &apiv2.PipelineTask{
				TaskId: parentTaskID,
				Name:   "loop-body",
				Type:   apiv2.PipelineTask_LOOP,
			}

			outputValue := structpb.NewStringValue("resolved")
			producerTask := &apiv2.PipelineTask{
				TaskId:       "producer-task",
				Name:         "produce",
				ParentTaskId: util.StringPointer(parentTaskID),
				Type:         test.producerType,
				TypeAttributes: &apiv2.PipelineTask_TypeAttributes{
					IterationIndex: util.Int64Pointer(0),
				},
				Outputs: &apiv2.PipelineTask_InputOutputs{
					Parameters: []*apiv2.PipelineTask_InputOutputs_IOParameter{
						{
							ParameterKey: "result",
							Value:        outputValue,
							Producer: &apiv2.IOProducer{
								TaskName: "produce",
							},
						},
					},
				},
			}

			opts := common.Options{
				ParentTask:     parentTask,
				Run:            &apiv2.Run{Tasks: []*apiv2.PipelineTask{producerTask}},
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

func TestResolveTaskOutputParameter_CollectionBoundary(t *testing.T) {
	for _, producerType := range []apiv2.PipelineTask_TaskType{
		apiv2.PipelineTask_RUNTIME, apiv2.PipelineTask_DAG, apiv2.PipelineTask_LOOP,
	} {
		for _, value := range []*structpb.Value{
			structpb.NewNumberValue(605), ToListValue([]*structpb.Value{structpb.NewNumberValue(605)}),
		} {
			t.Run(producerType.String()+"/"+value.String(), func(t *testing.T) {
				parent := &apiv2.PipelineTask{TaskId: "outer", Name: "outer", Type: apiv2.PipelineTask_LOOP}
				output := iteratorParameter("result", 0, "")
				output.Value = value
				producer := &apiv2.PipelineTask{
					TaskId: "producer", Name: "produce", Type: producerType,
					ParentTaskId:   util.StringPointer("outer"),
					TypeAttributes: &apiv2.PipelineTask_TypeAttributes{IterationIndex: util.Int64Pointer(0)},
					Outputs:        &apiv2.PipelineTask_InputOutputs{Parameters: []*apiv2.PipelineTask_InputOutputs_IOParameter{output}},
				}
				resolved, ioType, err := ResolveInputParameter(common.Options{
					ParentTask: parent, IterationIndex: 0,
					Run: &apiv2.Run{Tasks: []*apiv2.PipelineTask{producer}},
				}, common.InputParamTaskOutput("produce", "result"), nil)
				require.NoError(t, err)
				if producerType != apiv2.PipelineTask_RUNTIME {
					assert.Equal(t, apiv2.IOType_COLLECTED_INPUTS, ioType)
					require.Len(t, resolved.GetValue().GetListValue().GetValues(), 1)
					assert.Equal(t, value, resolved.GetValue().GetListValue().GetValues()[0])
				} else {
					assert.Equal(t, apiv2.IOType_TASK_OUTPUT_INPUT, ioType)
					assert.Equal(t, value, resolved.GetValue())
				}
			})
		}
	}
}

func TestResolveTaskFinalStatus_FindsIterationScopedProducer(t *testing.T) {
	parentTaskID := "loop-parent"
	parentTask := &apiv2.PipelineTask{
		TaskId: parentTaskID,
		Name:   "loop-body",
		Type:   apiv2.PipelineTask_LOOP,
	}

	producerTask := &apiv2.PipelineTask{
		TaskId:       "producer-task",
		Name:         "produce",
		ParentTaskId: util.StringPointer(parentTaskID),
		Type:         apiv2.PipelineTask_RUNTIME,
		State:        apiv2.PipelineTask_FAILED,
		TypeAttributes: &apiv2.PipelineTask_TypeAttributes{
			IterationIndex: util.Int64Pointer(1),
		},
		StatusMetadata: &apiv2.PipelineTask_StatusMetadata{
			Message: "boom",
		},
	}

	opts := common.Options{
		ParentTask:     parentTask,
		Run:            &apiv2.Run{Tasks: []*apiv2.PipelineTask{producerTask}},
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
	parentTask := &apiv2.PipelineTask{TaskId: parentTaskID, Name: "parent"}
	producerTask := &apiv2.PipelineTask{
		TaskId:       "producer-task",
		Name:         "produce",
		ParentTaskId: util.StringPointer(parentTaskID),
		Type:         apiv2.PipelineTask_RUNTIME,
		State:        apiv2.PipelineTask_SUCCEEDED,
	}
	opts := common.Options{
		ParentTask:     parentTask,
		Run:            &apiv2.Run{Tasks: []*apiv2.PipelineTask{producerTask}},
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

	parentTask := &apiv2.PipelineTask{
		TaskId: "parent-task",
		Inputs: &apiv2.PipelineTask_InputOutputs{
			Parameters: []*apiv2.PipelineTask_InputOutputs_IOParameter{
				{
					ParameterKey: "item",
					Value:        inputValue,
					Producer:     &apiv2.IOProducer{TaskName: "upstream"},
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
	parentTask := &apiv2.PipelineTask{
		TaskId: "parent-task",
		Inputs: &apiv2.PipelineTask_InputOutputs{
			Parameters: []*apiv2.PipelineTask_InputOutputs_IOParameter{
				{
					ParameterKey: "mode",
					Value:        structpb.NewStringValue("invalid"),
					Producer:     &apiv2.IOProducer{TaskName: "upstream"},
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
		ParentTask: &apiv2.PipelineTask{
			TaskId: "parent-task",
			Inputs: &apiv2.PipelineTask_InputOutputs{},
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
		ParentTask: &apiv2.PipelineTask{
			TaskId: "parent-task",
			Inputs: &apiv2.PipelineTask_InputOutputs{
				Parameters: []*apiv2.PipelineTask_InputOutputs_IOParameter{
					{
						ParameterKey: "loop-item-user",
						Value:        structpb.NewStringValue("value"),
						Producer:     &apiv2.IOProducer{TaskName: "upstream"},
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
		ParentTask: &apiv2.PipelineTask{
			TaskId: "parent-task",
			Inputs: &apiv2.PipelineTask_InputOutputs{
				Parameters: []*apiv2.PipelineTask_InputOutputs_IOParameter{
					{
						ParameterKey: "pipelinechannel--loop-item-param-3",
						Value:        structpb.NewStringValue("outer-0"),
						Producer: &apiv2.IOProducer{
							TaskName:  "outer-loop",
							Iteration: util.Int64Pointer(0),
						},
					},
					{
						ParameterKey: "pipelinechannel--loop-item-param-5",
						Value:        structpb.NewStringValue("inner-0"),
						Producer: &apiv2.IOProducer{
							TaskName:  "inner-loop",
							Iteration: util.Int64Pointer(0),
						},
					},
					{
						ParameterKey: "pipelinechannel--loop-item-param-5",
						Value:        structpb.NewStringValue("inner-1"),
						Producer: &apiv2.IOProducer{
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
	parentTask := &apiv2.PipelineTask{TaskId: parentTaskID, Name: "dag"}
	producerTask := &apiv2.PipelineTask{
		TaskId:       "loop-task",
		Name:         "loop",
		ParentTaskId: util.StringPointer(parentTaskID),
		Type:         apiv2.PipelineTask_LOOP,
		TypeAttributes: &apiv2.PipelineTask_TypeAttributes{
			IterationCount: util.Int64Pointer(0),
		},
	}
	opts := common.Options{
		ParentTask:     parentTask,
		Run:            &apiv2.Run{Tasks: []*apiv2.PipelineTask{producerTask}},
		IterationIndex: -1,
	}

	resolved, err := resolveTaskOutputParameter(opts, common.InputParamTaskOutput("loop", "result"))
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, apiv2.IOType_COLLECTED_INPUTS, resolved.GetType())
	assert.Empty(t, resolved.GetValue().GetListValue().GetValues())
}

func TestFindParameterByProducerKeyInList_OrdersIteratorOutputs(t *testing.T) {
	parameters := []*apiv2.PipelineTask_InputOutputs_IOParameter{
		iteratorParameter("result", 2, "two"),
		iteratorParameter("result", 0, "zero"),
		iteratorParameter("result", 1, "one"),
	}

	resolved, err := findParameterByProducerKeyInList("result", "producer", parameters, true)

	require.NoError(t, err)
	assert.Equal(t, apiv2.IOType_COLLECTED_INPUTS, resolved.GetType())
	require.Len(t, resolved.GetValue().GetListValue().GetValues(), 3)
	assert.Equal(t, "zero", resolved.GetValue().GetListValue().GetValues()[0].GetStringValue())
	assert.Equal(t, "one", resolved.GetValue().GetListValue().GetValues()[1].GetStringValue())
	assert.Equal(t, "two", resolved.GetValue().GetListValue().GetValues()[2].GetStringValue())
}

func TestFindParameterByProducerKeyInList_WrapsSingletonIteratorOutput(t *testing.T) {
	resolved, err := findParameterByProducerKeyInList(
		"result",
		"producer",
		[]*apiv2.PipelineTask_InputOutputs_IOParameter{
			iteratorParameter("result", 0, "only"),
		},
		true,
	)

	require.NoError(t, err)
	assert.Equal(t, apiv2.IOType_COLLECTED_INPUTS, resolved.GetType())
	require.Len(t, resolved.GetValue().GetListValue().GetValues(), 1)
	assert.Equal(t, "only", resolved.GetValue().GetListValue().GetValues()[0].GetStringValue())
}

func TestFindParameterByProducerKeyInList_RejectsInvalidIteratorMetadata(t *testing.T) {
	tests := []struct {
		name       string
		parameters []*apiv2.PipelineTask_InputOutputs_IOParameter
		errorText  string
	}{
		{
			name: "duplicate iteration",
			parameters: []*apiv2.PipelineTask_InputOutputs_IOParameter{
				iteratorParameter("result", 1, "first"),
				iteratorParameter("result", 1, "second"),
			},
			errorText: "duplicate iteration index 1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := findParameterByProducerKeyInList("result", "producer", test.parameters, true)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.errorText)
		})
	}
}

func TestFindParameterByProducerKeyInList_PreservesOrderWithoutIterationMetadata(t *testing.T) {
	parameters := []*apiv2.PipelineTask_InputOutputs_IOParameter{
		{
			ParameterKey: "result",
			Type:         apiv2.IOType_ITERATOR_OUTPUT,
			Value:        structpb.NewStringValue("first"),
			Producer:     &apiv2.IOProducer{TaskName: "producer"},
		},
		{
			ParameterKey: "result",
			Type:         apiv2.IOType_ITERATOR_OUTPUT,
			Value:        structpb.NewStringValue("second"),
			Producer:     &apiv2.IOProducer{TaskName: "producer"},
		},
	}

	resolved, err := findParameterByProducerKeyInList("result", "producer", parameters, true)

	require.NoError(t, err)
	require.Len(t, resolved.GetValue().GetListValue().GetValues(), 2)
	assert.Equal(t, "first", resolved.GetValue().GetListValue().GetValues()[0].GetStringValue())
	assert.Equal(t, "second", resolved.GetValue().GetListValue().GetValues()[1].GetStringValue())
}

func iteratorParameter(
	key string,
	iteration int64,
	value string,
) *apiv2.PipelineTask_InputOutputs_IOParameter {
	return &apiv2.PipelineTask_InputOutputs_IOParameter{
		ParameterKey: key,
		Type:         apiv2.IOType_ITERATOR_OUTPUT,
		Value:        structpb.NewStringValue(value),
		Producer: &apiv2.IOProducer{
			TaskName:  "producer",
			Iteration: util.Int64Pointer(iteration),
		},
	}
}

func TestResolveTaskFinalStatus_ExitHandlerResolutionPath(t *testing.T) {
	rootTask := &apiv2.PipelineTask{TaskId: "root", Name: "root"}

	// 1. Order 1: dag-1 with child-1 (FAILED) and child-2 (SUCCEEDED) -> dag-1 is FAILED
	dag1 := &apiv2.PipelineTask{
		TaskId:         "dag-1-id",
		Name:           "dag-1",
		ParentTaskId:   util.StringPointer("root"),
		Type:           apiv2.PipelineTask_DAG,
		State:          apiv2.PipelineTask_FAILED,
		StatusMetadata: &apiv2.PipelineTask_StatusMetadata{Message: "child 1 failed"},
	}
	opts1 := common.Options{
		ParentTask:     rootTask,
		Run:            &apiv2.Run{Tasks: []*apiv2.PipelineTask{dag1}},
		RunName:        "pipeline-run",
		IterationIndex: -1,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler-1"},
			DependentTasks: []string{"dag-1"},
		},
	}
	spec1 := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "dag-1"},
		},
	}
	res1, err := resolveTaskFinalStatus(opts1, spec1)
	require.NoError(t, err)
	assert.Equal(t, "FAILED", res1.GetStructValue().GetFields()["state"].GetStringValue())
	assert.Equal(t, "dag-1", res1.GetStructValue().GetFields()["pipelineTaskName"].GetStringValue())
	assert.Equal(t, float64(codes.Unknown), res1.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())

	// 2. Order 2: dag-2 with child-a (SUCCEEDED) and child-b (FAILED) -> dag-2 is FAILED
	dag2 := &apiv2.PipelineTask{
		TaskId:         "dag-2-id",
		Name:           "dag-2",
		ParentTaskId:   util.StringPointer("root"),
		Type:           apiv2.PipelineTask_DAG,
		State:          apiv2.PipelineTask_FAILED,
		StatusMetadata: &apiv2.PipelineTask_StatusMetadata{Message: "child b failed"},
	}
	opts2 := common.Options{
		ParentTask:     rootTask,
		Run:            &apiv2.Run{Tasks: []*apiv2.PipelineTask{dag2}},
		RunName:        "pipeline-run",
		IterationIndex: -1,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler-2"},
			DependentTasks: []string{"dag-2"},
		},
	}
	spec2 := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "dag-2"},
		},
	}
	res2, err := resolveTaskFinalStatus(opts2, spec2)
	require.NoError(t, err)
	assert.Equal(t, "FAILED", res2.GetStructValue().GetFields()["state"].GetStringValue())
	assert.Equal(t, float64(codes.Unknown), res2.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())

	// 3. Sibling still running (deferred parent): dag-running is still RUNNING
	dagRunning := &apiv2.PipelineTask{
		TaskId:       "dag-running-id",
		Name:         "dag-running",
		ParentTaskId: util.StringPointer("root"),
		Type:         apiv2.PipelineTask_DAG,
		State:        apiv2.PipelineTask_RUNNING,
	}
	optsRunning := common.Options{
		ParentTask:     rootTask,
		Run:            &apiv2.Run{Tasks: []*apiv2.PipelineTask{dagRunning}},
		RunName:        "pipeline-run",
		IterationIndex: -1,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "interim-monitor"},
			DependentTasks: []string{"dag-running"},
		},
	}
	specRunning := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "dag-running"},
		},
	}
	_, err = resolveTaskFinalStatus(optsRunning, specRunning)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "producer task state RUNNING is not terminal")

	// 4. Nested exit handlers: outer DAG -> inner DAG
	outerDag := &apiv2.PipelineTask{
		TaskId:       "outer-dag-id",
		Name:         "outer-dag",
		ParentTaskId: util.StringPointer("root"),
		Type:         apiv2.PipelineTask_DAG,
		State:        apiv2.PipelineTask_FAILED,
	}
	innerDag := &apiv2.PipelineTask{
		TaskId:       "inner-dag-id",
		Name:         "inner-dag",
		ParentTaskId: util.StringPointer("outer-dag-id"),
		Type:         apiv2.PipelineTask_DAG,
		State:        apiv2.PipelineTask_FAILED,
	}
	allNestedTasks := []*apiv2.PipelineTask{outerDag, innerDag}

	// Inner exit handler resolves inner-dag (parent is outer-dag)
	optsInner := common.Options{
		ParentTask:     outerDag,
		Run:            &apiv2.Run{Tasks: allNestedTasks},
		RunName:        "pipeline-run",
		IterationIndex: -1,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "inner-exit-handler"},
			DependentTasks: []string{"inner-dag"},
		},
	}
	specInner := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "inner-dag"},
		},
	}
	resInner, err := resolveTaskFinalStatus(optsInner, specInner)
	require.NoError(t, err)
	assert.Equal(t, "FAILED", resInner.GetStructValue().GetFields()["state"].GetStringValue())
	assert.Equal(t, "inner-dag", resInner.GetStructValue().GetFields()["pipelineTaskName"].GetStringValue())
	assert.Equal(t, float64(codes.Unknown), resInner.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())

	// Outer exit handler resolves outer-dag (parent is root)
	optsOuter := common.Options{
		ParentTask:     rootTask,
		Run:            &apiv2.Run{Tasks: allNestedTasks},
		RunName:        "pipeline-run",
		IterationIndex: -1,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "outer-exit-handler"},
			DependentTasks: []string{"outer-dag"},
		},
	}
	specOuter := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "outer-dag"},
		},
	}
	resOuter, err := resolveTaskFinalStatus(optsOuter, specOuter)
	require.NoError(t, err)
	assert.Equal(t, "FAILED", resOuter.GetStructValue().GetFields()["state"].GetStringValue())
	assert.Equal(t, "outer-dag", resOuter.GetStructValue().GetFields()["pipelineTaskName"].GetStringValue())
	assert.Equal(t, float64(codes.Unknown), resOuter.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())
}

func TestResolveTaskFinalStatus_BoundedWait_SucceedsWhenProducerBecomesTerminal(t *testing.T) {
	parent := &apiv2.PipelineTask{TaskId: "parent-1", Name: "root"}
	producer := &apiv2.PipelineTask{
		TaskId:       "task-1",
		Name:         "producer",
		Type:         apiv2.PipelineTask_RUNTIME,
		ParentTaskId: util.StringPointer("parent-1"),
		State:        apiv2.PipelineTask_RUNNING,
	}

	callCount := 0
	opts := common.Options{
		ParentTask:              parent,
		Run:                     &apiv2.Run{RunId: "run-1", Tasks: []*apiv2.PipelineTask{producer}},
		RunName:                 "run-1",
		IterationIndex:          -1,
		FinalStatusPollInterval: 10 * time.Millisecond,
		FinalStatusTimeout:      1 * time.Second,
		RunRefresher: func(ctx context.Context) (*apiv2.Run, error) {
			callCount++
			terminalState := apiv2.PipelineTask_RUNNING
			if callCount >= 2 {
				terminalState = apiv2.PipelineTask_SUCCEEDED
			}
			refreshedProducer := &apiv2.PipelineTask{
				TaskId:       "task-1",
				Name:         "producer",
				Type:         apiv2.PipelineTask_RUNTIME,
				ParentTaskId: util.StringPointer("parent-1"),
				State:        terminalState,
			}
			return &apiv2.Run{RunId: "run-1", Tasks: []*apiv2.PipelineTask{refreshedProducer}}, nil
		},
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"producer"},
		},
	}

	spec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "producer"},
		},
	}

	res, err := resolveTaskFinalStatus(opts, spec)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "SUCCEEDED", res.GetStructValue().GetFields()["state"].GetStringValue())
	assert.GreaterOrEqual(t, callCount, 2)
}

func TestResolveTaskFinalStatus_BoundedWait_DeadlineExceeded(t *testing.T) {
	parent := &apiv2.PipelineTask{TaskId: "parent-1", Name: "root"}
	producer := &apiv2.PipelineTask{
		TaskId:       "task-1",
		Name:         "producer",
		Type:         apiv2.PipelineTask_RUNTIME,
		ParentTaskId: util.StringPointer("parent-1"),
		State:        apiv2.PipelineTask_RUNNING,
	}

	opts := common.Options{
		ParentTask:              parent,
		Run:                     &apiv2.Run{RunId: "run-1", Tasks: []*apiv2.PipelineTask{producer}},
		RunName:                 "run-1",
		IterationIndex:          -1,
		FinalStatusPollInterval: 10 * time.Millisecond,
		FinalStatusTimeout:      40 * time.Millisecond,
		RunRefresher: func(ctx context.Context) (*apiv2.Run, error) {
			return &apiv2.Run{RunId: "run-1", Tasks: []*apiv2.PipelineTask{producer}}, nil
		},
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"producer"},
		},
	}

	spec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "producer"},
		},
	}

	_, err := resolveTaskFinalStatus(opts, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out waiting for producer task")
}

func TestResolveTaskFinalStatus_BoundedWait_DeadlineExceededWithFailedChild_FailsClosedToFailed(t *testing.T) {
	parent := &apiv2.PipelineTask{TaskId: "parent-1", Name: "root"}
	producerDAG := &apiv2.PipelineTask{
		TaskId:       "dag-1",
		Name:         "producer-dag",
		Type:         apiv2.PipelineTask_DAG,
		ParentTaskId: util.StringPointer("parent-1"),
		State:        apiv2.PipelineTask_RUNNING,
	}
	failedChild := &apiv2.PipelineTask{
		TaskId:         "child-failed",
		Name:           "child-failed",
		Type:           apiv2.PipelineTask_RUNTIME,
		ParentTaskId:   util.StringPointer("dag-1"),
		State:          apiv2.PipelineTask_FAILED,
		StatusMetadata: &apiv2.PipelineTask_StatusMetadata{Message: "child OOMKilled"},
	}
	runningChild := &apiv2.PipelineTask{
		TaskId:       "child-running",
		Name:         "child-running",
		Type:         apiv2.PipelineTask_RUNTIME,
		ParentTaskId: util.StringPointer("dag-1"),
		State:        apiv2.PipelineTask_RUNNING,
	}

	tasks := []*apiv2.PipelineTask{producerDAG, failedChild, runningChild}

	opts := common.Options{
		ParentTask:              parent,
		Run:                     &apiv2.Run{RunId: "run-1", Tasks: tasks},
		RunName:                 "run-1",
		IterationIndex:          -1,
		FinalStatusPollInterval: 10 * time.Millisecond,
		FinalStatusTimeout:      40 * time.Millisecond,
		RunRefresher: func(ctx context.Context) (*apiv2.Run, error) {
			return &apiv2.Run{RunId: "run-1", Tasks: tasks}, nil
		},
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"producer-dag"},
		},
	}

	spec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "producer-dag"},
		},
	}

	res, err := resolveTaskFinalStatus(opts, spec)
	require.NoError(t, err, "Must fail closed to FAILED on deadline timeout when a child is failed")
	assert.Equal(t, "FAILED", res.GetStructValue().GetFields()["state"].GetStringValue())
	assert.Equal(t, "producer-dag", res.GetStructValue().GetFields()["pipelineTaskName"].GetStringValue())
	assert.Equal(t, "child OOMKilled", res.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["message"].GetStringValue())
}

func TestResolveTaskFinalStatus_BoundedWait_ContextCanceled(t *testing.T) {
	parent := &apiv2.PipelineTask{TaskId: "parent-1", Name: "root"}
	producer := &apiv2.PipelineTask{
		TaskId:       "task-1",
		Name:         "producer",
		Type:         apiv2.PipelineTask_RUNTIME,
		ParentTaskId: util.StringPointer("parent-1"),
		State:        apiv2.PipelineTask_RUNNING,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel context

	opts := common.Options{
		ParentTask:              parent,
		Run:                     &apiv2.Run{RunId: "run-1", Tasks: []*apiv2.PipelineTask{producer}},
		RunName:                 "run-1",
		IterationIndex:          -1,
		FinalStatusPollInterval: 10 * time.Millisecond,
		FinalStatusTimeout:      5 * time.Second,
		RunRefresher: func(ctx context.Context) (*apiv2.Run, error) {
			return &apiv2.Run{RunId: "run-1", Tasks: []*apiv2.PipelineTask{producer}}, nil
		},
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"producer"},
		},
	}

	spec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "producer"},
		},
	}

	_, err := resolveTaskFinalStatusWithContext(ctx, &opts, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "canceled")
}

func TestResolveTaskFinalStatus_BoundedWait_RefresherErrorFailsClosed(t *testing.T) {
	parent := &apiv2.PipelineTask{TaskId: "parent-1", Name: "root"}
	producer := &apiv2.PipelineTask{
		TaskId:       "task-1",
		Name:         "producer",
		Type:         apiv2.PipelineTask_RUNTIME,
		ParentTaskId: util.StringPointer("parent-1"),
		State:        apiv2.PipelineTask_RUNNING,
	}

	opts := common.Options{
		ParentTask:              parent,
		Run:                     &apiv2.Run{RunId: "run-1", Tasks: []*apiv2.PipelineTask{producer}},
		RunName:                 "run-1",
		IterationIndex:          -1,
		FinalStatusPollInterval: 10 * time.Millisecond,
		FinalStatusTimeout:      1 * time.Second,
		RunRefresher: func(ctx context.Context) (*apiv2.Run, error) {
			return nil, fmt.Errorf("network disconnect")
		},
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"producer"},
		},
	}

	spec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "producer"},
		},
	}

	_, err := resolveTaskFinalStatus(opts, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network disconnect")
}

func TestResolveTaskFinalStatus_InconsistentParentSucceededWithFailedChild_FailsClosedToFailed(t *testing.T) {
	root := &apiv2.PipelineTask{TaskId: "root", Name: "root"}
	parentDAG := &apiv2.PipelineTask{
		TaskId:       "dag-1",
		Name:         "main-dag",
		Type:         apiv2.PipelineTask_DAG,
		ParentTaskId: util.StringPointer("root"),
		State:        apiv2.PipelineTask_SUCCEEDED, // stale fast-path in DB!
	}
	childTask := &apiv2.PipelineTask{
		TaskId:         "child-1",
		Name:           "child-task",
		Type:           apiv2.PipelineTask_RUNTIME,
		ParentTaskId:   util.StringPointer("dag-1"),
		State:          apiv2.PipelineTask_FAILED,
		StatusMetadata: &apiv2.PipelineTask_StatusMetadata{Message: "child task failed with OOMKilled"},
	}

	opts := common.Options{
		ParentTask: root,
		Run: &apiv2.Run{
			RunId: "run-1",
			Tasks: []*apiv2.PipelineTask{parentDAG, childTask},
		},
		RunName:        "pipeline-run",
		IterationIndex: -1,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"main-dag"},
		},
	}
	spec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "main-dag"},
		},
	}

	// Must NOT return SUCCEEDED even though parentDAG was SUCCEEDED; must fail closed to FAILED.
	res, err := resolveTaskFinalStatus(opts, spec)
	require.NoError(t, err)
	assert.Equal(t, "FAILED", res.GetStructValue().GetFields()["state"].GetStringValue(), "Must never return SUCCEEDED when a child task failed")
	assert.Equal(t, "main-dag", res.GetStructValue().GetFields()["pipelineTaskName"].GetStringValue())
	assert.Equal(t, float64(codes.Unknown), res.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())
	assert.Equal(t, "child task failed with OOMKilled", res.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["message"].GetStringValue())
}

func TestResolveTaskFinalStatus_InconsistentParentSucceededWithFailedChild_RefreshesAuthoritatively(t *testing.T) {
	root := &apiv2.PipelineTask{TaskId: "root", Name: "root"}
	parentDAG := &apiv2.PipelineTask{
		TaskId:       "dag-1",
		Name:         "main-dag",
		Type:         apiv2.PipelineTask_DAG,
		ParentTaskId: util.StringPointer("root"),
		State:        apiv2.PipelineTask_SUCCEEDED, // stale fast-path
	}
	childTask := &apiv2.PipelineTask{
		TaskId:         "child-1",
		Name:           "child-task",
		Type:           apiv2.PipelineTask_RUNTIME,
		ParentTaskId:   util.StringPointer("dag-1"),
		State:          apiv2.PipelineTask_FAILED,
		StatusMetadata: &apiv2.PipelineTask_StatusMetadata{Message: "child failed in argo"},
	}

	refreshCallCount := 0
	opts := common.Options{
		ParentTask: root,
		Run: &apiv2.Run{
			RunId: "run-1",
			Tasks: []*apiv2.PipelineTask{parentDAG, childTask},
		},
		RunName:                 "pipeline-run",
		IterationIndex:          -1,
		FinalStatusPollInterval: 10 * time.Millisecond,
		FinalStatusTimeout:      2 * time.Second,
		RunRefresher: func(ctx context.Context) (*apiv2.Run, error) {
			refreshCallCount++
			reconciledDAG := &apiv2.PipelineTask{
				TaskId:         "dag-1",
				Name:           "main-dag",
				Type:           apiv2.PipelineTask_DAG,
				ParentTaskId:   util.StringPointer("root"),
				State:          apiv2.PipelineTask_FAILED,
				StatusMetadata: &apiv2.PipelineTask_StatusMetadata{Message: "reconciled child failure from argo"},
			}
			return &apiv2.Run{
				RunId: "run-1",
				Tasks: []*apiv2.PipelineTask{reconciledDAG, childTask},
			}, nil
		},
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"main-dag"},
		},
	}
	spec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "main-dag"},
		},
	}

	res, err := resolveTaskFinalStatus(opts, spec)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, refreshCallCount, 1, "Must call RunRefresher to obtain authoritative synchronized status")
	assert.Equal(t, "FAILED", res.GetStructValue().GetFields()["state"].GetStringValue())
	assert.Equal(t, float64(codes.Unknown), res.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["code"].GetNumberValue())
	assert.Equal(t, "reconciled child failure from argo", res.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["message"].GetStringValue())
}

func TestResolveTaskFinalStatus_ParentSucceededWithRunningChild_WaitsUntilChildTerminal(t *testing.T) {
	root := &apiv2.PipelineTask{TaskId: "root", Name: "root"}
	parentDAG := &apiv2.PipelineTask{
		TaskId:       "dag-1",
		Name:         "main-dag",
		Type:         apiv2.PipelineTask_DAG,
		ParentTaskId: util.StringPointer("root"),
		State:        apiv2.PipelineTask_SUCCEEDED, // prematurely marked SUCCEEDED
	}
	childRunning := &apiv2.PipelineTask{
		TaskId:       "child-1",
		Name:         "child-task",
		Type:         apiv2.PipelineTask_RUNTIME,
		ParentTaskId: util.StringPointer("dag-1"),
		State:        apiv2.PipelineTask_RUNNING, // still running!
	}

	calls := 0
	opts := common.Options{
		ParentTask: root,
		Run: &apiv2.Run{
			RunId: "run-1",
			Tasks: []*apiv2.PipelineTask{parentDAG, childRunning},
		},
		RunName:                 "pipeline-run",
		IterationIndex:          -1,
		FinalStatusPollInterval: 10 * time.Millisecond,
		FinalStatusTimeout:      2 * time.Second,
		RunRefresher: func(ctx context.Context) (*apiv2.Run, error) {
			calls++
			if calls < 2 {
				return &apiv2.Run{RunId: "run-1", Tasks: []*apiv2.PipelineTask{parentDAG, childRunning}}, nil
			}
			terminalChild := &apiv2.PipelineTask{
				TaskId:         "child-1",
				Name:           "child-task",
				Type:           apiv2.PipelineTask_RUNTIME,
				ParentTaskId:   util.StringPointer("dag-1"),
				State:          apiv2.PipelineTask_FAILED,
				StatusMetadata: &apiv2.PipelineTask_StatusMetadata{Message: "child completed with error"},
			}
			reconciledParent := &apiv2.PipelineTask{
				TaskId:         "dag-1",
				Name:           "main-dag",
				Type:           apiv2.PipelineTask_DAG,
				ParentTaskId:   util.StringPointer("root"),
				State:          apiv2.PipelineTask_FAILED,
				StatusMetadata: &apiv2.PipelineTask_StatusMetadata{Message: "child completed with error"},
			}
			return &apiv2.Run{RunId: "run-1", Tasks: []*apiv2.PipelineTask{reconciledParent, terminalChild}}, nil
		},
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "exit-handler"},
			DependentTasks: []string{"main-dag"},
		},
	}
	spec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{
			TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "main-dag"},
		},
	}

	res, err := resolveTaskFinalStatus(opts, spec)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, calls, 2, "Must wait until running child task completes before resolving final status")
	assert.Equal(t, "FAILED", res.GetStructValue().GetFields()["state"].GetStringValue())
	assert.Equal(t, "child completed with error", res.GetStructValue().GetFields()["error"].GetStructValue().GetFields()["message"].GetStringValue())
}
