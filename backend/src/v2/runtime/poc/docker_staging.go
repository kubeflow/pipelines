package poc

import (
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
)

// Worker TMPDIR must not change the temporary directory inside Linux containers.
const dockerContainerTempRoot = "/tmp"

func isWorkspaceRuntimeArtifact(artifact *pipelinespec.RuntimeArtifact) bool {
	return artifact.GetMetadata().GetFields()["_kfp_workspace"].GetBoolValue()
}

func rebaseDockerInputArtifacts(input *pipelinespec.ExecutorInput, root string) error {
	for _, artifacts := range input.GetInputs().GetArtifacts() {
		for _, artifact := range artifacts.GetArtifacts() {
			if artifact == nil || isWorkspaceRuntimeArtifact(artifact) {
				continue
			}
			if root == "" {
				return fmt.Errorf("docker input artifacts require an attempt-local artifact root")
			}
			// Producer CustomPath belongs to another attempt. Only the durable URI
			// identifies bytes to download into this consumer's staging directory.
			path, err := resolvedRuntimeArtifactLocalPath(root, "", &pipelinespec.RuntimeArtifact{Uri: artifact.GetUri()})
			if err != nil {
				return fmt.Errorf("cannot stage Docker input artifact %q: %w", artifact.GetUri(), err)
			}
			artifact.CustomPath = &path
		}
	}
	return nil
}

func rebaseDockerOutputFiles(input *pipelinespec.ExecutorInput, root, tempRoot string) {
	for _, parameter := range input.GetOutputs().GetParameters() {
		if parameter == nil {
			continue
		}
		parameter.OutputFile = hostPathForContainerPath(parameter.GetOutputFile(), root, tempRoot)
	}
	if outputs := input.GetOutputs(); outputs != nil && outputs.OutputFile != "" {
		outputs.OutputFile = hostPathForContainerPath(outputs.OutputFile, root, tempRoot)
	}
}
