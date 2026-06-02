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
	opts := common.Options{IterationIndex: 1}
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
