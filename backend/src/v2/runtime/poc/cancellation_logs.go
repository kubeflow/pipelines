package poc

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

func (c *Coordinator) captureUnstartedCancellationLogs(ctx context.Context, run *model.Run, attempt *model.RuntimeTask, request TaskExecutionRequest) (string, error) {
	if request.ArtifactLocalRoot == "" {
		request.ArtifactLocalRoot = filepath.Join(c.artifactLocalRoot(run, ExecutorDocker), *attempt.AttemptUUID)
	}
	source := ExecutionLogSource{
		Identity: request.Identity,
		stream: func(_ context.Context, _ bool, destination io.Writer) error {
			_, err := io.WriteString(destination, "Task canceled before execution; no component output was produced.\n")
			return err
		},
	}
	return writeExecutionLogSource(ctx, source, request)
}

func (c *Coordinator) captureCancellationLogs(ctx context.Context, run *model.Run, executor WorkloadExecutorPlugin, attempt *model.RuntimeTask, reference ExecutionReference, request TaskExecutionRequest) (string, error) {
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
	var reference ExecutionReference
	if err := fromJSONData(attempt.ExecutionReference, &reference); err != nil {
		return err
	}
	checkpoint, err := loadExecutorCompletion(reference)
	if err != nil {
		return err
	}
	var publicationErr error
	for retry := 0; retry < publicationRetryLimit; retry++ {
		data, err := toJSONData(checkpoint)
		if err != nil {
			return err
		}
		if reference.Data == nil {
			reference.Data = model.JSONData{}
		}
		reference.Data[executorCompletionKey] = data
		attempt.ExecutionReference, err = toJSONData(reference)
		if err != nil {
			return err
		}
		if err := c.runtimeStore.CheckpointCancellation(attempt, &model.Task{
			UUID: attempt.TaskUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING), StatusMetadata: metadata,
		}); err != nil {
			return pendingAttempt(err)
		}
		root := request.ArtifactLocalRoot
		if root == "" {
			root = filepath.Join(c.artifactLocalRoot(run, ExecutorDocker), *attempt.AttemptUUID)
		}
		publicationErr = c.publishCanceledLogArtifacts(ctx, run, attempt.TaskUUID, request.TaskName, *attempt.AttemptUUID, root, logsOnlyInput(request.ExecutorInput))
		if publicationErr == nil {
			checkpoint.LogsRetained = true
			data, err := toJSONData(checkpoint)
			if err != nil {
				return err
			}
			reference.Data[executorCompletionKey] = data
			attempt.ExecutionReference, err = toJSONData(reference)
			if err != nil {
				return err
			}
			if err := c.runtimeStore.CheckpointCancellation(attempt, &model.Task{
				UUID: attempt.TaskUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING), StatusMetadata: metadata,
			}); err != nil {
				return pendingAttempt(err)
			}
			return nil
		}
		if err := waitForAttempt(ctx, time.Duration(retry+1)*200*time.Millisecond); err != nil {
			return pendingAttempt(err)
		}
	}
	return publicationErr
}
