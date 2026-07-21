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

func TestResolveDriverSpecs_FallsBackToCompleteLegacyTuple(t *testing.T) {
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
				Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "exec-missing"},
			},
		},
	}

	scopePath := mustBuildScopePath(t, spec, "root", "task-1")
	legacyComponentSpec := mustProtoJSON(t, &pipelinespec.ComponentSpec{
		Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "legacy-exec"},
	})
	legacyTaskSpec := mustProtoJSON(t, &pipelinespec.PipelineTaskSpec{
		TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "legacy task"},
	})
	legacyContainerSpec := mustProtoJSON(t, &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
		Image:   "python:3.9",
		Command: []string{"python"},
		Args:    []string{"-m", "legacy"},
	})

	componentSpec, taskSpec, containerSpec, err := resolveDriverSpecs(
		scopePath,
		CONTAINER,
		legacyComponentSpec,
		legacyTaskSpec,
		legacyContainerSpec,
	)
	require.NoError(t, err)
	require.NotNil(t, componentSpec)
	require.NotNil(t, taskSpec)
	require.NotNil(t, containerSpec)
	assert.Equal(t, "legacy-exec", componentSpec.GetExecutorLabel())
	assert.Equal(t, "legacy task", taskSpec.GetTaskInfo().GetName())
	assert.Equal(t, "python:3.9", containerSpec.GetImage())
}

func TestResolveDriverSpecs_DoesNotFallbackOnMalformedDeploymentSpec(t *testing.T) {
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
		DeploymentSpec: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"executors": structpb.NewStringValue("not-an-object"),
			},
		},
	}

	scopePath := mustBuildScopePath(t, spec, "root", "task-1")
	_, _, _, err := resolveDriverSpecs(
		scopePath,
		CONTAINER,
		mustProtoJSON(t, &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "legacy-exec"},
		}),
		mustProtoJSON(t, &pipelinespec.PipelineTaskSpec{}),
		mustProtoJSON(t, &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.9"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal deployment spec")
}

func TestResolveDriverSpecs_RejectsRootDriverOnNonDagComponent(t *testing.T) {
	spec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "exec-1"},
		},
	}

	scopePath := mustBuildScopePath(t, spec, "root")
	_, _, _, err := resolveDriverSpecs(scopePath, ROOT_DAG, "", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "root driver requires a DAG root component")
}

func TestResolveDriverSpecs_RejectsContainerDriverOnWrongExecutorKind(t *testing.T) {
	deploymentConfig := &pipelinespec.PipelineDeploymentConfig{
		Executors: map[string]*pipelinespec.PipelineDeploymentConfig_ExecutorSpec{
			"exec-1": {
				Spec: &pipelinespec.PipelineDeploymentConfig_ExecutorSpec_Importer{
					Importer: &pipelinespec.PipelineDeploymentConfig_ImporterSpec{},
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
	_, _, _, err := resolveDriverSpecs(scopePath, CONTAINER, "", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not contain a container spec")
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

func mustProtoJSON(t *testing.T, message proto.Message) string {
	t.Helper()

	jsonBytes, err := protojson.Marshal(message)
	require.NoError(t, err)
	return string(jsonBytes)
}
