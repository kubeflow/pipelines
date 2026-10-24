package poc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

func writeExecutionLogs(ctx context.Context, executor WorkloadExecutorPlugin, reference ExecutionReference, request TaskExecutionRequest) (string, error) {
	reader, ok := executor.(ExecutionLogReader)
	if !ok || reference.Executor == "" {
		return "", nil
	}
	logs := request.ExecutorInput.GetOutputs().GetArtifacts()["executor-logs"].GetArtifacts()
	if len(logs) == 0 {
		return "", nil
	}
	if len(logs) != 1 {
		return "", fmt.Errorf("task %s must have exactly one executor-logs artifact", request.Identity.TaskID)
	}
	localPath, err := resolvedRuntimeArtifactLocalPath(request.ArtifactLocalRoot, dockerTempMountSourceForArtifactRoot(request.ArtifactLocalRoot), logs[0])
	if err != nil {
		return "", err
	}
	if err := ensureContainerWritableDir(filepath.Dir(localPath)); err != nil {
		return "", err
	}
	file, err := os.CreateTemp(filepath.Dir(localPath), ".kfp-attempt-logs-")
	if err != nil {
		return "", err
	}
	temporaryPath := file.Name()
	defer os.Remove(temporaryPath)
	readErr := reader.ReadLogs(ctx, reference, false, file)
	var syncErr error
	if readErr == nil {
		syncErr = file.Sync()
	}
	closeErr := file.Close()
	if err := errors.Join(readErr, syncErr, closeErr); err != nil {
		return "", err
	}
	if err := os.Rename(temporaryPath, localPath); err != nil {
		return "", err
	}
	return logs[0].GetUri(), nil
}

func (c *Coordinator) ensureExecutionLogs(ctx context.Context, item *queuedRun, executor WorkloadExecutorPlugin, attempt *model.RuntimeTask, request TaskExecutionRequest) error {
	if _, ok := executor.(ExecutionLogReader); !ok {
		return nil
	}
	logs := request.ExecutorInput.GetOutputs().GetArtifacts()["executor-logs"].GetArtifacts()
	if len(logs) == 0 {
		return nil
	}
	if len(logs) != 1 {
		return fmt.Errorf("task %s has multiple executor-logs artifacts", attempt.TaskUUID)
	}
	if request.ArtifactLocalRoot == "" {
		request.ArtifactLocalRoot = filepath.Join(c.artifactLocalRoot(item.run, ExecutorDocker), *attempt.AttemptUUID)
	}
	path, err := resolvedRuntimeArtifactLocalPath(request.ArtifactLocalRoot, dockerTempMountSourceForArtifactRoot(request.ArtifactLocalRoot), logs[0])
	if err != nil {
		return err
	}
	if info, err := os.Stat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("attempt log staging path %q is not a regular file", path)
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(attempt.ExecutionReference) == 0 {
		return fmt.Errorf("task %s has no execution reference for log publication", attempt.TaskUUID)
	}
	var reference ExecutionReference
	if err := fromJSONData(attempt.ExecutionReference, &reference); err != nil {
		return err
	}
	_, err = writeExecutionLogs(ctx, executor, reference, request)
	return err
}

func failureDiagnostics(metadata model.JSONData) (executionFailure, bool, error) {
	properties, ok := metadata["customProperties"].(map[string]interface{})
	if !ok {
		return executionFailure{}, false, nil
	}
	failure, ok := properties["executionFailure"]
	if !ok {
		return executionFailure{}, false, nil
	}
	encoded, err := toJSONData(failure)
	if err != nil {
		return executionFailure{}, false, err
	}
	var diagnostic executionFailure
	if err := fromJSONData(encoded, &diagnostic); err != nil {
		return executionFailure{}, false, err
	}
	return diagnostic, true, nil
}
