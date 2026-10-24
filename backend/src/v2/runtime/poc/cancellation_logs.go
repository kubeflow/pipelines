package poc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

func (c *Coordinator) captureCancellationLogs(ctx context.Context, run *model.Run, executor WorkloadExecutorPlugin, attempt *model.RuntimeTask, reference ExecutionReference, request TaskExecutionRequest) (string, error) {
	if _, ok := executor.(ExecutionLogReader); !ok {
		return "", nil
	}
	logs := request.ExecutorInput.GetOutputs().GetArtifacts()["executor-logs"].GetArtifacts()
	if len(logs) != 1 {
		return "", nil
	}
	if request.ArtifactLocalRoot == "" {
		request.ArtifactLocalRoot = filepath.Join(c.artifactLocalRoot(run, ExecutorDocker), *attempt.AttemptUUID)
	}
	path, err := resolvedRuntimeArtifactLocalPath(request.ArtifactLocalRoot, dockerTempMountSourceForArtifactRoot(request.ArtifactLocalRoot), logs[0])
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
		return logs[0].GetUri(), nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return writeExecutionLogs(ctx, executor, reference, request)
}

func (c *Coordinator) publishCancellationLogs(ctx context.Context, run *model.Run, attempt *model.RuntimeTask, request TaskExecutionRequest, metadata model.JSONData) error {
	attempt.State = model.RuntimeTaskCanceling
	var publicationErr error
	if attempt.PublicationRetryCount >= publicationRetryLimit {
		publicationErr = errors.New("canceled-attempt log publication budget was exhausted")
	}
	for attempt.PublicationRetryCount < publicationRetryLimit {
		attempt.PublicationRetryCount++
		if err := c.runtimeStore.CheckpointCancellation(attempt, &model.Task{
			UUID: attempt.TaskUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING), StatusMetadata: metadata,
		}); err != nil {
			return pendingAttempt(err)
		}
		root := request.ArtifactLocalRoot
		if root == "" {
			root = filepath.Join(c.artifactLocalRoot(run, ExecutorDocker), *attempt.AttemptUUID)
		}
		publicationErr = c.publishOutputArtifacts(ctx, run, attempt.TaskUUID, request.TaskName, *attempt.AttemptUUID, root, logsOnlyInput(request.ExecutorInput))
		if publicationErr == nil {
			return nil
		}
		if err := waitForAttempt(ctx, time.Duration(attempt.PublicationRetryCount)*200*time.Millisecond); err != nil {
			return pendingAttempt(err)
		}
	}
	return publicationErr
}
