package resolver

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveArtifactComponentInputParameter_UsesMatchingIteration(t *testing.T) {
	opts := common.Options{
		IterationIndex: 1,
		Task: &pipelinespec.PipelineTaskSpec{
			Iterator: &pipelinespec.PipelineTaskSpec_ArtifactIterator{
				ArtifactIterator: &pipelinespec.ArtifactIteratorSpec{
					ItemInput: "pipelinechannel--loop-item",
				},
			},
		},
	}
	artifactSpec := &pipelinespec.TaskInputsSpec_InputArtifactSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputArtifactSpec_ComponentInputArtifact{
			ComponentInputArtifact: "pipelinechannel--loop-item",
		},
	}
	inputArtifacts := []*apiv2beta1.PipelineTask_InputOutputs_IOArtifact{
		{
			ArtifactKey: "pipelinechannel--loop-item",
			Artifacts:   []*apiv2beta1.Artifact{{ArtifactId: "artifact-0"}},
			Producer: &apiv2beta1.IOProducer{
				TaskName:  "loop-task",
				Iteration: util.Int64Pointer(0),
			},
		},
		{
			ArtifactKey: "pipelinechannel--loop-item",
			Artifacts:   []*apiv2beta1.Artifact{{ArtifactId: "artifact-1"}},
			Producer: &apiv2beta1.IOProducer{
				TaskName:  "loop-task",
				Iteration: util.Int64Pointer(1),
			},
		},
	}

	resolved, err := resolveArtifactComponentInputParameter(opts, artifactSpec, inputArtifacts)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.Len(t, resolved.GetArtifacts(), 1)
	assert.Equal(t, "artifact-1", resolved.GetArtifacts()[0].GetArtifactId())
}

func TestResolveArtifactComponentInputParameter_CollectsAllMatchingArtifacts(t *testing.T) {
	opts := common.Options{}
	artifactSpec := &pipelinespec.TaskInputsSpec_InputArtifactSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputArtifactSpec_ComponentInputArtifact{
			ComponentInputArtifact: "model_list",
		},
	}
	inputArtifacts := []*apiv2beta1.PipelineTask_InputOutputs_IOArtifact{
		{
			ArtifactKey: "model_list",
			Artifacts:   []*apiv2beta1.Artifact{{ArtifactId: "artifact-1"}},
			Type:        apiv2beta1.IOType_COMPONENT_INPUT,
			Producer:    &apiv2beta1.IOProducer{TaskName: "producer"},
		},
		{
			ArtifactKey: "model_list",
			Artifacts:   []*apiv2beta1.Artifact{{ArtifactId: "artifact-2"}},
			Type:        apiv2beta1.IOType_COMPONENT_INPUT,
			Producer:    &apiv2beta1.IOProducer{TaskName: "producer"},
		},
	}

	resolved, err := resolveArtifactComponentInputParameter(opts, artifactSpec, inputArtifacts)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.Len(t, resolved.GetArtifacts(), 2)
	assert.Equal(t, "artifact-1", resolved.GetArtifacts()[0].GetArtifactId())
	assert.Equal(t, "artifact-2", resolved.GetArtifacts()[1].GetArtifactId())
}

func TestResolveArtifacts_SkipsMissingOptionalComponentInput(t *testing.T) {
	opts := common.Options{
		ParentTask: &apiv2beta1.PipelineTask{
			TaskId: "parent-task",
			Inputs: &apiv2beta1.PipelineTask_InputOutputs{},
		},
		Task: &pipelinespec.PipelineTaskSpec{
			Inputs: &pipelinespec.TaskInputsSpec{
				Artifacts: map[string]*pipelinespec.TaskInputsSpec_InputArtifactSpec{
					"dataset": {
						Kind: &pipelinespec.TaskInputsSpec_InputArtifactSpec_ComponentInputArtifact{
							ComponentInputArtifact: "dataset",
						},
					},
				},
			},
		},
		Component: &pipelinespec.ComponentSpec{
			InputDefinitions: &pipelinespec.ComponentInputsSpec{
				Artifacts: map[string]*pipelinespec.ComponentInputsSpec_ArtifactSpec{
					"dataset": {IsOptional: true},
				},
			},
		},
	}

	artifacts, err := resolveArtifacts(opts)
	require.NoError(t, err)
	assert.Empty(t, artifacts)
}

func TestResolveTaskOutputArtifact_EmptyLoopProducesEmptyCollection(t *testing.T) {
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

	producer, resolved, err := resolveTaskOutputArtifact(opts, &pipelinespec.TaskInputsSpec_InputArtifactSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact{
			TaskOutputArtifact: &pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec{
				ProducerTask:      "loop",
				OutputArtifactKey: "models",
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, producer)
	require.NotNil(t, resolved)
	assert.Equal(t, apiv2beta1.IOType_COLLECTED_INPUTS, resolved.GetType())
	assert.Empty(t, resolved.GetArtifacts())
}

func TestFindArtifactByProducerKeyInList_WrapsSingletonIteratorOutput(t *testing.T) {
	resolved, err := findArtifactByProducerKeyInList(
		"result",
		"producer",
		[]*apiv2beta1.PipelineTask_InputOutputs_IOArtifact{
			iteratorArtifact("result", 0, "artifact-1"),
		},
	)

	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, apiv2beta1.IOType_COLLECTED_INPUTS, resolved.GetType())
	require.Len(t, resolved.GetArtifacts(), 1)
	assert.Equal(t, "artifact-1", resolved.GetArtifacts()[0].GetArtifactId())
}

func TestFindArtifactByProducerKeyInList_OrdersIteratorOutputs(t *testing.T) {
	resolved, err := findArtifactByProducerKeyInList(
		"result",
		"producer",
		[]*apiv2beta1.PipelineTask_InputOutputs_IOArtifact{
			iteratorArtifact("result", 2, "artifact-2"),
			iteratorArtifact("result", 0, "artifact-0"),
			iteratorArtifact("result", 1, "artifact-1"),
		},
	)

	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, apiv2beta1.IOType_COLLECTED_INPUTS, resolved.GetType())
	require.Len(t, resolved.GetArtifacts(), 3)
	assert.Equal(t, "artifact-0", resolved.GetArtifacts()[0].GetArtifactId())
	assert.Equal(t, "artifact-1", resolved.GetArtifacts()[1].GetArtifactId())
	assert.Equal(t, "artifact-2", resolved.GetArtifacts()[2].GetArtifactId())
}

func iteratorArtifact(
	key string,
	iteration int64,
	artifactID string,
) *apiv2beta1.PipelineTask_InputOutputs_IOArtifact {
	return &apiv2beta1.PipelineTask_InputOutputs_IOArtifact{
		ArtifactKey: key,
		Type:        apiv2beta1.IOType_ITERATOR_OUTPUT,
		Artifacts:   []*apiv2beta1.Artifact{{ArtifactId: artifactID}},
		Producer: &apiv2beta1.IOProducer{
			TaskName:  "producer",
			Iteration: util.Int64Pointer(iteration),
		},
	}
}
