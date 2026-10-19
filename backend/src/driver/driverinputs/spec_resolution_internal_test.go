package driverinputs

import (
	"context"
	"errors"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
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
	componentSpec, taskSpec, containerSpec, err := ResolveSpecsFromScopePath(scopePath, driverapi.ContainerDriver)
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

func TestResolveDriverSpecsForRootDAG(t *testing.T) {
	spec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{},
			},
		},
	}

	scopePath := mustBuildScopePath(t, spec, "root")
	componentSpec, taskSpec, containerSpec, err := ResolveSpecsFromScopePath(scopePath, driverapi.RootDAGDriver)
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
	_, _, _, err := ResolveSpecsFromScopePath(scopePath, driverapi.ContainerDriver)
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
	_, _, _, err := ResolveSpecsFromScopePath(scopePath, driverapi.RootDAGDriver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ROOT_DAG driver requires a DAG component")
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
	_, _, _, err := ResolveSpecsFromScopePath(scopePath, driverapi.ContainerDriver)
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
		driverType driverapi.DriverType
		parentTask *go_client.PipelineTask
		taskName   string
		wantPath   string
	}{
		{name: "root", driverType: driverapi.RootDAGDriver, wantPath: "root"},
		{name: "dag", driverType: driverapi.DAGDriver, parentTask: &go_client.PipelineTask{ScopePath: "root"}, taskName: "group", wantPath: "root.group"},
		{name: "container", driverType: driverapi.ContainerDriver, parentTask: &go_client.PipelineTask{ScopePath: "root.group"}, taskName: "leaf", wantPath: "root.group.leaf"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			api := scopePathAPI{t: t, run: run, spec: spec}

			path, err := BuildScopePath(context.Background(), run, tc.parentTask, tc.taskName, tc.driverType, api)

			require.NoError(t, err)
			require.NotNil(t, path)
			assert.Equal(t, tc.wantPath, path.DotNotation())
		})
	}
}

func TestBuildScopePathPropagatesRunSpecLookupFailure(t *testing.T) {
	run := &go_client.Run{RunId: "run-id"}
	api := scopePathAPI{t: t, run: run, err: assert.AnError}

	path, err := BuildScopePath(context.Background(), run, nil, "", driverapi.RootDAGDriver, api)

	assert.Nil(t, path)
	assert.ErrorIs(t, err, assert.AnError)
}

func strPtr(s string) *string {
	return &s
}

func TestParseExecConfigJSON(t *testing.T) {
	tt := []struct {
		name     string
		input    *string
		expected *kubernetesplatform.KubernetesExecutorConfig
		wantErr  bool
	}{
		{
			"Valid - test kubecfg value parse.",
			strPtr("{\"imagePullSecret\":[{\"secret_name\":\"value1\"}]}"),
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullSecret: []*kubernetesplatform.ImagePullSecret{
					{SecretName: "value1"},
				},
			},
			false,
		},
		{
			"Valid - test kubecfg value ignores unknown field.",
			strPtr("{\"imagePullSecret\":[{\"secret_name\":\"value1\"}], \"unknown_field\": \"something\"}"),
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullSecret: []*kubernetesplatform.ImagePullSecret{
					{SecretName: "value1"},
				},
			},
			false,
		},
	}

	for _, tc := range tt {
		t.Logf("Running test case: %s", tc.name)
		cfg, err := ParseKubernetesExecutorConfig(tc.input)
		assert.Equal(t, tc.wantErr, err != nil)
		assert.True(t, proto.Equal(tc.expected, cfg))
	}
}

func TestParseExecConfigJsonErrorDoesNotIncludeConfigContent(t *testing.T) {
	kubernetesConfig := `"mlflow-secret"`

	_, err := ParseKubernetesExecutorConfig(&kubernetesConfig)

	require.Error(t, err)
	assert.Equal(t, "failed to unmarshal Kubernetes config; provide valid KubernetesExecutorConfig JSON", err.Error())
	assert.NotContains(t, err.Error(), "mlflow-secret")
	assert.NotNil(t, errors.Unwrap(err))
}

func TestParseOptionalBoolFlag(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantNil bool
		want    bool
		wantErr bool
	}{
		{name: "unset empty", value: "", wantNil: true},
		{name: "true", value: "true", want: true},
		{name: "false", value: "false", want: false},
		{name: "invalid", value: "maybe", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseOptionalBool("--default_host_users", tc.value)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}
			assert.NoError(t, err)
			if tc.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.want, *got)
		})
	}
}
