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
	"fmt"
	"testing"

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

// The SDK compiles f-strings and str.format() arguments that reference pipeline channels into a
// constant with embedded placeholders plus a pipelinechannel--* task input for each channel.
func TestResolveInputParameter_ConstantWithEmbeddedPipelineChannels(t *testing.T) {
	parentTask := &apiv2.PipelineTask{TaskId: "parent-task", Name: "parent"}
	parentInputs := []*apiv2.PipelineTask_InputOutputs_IOParameter{
		{ParameterKey: "name", Value: structpb.NewStringValue("world")},
		{ParameterKey: "other", Value: structpb.NewStringValue("KFP")},
		{ParameterKey: "num", Value: structpb.NewNumberValue(3)},
	}
	opts := common.Options{
		ParentTask: parentTask,
		Run:        &apiv2.Run{RunId: "run-1"},
		Task: &pipelinespec.PipelineTaskSpec{Inputs: &pipelinespec.TaskInputsSpec{
			Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
				"pipelinechannel--name":  common.InputParamComponent("name"),
				"pipelinechannel--other": common.InputParamComponent("other"),
				"pipelinechannel--num":   common.InputParamComponent("num"),
			},
		}},
	}
	channel := func(name string) string { return "{{$.inputs.parameters['pipelinechannel--" + name + "']}}" }

	tests := []struct {
		name     string
		constant string
		want     *structpb.Value
	}{
		{"prefix text", "Hello " + channel("name"), structpb.NewStringValue("Hello world")},
		{"suffix text", channel("name") + ", again.", structpb.NewStringValue("world, again.")},
		{"two channels", channel("name") + "-" + channel("other"), structpb.NewStringValue("world-KFP")},
		{"number inside text", "n=" + channel("num"), structpb.NewStringValue("n=3")},
		{"non-channel placeholder kept", "{{$.inputs.parameters['x']}} " + channel("name"), structpb.NewStringValue("{{$.inputs.parameters['x']}} world")},
		{"single channel", channel("name"), structpb.NewStringValue("world")},
		{"single channel keeps type", channel("num"), structpb.NewNumberValue(3)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolved, _, err := ResolveInputParameter(opts, common.InputParamConstant(test.constant), parentInputs)
			require.NoError(t, err)
			assert.Equal(t, test.want.AsInterface(), resolved.GetValue().AsInterface())
		})
	}

	_, _, err := ResolveInputParameter(opts, common.InputParamConstant("Hello "+channel("missing")), parentInputs)
	require.ErrorContains(t, err, "pipelinechannel--missing")
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

// A task inside a ParallelFor body, including one compiled into an exit
// lifecycle hook, resolves the loop item from the loop task's inputs. Those
// inputs hold one entry per iteration under the same parameter key, so the
// iteration index is what disambiguates them. Resolving without it would return
// the first iteration's value for every iteration.
func TestResolveParameterComponentInputParameter_SelectsMatchingIteration(t *testing.T) {
	const itemKey = "pipelinechannel--loop-item-param-1"
	loopTask := &apiv2.PipelineTask{
		TaskId: "loop-dag-1",
		Name:   "for-loop-1",
		Type:   apiv2.PipelineTask_LOOP,
		Inputs: &apiv2.PipelineTask_InputOutputs{
			Parameters: []*apiv2.PipelineTask_InputOutputs_IOParameter{
				{
					ParameterKey: itemKey,
					Type:         apiv2.IOType_ITERATOR_INPUT,
					Value:        structpb.NewStringValue("A"),
					Producer: &apiv2.IOProducer{
						TaskName:  "for-loop-1",
						Iteration: util.Int64Pointer(0),
					},
				},
				{
					ParameterKey: itemKey,
					Type:         apiv2.IOType_ITERATOR_INPUT,
					Value:        structpb.NewStringValue("B"),
					Producer: &apiv2.IOProducer{
						TaskName:  "for-loop-1",
						Iteration: util.Int64Pointer(1),
					},
				},
			},
		},
	}

	for _, test := range []struct {
		iterationIndex int
		want           string
	}{
		{iterationIndex: 0, want: "A"},
		{iterationIndex: 1, want: "B"},
	} {
		opts := common.Options{
			ParentTask:     loopTask,
			Run:            &apiv2.Run{RunId: "run-1"},
			IterationIndex: test.iterationIndex,
		}
		resolved, err := resolveParameterComponentInputParameter(
			opts,
			common.InputParamComponent(itemKey),
			loopTask.GetInputs().GetParameters(),
		)
		require.NoError(t, err)
		require.NotNil(t, resolved)
		assert.Equal(t, test.want, resolved.GetValue().GetStringValue(),
			"iteration %d resolved the wrong loop item", test.iterationIndex)
		require.NotNil(t, resolved.GetProducer().Iteration)
		assert.Equal(t, int64(test.iterationIndex), *resolved.GetProducer().Iteration)
	}
}

// Loop children are keyed by task name, parent DAG task id and iteration
// index, so a consumer in the same loop body must carry the iteration index to
// find its sibling producer.
func TestResolveTaskOutputParameter_RequiresIterationIndexForLoopSibling(t *testing.T) {
	parentTaskID := "loop-dag-1"
	loopTask := &apiv2.PipelineTask{
		TaskId: parentTaskID,
		Name:   "for-loop-1",
		Type:   apiv2.PipelineTask_LOOP,
	}
	var producers []*apiv2.PipelineTask
	for index, value := range []string{"A-out", "B-out"} {
		producers = append(producers, &apiv2.PipelineTask{
			TaskId:       fmt.Sprintf("produce-%d", index),
			Name:         "produce",
			ParentTaskId: util.StringPointer(parentTaskID),
			Type:         apiv2.PipelineTask_RUNTIME,
			TypeAttributes: &apiv2.PipelineTask_TypeAttributes{
				IterationIndex: util.Int64Pointer(int64(index)),
			},
			Outputs: &apiv2.PipelineTask_InputOutputs{
				Parameters: []*apiv2.PipelineTask_InputOutputs_IOParameter{{
					ParameterKey: "Output",
					Value:        structpb.NewStringValue(value),
					Producer:     &apiv2.IOProducer{TaskName: "produce"},
				}},
			},
		})
	}
	paramSpec := common.InputParamTaskOutput("produce", "Output")

	for _, test := range []struct {
		iterationIndex int
		want           string
	}{
		{iterationIndex: 0, want: "A-out"},
		{iterationIndex: 1, want: "B-out"},
	} {
		opts := common.Options{
			ParentTask:     loopTask,
			Run:            &apiv2.Run{RunId: "run-1", Tasks: producers},
			IterationIndex: test.iterationIndex,
		}
		resolved, err := resolveTaskOutputParameter(opts, paramSpec)
		require.NoError(t, err)
		require.NotNil(t, resolved)
		assert.Equal(t, test.want, resolved.GetValue().GetStringValue())
	}

	// Without an iteration index the lookup key loses its iteration suffix and
	// no sibling producer matches, even though both producers succeeded.
	opts := common.Options{
		ParentTask:     loopTask,
		Run:            &apiv2.Run{RunId: "run-1", Tasks: producers},
		IterationIndex: -1,
	}
	_, err := resolveTaskOutputParameter(opts, paramSpec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "produce_"+parentTaskID+" not found")
}
