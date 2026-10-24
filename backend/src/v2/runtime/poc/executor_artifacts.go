package poc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"google.golang.org/protobuf/proto"
)

type invalidDeclaredOutput struct{ err error }

func (e *invalidDeclaredOutput) Error() string { return e.err.Error() }
func (e *invalidDeclaredOutput) Unwrap() error { return e.err }

// uploadExecutionArtifacts is shared worker-side transfer code, independent of scheduling.
func uploadExecutionArtifacts(ctx context.Context, store component.ObjectStoreClientInterface, root string, input *pipelinespec.ExecutorInput) error {
	if input.GetOutputs() == nil || len(input.GetOutputs().GetArtifacts()) == 0 {
		return nil
	}
	if store == nil {
		return fmt.Errorf("artifact storage is unavailable; configure the worker's object store")
	}
	input = proto.Clone(input).(*pipelinespec.ExecutorInput)
	updated, err := loadExecutorOutputArtifacts(input.GetOutputs().GetOutputFile())
	if err != nil {
		return &invalidDeclaredOutput{err: err}
	}
	for key, artifacts := range input.GetOutputs().GetArtifacts() {
		for index, artifact := range artifacts.GetArtifacts() {
			if index < len(updated[key].GetArtifacts()) {
				mergeRuntimeArtifacts(updated[key].GetArtifacts()[index], artifact)
			}
			path, err := resolvedRuntimeArtifactLocalPath(root, dockerTempMountSourceForArtifactRoot(root), artifact)
			if err != nil {
				return &invalidDeclaredOutput{err: err}
			}
			if _, err := os.Stat(path); err != nil {
				if errors.Is(err, os.ErrNotExist) && runtimeArtifactMayOmitLocalFile(artifact) {
					continue
				}
				return &invalidDeclaredOutput{err: fmt.Errorf("declared output %q is unavailable: %w", key, err)}
			}
			if artifact.GetUri() == "" {
				return &invalidDeclaredOutput{err: fmt.Errorf("declared output %q has no durable URI", key)}
			}
			if !strings.HasPrefix(artifact.GetUri(), "oci://") {
				if err := store.UploadArtifact(ctx, path, artifact.GetUri(), key); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
