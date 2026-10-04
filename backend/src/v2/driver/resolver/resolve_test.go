package resolver

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveInputs_AllowsEmptyParameterIterator(t *testing.T) {
	opts := common.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: "parent-task",
			Inputs: &apiv2beta1.PipelineTask_InputOutputs{},
		},
		Task: &pipelinespec.PipelineTaskSpec{
			Iterator: &pipelinespec.PipelineTaskSpec_ParameterIterator{
				ParameterIterator: &pipelinespec.ParameterIteratorSpec{
					ItemInput: "item",
					Items: &pipelinespec.ParameterIteratorSpec_ItemsSpec{
						Kind: &pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw{
							Raw: "[]",
						},
					},
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{},
	}

	inputs, iterationCount, err := ResolveInputs(context.Background(), opts)
	require.NoError(t, err)
	require.NotNil(t, inputs)
	require.NotNil(t, iterationCount)
	assert.Equal(t, 0, *iterationCount)
}
