package poc

import (
	"context"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestMultiUserValidationDoesNotDependOnLostTaskRecovery(t *testing.T) {
	require.ErrorContains(t, validateExecutor(&barrierExecutor{}, true), "multi-user")
	require.NoError(t, validateExecutor(&barrierExecutor{}, false))
}

const nestedCapabilityPipeline = `{
  "root":{"dag":{"tasks":{"outer":{"componentRef":{"name":"nested"}}}}},
  "components":{
    "nested":{"dag":{"tasks":{"hidden":{"componentRef":{"name":"leaf"},"triggerPolicy":{"condition":"false"}}}}},
    "leaf":{"executorLabel":"exec"}
  },
  "deploymentSpec":{"executors":{"exec":{"container":{"image":"alpine","command":["true"]}}}}
}`

func TestWholePipelineCapabilitiesIncludeUnselectedNestedBranches(t *testing.T) {
	var spec pipelinespec.PipelineSpec
	var platform pipelinespec.SinglePlatformSpec
	require.NoError(t, protojson.Unmarshal([]byte(nestedCapabilityPipeline), &spec))
	require.NoError(t, protojson.Unmarshal([]byte(`{"deploymentSpec":{"executors":{"exec":{"imagePullPolicy":"Always"}}}}`), &platform))
	err := validatePipelineCapabilities(&spec, &platform, dockerExecutor{}.Capabilities(), "")
	require.ErrorContains(t, err, "root.outer.hidden")
	require.ErrorContains(t, err, "SupportsKubernetes")
	require.NoError(t, validatePipelineCapabilities(&spec, &platform, kubernetesExecutor{}.Capabilities(), ""))
	require.ErrorContains(t, validatePipelineCapabilities(&spec, nil, dockerExecutor{}.Capabilities(), WorkspaceTypePVC), "workspace provisioner")
	require.NoError(t, validatePipelineCapabilities(&spec, nil, dockerExecutor{}.Capabilities(), WorkspaceTypeHostPath))
	require.ErrorContains(t, validatePipelineCapabilities(&spec, nil, ExecutorCapabilities{}, ""), "SupportsContainers")
}

func TestPipelineCapabilityFailurePrecedesTaskCreation(t *testing.T) {
	coordinator := NewCoordinator(nil, nil, nil, nil, nil, nil, nil, false)
	var job pipelinespec.PipelineJob
	require.NoError(t, protojson.Unmarshal([]byte(`{"pipelineSpec":`+nestedCapabilityPipeline+`}`), &job))
	var platform pipelinespec.SinglePlatformSpec
	require.NoError(t, protojson.Unmarshal([]byte(`{"deploymentSpec":{"executors":{"exec":{"imagePullPolicy":"Always"}}}}`), &platform))
	executor := ExecutorDocker
	err := coordinator.SubmitRun(context.Background(), &model.Run{
		UUID: "unsupported", RunDetails: model.RunDetails{ExecutorType: &executor},
	}, &job, &platform)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "SupportsKubernetes"), err)
}

func TestWholePipelineValidationIncludesDockerLifecycleAndSchedulingRequests(t *testing.T) {
	for _, requirements := range []string{
		`"lifecycle":{"preCacheCheck":{"command":["true"]}}`,
		`"resources":{"resourceCpuRequest":"1"}`,
		`"resources":{"resourceMemoryRequest":"1Gi"}`,
	} {
		var spec pipelinespec.PipelineSpec
		source := strings.Replace(nestedCapabilityPipeline, `"command":["true"]`, `"command":["true"],`+requirements, 1)
		require.NoError(t, protojson.Unmarshal([]byte(source), &spec))
		err := validatePipelineCapabilities(&spec, nil, dockerExecutor{}.Capabilities(), "")
		require.ErrorContains(t, err, "root.outer.hidden", requirements)
	}
}
