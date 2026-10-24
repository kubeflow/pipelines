package poc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
)

// Worker TMPDIR must not change the temporary directory inside Linux containers.
const dockerContainerTempRoot = "/tmp"

// stageDockerRequest is idempotent even when Submit is repeated after worker loss.
func stageDockerRequest(ctx context.Context, store component.ObjectStoreClientInterface, request TaskExecutionRequest) error {
	if request.ArtifactLocalRoot == "" || request.Identity.AttemptID == "" {
		return fmt.Errorf("docker staging requires an attempt identity and a private staging root")
	}
	if err := ensureContainerWritableDir(request.ArtifactLocalRoot); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(request.ArtifactLocalRoot, ".staging.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	for {
		err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return err
		}
		if err := waitForPluginPoll(ctx); err != nil {
			return err
		}
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	marker := filepath.Join(request.ArtifactLocalRoot, ".staged")
	if identity, err := os.ReadFile(marker); err == nil {
		if string(identity) != request.Identity.AttemptID {
			return fmt.Errorf("docker staging root belongs to a different attempt")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := downloadInputArtifacts(ctx, store, request.ExecutorInput, request.ArtifactLocalRoot); err != nil {
		return err
	}
	if err := prepareOutputArtifactFolders(request.ExecutorInput); err != nil {
		return err
	}
	file, err := os.CreateTemp(request.ArtifactLocalRoot, ".staged-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.WriteString(request.Identity.AttemptID)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), marker); err != nil {
		return err
	}
	directory, err := os.Open(request.ArtifactLocalRoot)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

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
