package poc

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestDockerContainerEnvironmentAndCommand(t *testing.T) {
	request := dockerTestRequest(t)
	request.ArtifactLocalRoot = t.TempDir()
	request.Container.Env = []*pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_EnvVar{
		{Name: "GREETING", Value: "{{$.inputs.parameters['greeting']}}"},
		{Name: "MESSAGE", Value: "$(GREETING) world"},
		{Name: "LITERAL", Value: "$$(GREETING) $(UNKNOWN)"},
	}
	request.ExecutorInput = &pipelinespec.ExecutorInput{Inputs: &pipelinespec.ExecutorInput_Inputs{
		ParameterValues: map[string]*structpb.Value{"greeting": structpb.NewStringValue("hello")},
	}}
	request.Container.Command = []string{"sh", "-c"}
	request.Container.Args = []string{"$(MESSAGE)", "$$(GREETING)"}
	args, err := dockerCreateArgs(request, attemptWorkloadName(request.Identity))
	require.NoError(t, err)
	assert.Contains(t, args, "GREETING=hello")
	assert.Contains(t, args, "MESSAGE=hello world")
	assert.Contains(t, args, "LITERAL=$(GREETING) $(UNKNOWN)")
	assert.Contains(t, args, "ARTIFACT_LOCAL_PATH="+request.ArtifactLocalRoot)
	assert.Contains(t, args, "--entrypoint")
	assert.Equal(t, []string{"python:3.11", "-c", "hello world", "$(GREETING)"}, args[len(args)-4:])
	request.Container.Command = nil
	request.Container.Args = []string{"{{$.inputs.parameters['greeting']}}"}
	args, err = dockerCreateArgs(request, attemptWorkloadName(request.Identity))
	require.NoError(t, err)
	assert.NotContains(t, args, "--entrypoint")
	assert.Equal(t, "hello", args[len(args)-1])
}

func TestDockerContainerResourceLimits(t *testing.T) {
	for _, test := range []struct {
		name     string
		spec     *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec
		expected []string
	}{
		{"legacy", &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{CpuLimit: 1.5, MemoryLimit: 2}, []string{"--cpus", "1.5", "--memory", "2000000000"}}, //nolint:staticcheck // Legacy IR fields are deliberately exercised.
		{"modern", &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{ResourceCpuLimit: "500m", ResourceMemoryLimit: "1Gi"}, []string{"--cpus", "0.5", "--memory", "1073741824"}},
		{"parameter", &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{ResourceCpuLimit: "{{$.inputs.parameters['cpu']}}"}, []string{"--cpus", "2"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			container := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Resources: test.spec}
			args, err := dockerResourceArgs(container, &pipelinespec.ExecutorInput{Inputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{"cpu": structpb.NewNumberValue(2)},
			}})
			require.NoError(t, err)
			assert.Equal(t, test.expected, args)
		})
	}
}

func TestDockerContainerRejectsUnsupportedResources(t *testing.T) {
	for _, test := range []struct {
		name    string
		spec    *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec
		message string
	}{
		{"cpu request", &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{ResourceCpuRequest: "1"}, "scheduling requests"},
		{"memory request", &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{MemoryRequest: 1}, "scheduling requests"}, //nolint:staticcheck // Legacy IR fields are deliberately exercised.
		{"accelerator", &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{Accelerator: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec_AcceleratorConfig{ResourceType: "nvidia.com/gpu"}}, "accelerator"},
		{"invalid", &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{ResourceCpuLimit: "not-a-quantity"}, "invalid"},
		{"negative", &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec_ResourceSpec{ResourceMemoryLimit: "-1Gi"}, "negative"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := dockerResourceArgs(&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Resources: test.spec}, nil)
			require.ErrorContains(t, err, test.message)
		})
	}
}

func TestPrepareDockerTaskRejectsKubernetesRequirementsButAcceptsEmptyConfig(t *testing.T) {
	for _, config := range []*kubernetesplatform.KubernetesExecutorConfig{
		nil, {}, {NodeSelector: &kubernetesplatform.NodeSelector{}},
		{ImagePullPolicy: "IfNotPresent"},
	} {
		preparation := &TaskPreparationRequest{Run: &model.Run{UUID: "run"}, Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python"}, KubernetesExecutorConfig: config}
		require.NoError(t, PrepareDockerTask(context.Background(), preparation))
	}
	preparation := &TaskPreparationRequest{
		Run: &model.Run{UUID: "run"}, Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python"},
		KubernetesExecutorConfig: &kubernetesplatform.KubernetesExecutorConfig{ImagePullPolicy: "Never"},
	}
	require.ErrorContains(t, PrepareDockerTask(context.Background(), preparation), "Kubernetes execution requirements")
	preparation.KubernetesExecutorConfig = &kubernetesplatform.KubernetesExecutorConfig{ImagePullPolicy: "Always"}
	require.ErrorContains(t, PrepareDockerTask(context.Background(), preparation), "Kubernetes execution requirements")
	preparation.KubernetesExecutorConfig = &kubernetesplatform.KubernetesExecutorConfig{ImagePullPolicy: "IfNotPresent", ActiveDeadlineSeconds: 1}
	require.ErrorContains(t, PrepareDockerTask(context.Background(), preparation), "Kubernetes execution requirements")
	preparation.KubernetesExecutorConfig = &kubernetesplatform.KubernetesExecutorConfig{ImagePullPolicy: "IfNotPresent"}
	require.NoError(t, PrepareDockerTask(context.Background(), preparation))
	assert.Equal(t, "IfNotPresent", preparation.KubernetesExecutorConfig.ImagePullPolicy, "validation must not mutate the original config")
}
