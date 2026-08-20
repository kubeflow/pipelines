// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	runmodel "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestParentRunLabels(t *testing.T) {
	labels := ParentRunLabels("parent-123", "trigger-task")
	assert.Equal(t, "parent-123", labels[LabelParentRunID])
	assert.Equal(t, "trigger-task", labels[LabelParentTaskName])
}

func TestFormatParentLabelsDescription(t *testing.T) {
	got := FormatParentLabelsDescription("run-a", "task-b")
	want := "pipelines.kubeflow.org/parent-run-id=run-a pipelines.kubeflow.org/parent-task-name=task-b"
	assert.Equal(t, want, got)
}

func TestIsTerminalRuntimeState(t *testing.T) {
	assert.True(t, isTerminalRuntimeState(runmodel.V2beta1RuntimeStateSUCCEEDED))
	assert.True(t, isTerminalRuntimeState(runmodel.V2beta1RuntimeStateFAILED))
	assert.True(t, isTerminalRuntimeState(runmodel.V2beta1RuntimeStateCANCELED))
	assert.True(t, isTerminalRuntimeState(runmodel.V2beta1RuntimeStateSKIPPED))
	assert.False(t, isTerminalRuntimeState(runmodel.V2beta1RuntimeStateRUNNING))
	assert.False(t, isTerminalRuntimeState(runmodel.V2beta1RuntimeStatePENDING))
}

func TestResolveTriggerInputParameterConstant(t *testing.T) {
	spec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
			RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue("hello"),
				},
			},
		},
	}
	v, err := resolveTriggerInputParameter(spec, nil, nil, 1)
	require.NoError(t, err)
	assert.Equal(t, "hello", v.GetStringValue())
}

func TestResolveTriggerInputParameterComponentInput(t *testing.T) {
	spec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
			ComponentInputParameter: "model_name",
		},
	}
	dagInputs := map[string]*structpb.Value{
		"model_name": structpb.NewStringValue("SASRec"),
	}
	v, err := resolveTriggerInputParameter(spec, dagInputs, nil, 1)
	require.NoError(t, err)
	assert.Equal(t, "SASRec", v.GetStringValue())
}

func TestStructpbValueToInterface(t *testing.T) {
	assert.Equal(t, "x", structpbValueToInterface(structpb.NewStringValue("x")))
	assert.Equal(t, true, structpbValueToInterface(structpb.NewBoolValue(true)))
	assert.Equal(t, float64(3), structpbValueToInterface(structpb.NewNumberValue(3)))
}

func TestTriggerPipelineLauncherOptionsValidate(t *testing.T) {
	err := (&TriggerPipelineLauncherOptions{}).validate()
	assert.Error(t, err)
	err = (&TriggerPipelineLauncherOptions{
		PipelineName: "p",
		RunID:        "r",
		ParentDagID:  1,
	}).validate()
	assert.NoError(t, err)
}
