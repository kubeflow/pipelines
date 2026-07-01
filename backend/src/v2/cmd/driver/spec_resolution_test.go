package main

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestResolveDriverSpecsFromScopePath(t *testing.T) {
	deploymentConfig := &pipelinespec.PipelineDeploymentConfig{
		Executors: map[string]*pipelinespec.PipelineDeploymentConfig_ExecutorSpec{
			"exec-1": {
				Spec: &pipelinespec.PipelineDeploymentConfig_ExecutorSpec_Container{
					Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
						Image:   "python:3.11",
						Command: []string{"python"},
						Args:    []string{"-c", "print('hello')"},
					},
				},
			},
		},
	}

	spec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"task-1": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "display task"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "comp-1"},
						},
					},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"comp-1": {
				Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "exec-1"},
			},
		},
		DeploymentSpec: mustStructFromProtoJSON(t, deploymentConfig),
	}

	scopePath := mustBuildScopePath(t, spec, "root", "task-1")
	componentSpec, taskSpec, containerSpec, err := resolveDriverSpecs(scopePath, CONTAINER, "", "", "")
	require.NoError(t, err)

	require.NotNil(t, componentSpec)
	assert.Equal(t, "exec-1", componentSpec.GetExecutorLabel())

	require.NotNil(t, taskSpec)
	assert.Equal(t, "display task", taskSpec.GetTaskInfo().GetName())

	require.NotNil(t, containerSpec)
	assert.Equal(t, "python:3.11", containerSpec.GetImage())
	assert.Equal(t, []string{"python"}, containerSpec.GetCommand())
	assert.Equal(t, []string{"-c", "print('hello')"}, containerSpec.GetArgs())
}

func TestResolveDriverSpecsForRootDag(t *testing.T) {
	spec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{},
			},
		},
	}

	scopePath := mustBuildScopePath(t, spec, "root")
	componentSpec, taskSpec, containerSpec, err := resolveDriverSpecs(scopePath, ROOT_DAG, "", "", "")
	require.NoError(t, err)

	require.NotNil(t, componentSpec)
	assert.NotNil(t, componentSpec.GetDag())
	assert.Nil(t, taskSpec)
	assert.Nil(t, containerSpec)
}

func mustBuildScopePath(t *testing.T, spec *pipelinespec.PipelineSpec, tasks ...string) *util.ScopePath {
	t.Helper()

	rawSpec := mustStructFromProtoJSON(t, spec)
	scopePath, err := util.NewScopePathFromStruct(rawSpec)
	require.NoError(t, err)

	for _, taskName := range tasks {
		require.NoError(t, scopePath.Push(taskName))
	}

	return &scopePath
}

func mustStructFromProtoJSON(t *testing.T, message proto.Message) *structpb.Struct {
	t.Helper()

	jsonBytes, err := protojson.Marshal(message)
	require.NoError(t, err)

	rawStruct := &structpb.Struct{}
	require.NoError(t, rawStruct.UnmarshalJSON(jsonBytes))
	return rawStruct
}
