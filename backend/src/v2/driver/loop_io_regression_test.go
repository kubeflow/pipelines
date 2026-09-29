// Copyright 2026 The Kubeflow Authors
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

package driver

import (
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestNestedLoopCollectedParametersFixture(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{},
		"../../../../test_data/sdk_compiled_pipelines/valid/parameters_complex.yaml")
	_, outer := tc.RunDagDriver("for-loop-2", tc.RootTask)
	for outerIndex := int64(0); outerIndex < 3; outerIndex++ {
		innerExecution, inner := tc.RunDagDriver("for-loop-4", outer, outerIndex)
		require.Equal(t, apiv2beta1.PipelineTask_LOOP, inner.GetType())
		require.Equal(t, outerIndex, inner.GetTypeAttributes().GetIterationIndex())
		require.Equal(t, 3, *innerExecution.IterationCount)
		for innerIndex := int64(0); innerIndex < 3; innerIndex++ {
			execution, _ := tc.RunContainerDriver("double-2", inner, util.Int64Pointer(innerIndex), false)
			require.Equal(t, float64(innerIndex+4), execution.ExecutorInput.GetInputs().GetParameterValues()["num"].GetNumberValue())
			outputPath := execution.ExecutorInput.GetOutputs().GetParameters()["Output"].GetOutputFile()
			tc.RunLauncher(execution, map[string][]byte{
				"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
				outputPath:                              []byte(fmt.Sprint((innerIndex + 4) * 2)),
			}, true)
		}
		tc.ExitDag()
		consumer, _ := tc.RunContainerDriver("simple-add", outer, util.Int64Pointer(outerIndex), true)
		values := consumer.ExecutorInput.GetInputs().GetParameterValues()["nums"].GetListValue().GetValues()
		require.Len(t, values, 3)
		for i, value := range values {
			require.Equal(t, float64((i+4)*2), value.GetNumberValue())
		}
	}
	tc.ExitDag()
	consumer, _ := tc.RunContainerDriver("nested-add", tc.RootTask, nil, true)
	outerValues := consumer.ExecutorInput.GetInputs().GetParameterValues()["nums"].GetListValue().GetValues()
	require.Len(t, outerValues, 3)
	for _, outerValue := range outerValues {
		innerValues := outerValue.GetListValue().GetValues()
		require.Len(t, innerValues, 3)
		for i, value := range innerValues {
			require.Equal(t, float64((i+4)*2), value.GetNumberValue())
		}
	}
}

func TestSameIterationConditionScalarFixture(t *testing.T) {
	trials, err := structpb.NewValue([]interface{}{1, 2, 3})
	require.NoError(t, err)
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{ParameterValues: map[string]*structpb.Value{
		"trials": trials, "add_drumroll": structpb.NewBoolValue(true), "repeat_if_lucky_number": structpb.NewBoolValue(true),
	}},
		"../../../../test_data/sdk_compiled_pipelines/valid/if_elif_else_complex.yaml")
	_, loop := tc.RunDagDriver("for-loop-1", tc.RootTask)
	for index := int64(0); index < 3; index++ {
		execution, _ := tc.RunContainerDriver("int-0-to-9999", loop, util.Int64Pointer(index), false)
		outputPath := execution.ExecutorInput.GetOutputs().GetParameters()["Output"].GetOutputFile()
		tc.RunLauncher(execution, map[string][]byte{
			"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
			outputPath:                              []byte(fmt.Sprint(605 + index)),
		}, true)
		consumer, _ := tc.RunDagDriver("condition-branches-4", loop, index)
		require.Equal(t, float64(605+index), consumer.ExecutorInput.GetInputs().GetParameterValues()["pipelinechannel--int-0-to-9999-Output"].GetNumberValue())
		tc.ExitDag()
	}
}
