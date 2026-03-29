package main

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
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
	componentSpec, taskSpec, containerSpec, err := resolveDriverSpecs(scopePath, CONTAINER)
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
	componentSpec, taskSpec, containerSpec, err := resolveDriverSpecs(scopePath, RootDag)
	require.NoError(t, err)

	require.NotNil(t, componentSpec)
	assert.NotNil(t, componentSpec.GetDag())
	assert.Nil(t, taskSpec)
	assert.Nil(t, containerSpec)
}

func TestResolveDriverSpecs_ErrorsOnMalformedDeploymentSpec(t *testing.T) {
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
	_, _, _, err := resolveDriverSpecs(scopePath, CONTAINER)
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
	_, _, _, err := resolveDriverSpecs(scopePath, RootDag)
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
	_, _, _, err := resolveDriverSpecs(scopePath, CONTAINER)
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

type scopePathAPI struct {
	kfpapi.API
	t    *testing.T
	run  *go_client.Run
	spec *structpb.Struct
	err  error
}

func (a scopePathAPI) FetchPipelineSpecFromRun(_ context.Context, run *go_client.Run) (*structpb.Struct, error) {
	a.t.Helper()
	assert.Same(a.t, a.run, run)
	return a.spec, a.err
}

func TestBuildScopePathUsesRequestDriverType(t *testing.T) {
	spec := &structpb.Struct{}
	require.NoError(t, spec.UnmarshalJSON([]byte(`{
		"root": {"dag": {"tasks": {"group": {"componentRef": {"name": "group"}}}}},
		"components": {
			"group": {"dag": {"tasks": {"leaf": {"componentRef": {"name": "leaf"}}}}},
			"leaf": {"executorLabel": "exec-leaf"}
		}
	}`)))
	run := &go_client.Run{RunId: "run-id"}

	for _, tc := range []struct {
		name       string
		driverType string
		parentTask *go_client.PipelineTask
		taskName   string
		wantPath   string
	}{
		{name: "root", driverType: RootDag, wantPath: "root"},
		{name: "dag", driverType: DAG, parentTask: &go_client.PipelineTask{ScopePath: "root"}, taskName: "group", wantPath: "root.group"},
		{name: "container", driverType: CONTAINER, parentTask: &go_client.PipelineTask{ScopePath: "root.group"}, taskName: "leaf", wantPath: "root.group.leaf"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			api := scopePathAPI{t: t, run: run, spec: spec}

			path, err := buildScopePath(context.Background(), run, tc.parentTask, tc.taskName, tc.driverType, api)

			require.NoError(t, err)
			require.NotNil(t, path)
			assert.Equal(t, tc.wantPath, path.DotNotation())
		})
	}
}

func TestBuildScopePathPropagatesRunSpecLookupFailure(t *testing.T) {
	run := &go_client.Run{RunId: "run-id"}
	api := scopePathAPI{t: t, run: run, err: assert.AnError}

	path, err := buildScopePath(context.Background(), run, nil, "", RootDag, api)

	assert.Nil(t, path)
	assert.ErrorIs(t, err, assert.AnError)
}
