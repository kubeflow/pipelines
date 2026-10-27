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
	if reference.Executor == "" {
		return "", nil
	}
	source, err := executor.Logs(ctx, reference)
	if err != nil {
		return "", err
	}
	return writeExecutionLogSource(ctx, source, request)
}

func writeExecutionLogSource(ctx context.Context, source ExecutionLogSource, request TaskExecutionRequest) (string, error) {
	if source.Identity != request.Identity {
		return "", fmt.Errorf("execution log source does not match task attempt %s", request.Identity.AttemptID)
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
	readErr := source.Stream(ctx, false, file)
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
