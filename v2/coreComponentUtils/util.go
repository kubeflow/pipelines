package coreComponentUtils

import (
	"fmt"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"os"
	"path/filepath"
	"strings"
)

func PrepareOutputFolders(executorInput *pipelinespec.ExecutorInput) error {
	for name, parameter := range executorInput.GetOutputs().GetParameters() {
		dir := filepath.Dir(parameter.OutputFile)
		if err := os.MkdirAll(dir, 0644); err != nil {
			return fmt.Errorf("failed to create directory %q for output parameter %q: %w", dir, name, err)
		}
	}

	for name, artifactList := range executorInput.GetOutputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		outputArtifact := artifactList.Artifacts[0]

		localPath, err := LocalPathForURI(outputArtifact.Uri)
		if err != nil {
			return fmt.Errorf("failed to generate local storage path for output artifact %q: %w", name, err)
		}

		if err := os.MkdirAll(filepath.Dir(localPath), 0644); err != nil {
			return fmt.Errorf("unable to create directory %q for output artifact %q: %w", filepath.Dir(localPath), name, err)
		}
	}

	return nil
}

func LocalPathForURI(uri string) (string, error) {
	if strings.HasPrefix(uri, "gs://") {
		return "/gcs/" + strings.TrimPrefix(uri, "gs://"), nil
	}
	if strings.HasPrefix(uri, "minio://") {
		return "/minio/" + strings.TrimPrefix(uri, "minio://"), nil
	}
	if strings.HasPrefix(uri, "s3://") {
		return "/s3/" + strings.TrimPrefix(uri, "s3://"), nil
	}
	return "", fmt.Errorf("failed to generate local path for URI %s: unsupported storage scheme", uri)
}

// Add outputs info from component spec to executor input.
func AddOutputs(executorInput *pipelinespec.ExecutorInput, outputs *pipelinespec.ComponentOutputsSpec) error {
	if executorInput == nil {
		return fmt.Errorf("cannot add outputs to nil executor input")
	}
	if executorInput.Outputs == nil {
		executorInput.Outputs = &pipelinespec.ExecutorInput_Outputs{}
	}
	if executorInput.Outputs.Parameters == nil {
		executorInput.Outputs.Parameters = make(map[string]*pipelinespec.ExecutorInput_OutputParameter)
	}
	if executorInput.Outputs.OutputFile == "" {
		executorInput.Outputs.OutputFile = OutputMetadataFilepath
	}
	for name := range outputs.GetParameters() {
		executorInput.Outputs.Parameters[name] = &pipelinespec.ExecutorInput_OutputParameter{
			OutputFile: fmt.Sprintf("/tmp/kfp/outputs/%s", name),
		}
	}
	// artifact outputs are added in driver step
	return nil
}

const OutputMetadataFilepath = "/tmp/kfp_outputs/output_metadata.json"

