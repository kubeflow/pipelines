package resolver

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
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
	inputArtifacts := []*apiv2.PipelineTask_InputOutputs_IOArtifact{
		{
			ArtifactKey: "pipelinechannel--loop-item",
			Artifacts:   []*apiv2.Artifact{{ArtifactId: "artifact-0"}},
			Producer: &apiv2.IOProducer{
				TaskName:  "loop-task",
				Iteration: util.Int64Pointer(0),
			},
		},
		{
			ArtifactKey: "pipelinechannel--loop-item",
			Artifacts:   []*apiv2.Artifact{{ArtifactId: "artifact-1"}},
			Producer: &apiv2.IOProducer{
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
	inputArtifacts := []*apiv2.PipelineTask_InputOutputs_IOArtifact{
		{
			ArtifactKey: "model_list",
			Artifacts:   []*apiv2.Artifact{{ArtifactId: "artifact-1"}},
			Type:        apiv2.IOType_COMPONENT_INPUT,
			Producer:    &apiv2.IOProducer{TaskName: "producer"},
		},
		{
			ArtifactKey: "model_list",
			Artifacts:   []*apiv2.Artifact{{ArtifactId: "artifact-2"}},
			Type:        apiv2.IOType_COMPONENT_INPUT,
			Producer:    &apiv2.IOProducer{TaskName: "producer"},
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
		ParentTask: &apiv2.PipelineTask{
			TaskId: "parent-task",
			Inputs: &apiv2.PipelineTask_InputOutputs{},
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
	assert.Equal(t, apiv2.IOType_COLLECTED_INPUTS, resolved.GetType())
	assert.Empty(t, resolved.GetArtifacts())
}

func TestResolveTaskOutputArtifact_CollectionBoundary(t *testing.T) {
	for _, taskType := range []apiv2.PipelineTask_TaskType{apiv2.PipelineTask_RUNTIME, apiv2.PipelineTask_LOOP} {
		t.Run(taskType.String(), func(t *testing.T) {
			parent := &apiv2.PipelineTask{TaskId: "outer", Name: "outer", Type: apiv2.PipelineTask_LOOP}
			producer := &apiv2.PipelineTask{
				TaskId: "producer", Name: "produce", Type: taskType,
				ParentTaskId:   util.StringPointer("outer"),
				TypeAttributes: &apiv2.PipelineTask_TypeAttributes{IterationIndex: util.Int64Pointer(0)},
				Outputs:        &apiv2.PipelineTask_InputOutputs{Artifacts: []*apiv2.PipelineTask_InputOutputs_IOArtifact{iteratorArtifact("result", 0, "artifact")}},
			}
			_, resolved, ioType, err := resolveInputArtifact(common.Options{
				ParentTask: parent, IterationIndex: 0, Run: &apiv2.Run{Tasks: []*apiv2.PipelineTask{producer}},
			}, "input", &pipelinespec.TaskInputsSpec_InputArtifactSpec{
				Kind: &pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact{
					TaskOutputArtifact: &pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec{ProducerTask: "produce", OutputArtifactKey: "result"},
				},
			}, nil)
			require.NoError(t, err)
			require.Len(t, resolved.GetArtifacts(), 1)
			assert.Equal(t, "artifact", resolved.GetArtifacts()[0].GetArtifactId())
			if taskType == apiv2.PipelineTask_LOOP {
				assert.Equal(t, apiv2.IOType_COLLECTED_INPUTS, ioType)
			} else {
				assert.Equal(t, apiv2.IOType_TASK_OUTPUT_INPUT, ioType)
			}
		})
	}
}

func TestFindArtifactByProducerKeyInList_WrapsSingletonIteratorOutput(t *testing.T) {
	resolved, err := findArtifactByProducerKeyInList(
		"result",
		"producer",
		[]*apiv2.PipelineTask_InputOutputs_IOArtifact{
			iteratorArtifact("result", 0, "artifact-1"),
		},
		true,
	)

	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, apiv2.IOType_COLLECTED_INPUTS, resolved.GetType())
	require.Len(t, resolved.GetArtifacts(), 1)
	assert.Equal(t, "artifact-1", resolved.GetArtifacts()[0].GetArtifactId())
}

func TestFindArtifactByProducerKeyInList_OrdersIteratorOutputs(t *testing.T) {
	resolved, err := findArtifactByProducerKeyInList(
		"result",
		"producer",
		[]*apiv2.PipelineTask_InputOutputs_IOArtifact{
			iteratorArtifact("result", 2, "artifact-2"),
			iteratorArtifact("result", 0, "artifact-0"),
			iteratorArtifact("result", 1, "artifact-1"),
		},
		true,
	)

	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, apiv2.IOType_COLLECTED_INPUTS, resolved.GetType())
	require.Len(t, resolved.GetArtifacts(), 3)
	assert.Equal(t, "artifact-0", resolved.GetArtifacts()[0].GetArtifactId())
	assert.Equal(t, "artifact-1", resolved.GetArtifacts()[1].GetArtifactId())
	assert.Equal(t, "artifact-2", resolved.GetArtifacts()[2].GetArtifactId())
}

func iteratorArtifact(
	key string,
	iteration int64,
	artifactID string,
) *apiv2.PipelineTask_InputOutputs_IOArtifact {
	return &apiv2.PipelineTask_InputOutputs_IOArtifact{
		ArtifactKey: key,
		Type:        apiv2.IOType_ITERATOR_OUTPUT,
		Artifacts:   []*apiv2.Artifact{{ArtifactId: artifactID}},
		Producer: &apiv2.IOProducer{
			TaskName:  "producer",
			Iteration: util.Int64Pointer(iteration),
		},
	}
}
