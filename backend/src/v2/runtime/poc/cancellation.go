package poc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
)

type cancellationExecution struct {
	done chan struct{}
	err  error
}

func (c *Coordinator) cancelRuntimeAttempts(ctx context.Context, runID string) error {
	if c.runtimeStore == nil {
		return nil
	}
	c.cancellationMu.Lock()
	if active := c.cancellations[runID]; active != nil {
		c.cancellationMu.Unlock()
		select {
		case <-active.done:
			return active.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	active := &cancellationExecution{done: make(chan struct{})}
	if c.cancellations == nil {
		c.cancellations = map[string]*cancellationExecution{}
	}
	c.cancellations[runID] = active
	c.cancellationMu.Unlock()
	err := c.cancelRuntimeAttemptsOnce(ctx, runID)
	c.cancellationMu.Lock()
	active.err = err
	delete(c.cancellations, runID)
	close(active.done)
	c.cancellationMu.Unlock()
	return err
}

func (c *Coordinator) cancelRuntimeAttemptsOnce(ctx context.Context, runID string) error {
	attempts, err := c.runtimeStore.ListByRun(runID)
	if err != nil {
		return err
	}
	run, err := c.runStore.GetRun(runID, false)
	if err != nil {
		return err
	}
	manifest, err := ParseManifest(string(run.PipelineRuntimeManifest))
	if err != nil {
		return fmt.Errorf("load runtime for cancellation of run %s: %w", runID, err)
	}
	if manifest == nil {
		return fmt.Errorf("run %s has no runtime manifest for cancellation", runID)
	}
	executor, ok := c.executors[manifest.Executor]
	if !ok {
		return fmt.Errorf("cannot cancel run %s: executor %q is not configured", runID, manifest.Executor)
	}
	for _, attempt := range attempts {
		err := c.cancelRuntimeAttempt(ctx, run, executor, attempt)
		if errors.Is(err, storage.ErrRuntimeTaskConflict) {
			if _, getErr := c.runtimeStore.Get(attempt.TaskUUID); errors.Is(getErr, sql.ErrNoRows) {
				continue
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Coordinator) cancelRuntimeAttempt(ctx context.Context, run *model.Run, executor WorkloadExecutorPlugin, attempt *model.RuntimeTask) error {
	if attempt.AttemptUUID == nil {
		return fmt.Errorf("cannot cancel task %s: attempt identity is missing", attempt.TaskUUID)
	}
	if attempt.State != model.RuntimeTaskCancelFinalizing {
		var request TaskExecutionRequest
		if err := fromJSONData(attempt.ExecutionRequest, &request); err != nil {
			return err
		}
		var reference ExecutionReference
		if len(attempt.ExecutionReference) > 0 {
			if err := fromJSONData(attempt.ExecutionReference, &reference); err != nil {
				return err
			}
		} else if attempt.State == model.RuntimeTaskSubmitting || attempt.State == model.RuntimeTaskRunning {
			recovered, _, err := recoverAttempt(ctx, executor, request.Identity, nil)
			if err != nil {
				return err
			}
			reference = recovered
		}
		var logURI string
		var captureErr error
		if reference.Executor != "" {
			if executor.Type() == ExecutorKubernetes {
				logURI, captureErr = c.captureCancellationLogs(ctx, run, executor, attempt, reference, request)
			}
			if err := executor.Cancel(ctx, reference); err != nil {
				return pendingAttempt(err)
			}
			var err error
			attempt.ExecutionReference, err = toJSONData(reference)
			if err != nil {
				return err
			}
			if executor.Type() != ExecutorKubernetes {
				logURI, captureErr = c.captureCancellationLogs(ctx, run, executor, attempt, reference, request)
			}
		}
		c.releaseAttempt(*attempt.AttemptUUID)
		task, err := c.taskStore.GetTask(attempt.TaskUUID)
		if err != nil {
			return err
		}
		metadata := mergeJSONData(task.StatusMetadata, customStatusMetadata("run execution was canceled", map[string]any{"attemptID": *attempt.AttemptUUID}))
		if captureErr != nil {
			metadata = mergeJSONData(metadata, customStatusMetadata("run execution was canceled; logs are unavailable: "+captureErr.Error(), map[string]any{"logsUnavailable": captureErr.Error()}))
		}
		if logURI != "" {
			publicationErr := c.publishCancellationLogs(ctx, run, attempt, request, metadata)
			var unresolved *unresolvedAttempt
			if errors.As(publicationErr, &unresolved) {
				return publicationErr
			}
			if publicationErr != nil {
				metadata = mergeJSONData(metadata, customStatusMetadata("run execution was canceled; log publication failed: "+publicationErr.Error(), map[string]any{"logPublicationFailure": publicationErr.Error()}))
			} else {
				metadata = mergeJSONData(metadata, customStatusMetadata("", map[string]any{"attemptLogURI": logURI}))
			}
		}
		attempt.State = model.RuntimeTaskCancelFinalizing
		if err := c.runtimeStore.CheckpointCancellation(attempt, &model.Task{
			UUID: attempt.TaskUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING), StatusMetadata: metadata,
		}); err != nil {
			return err
		}
	}
	c.releaseAttempt(*attempt.AttemptUUID)
	if err := cleanupExecution(ctx, executor, attempt); err != nil {
		return pendingAttempt(err)
	}
	task, err := c.taskStore.GetTask(attempt.TaskUUID)
	if err != nil {
		return err
	}
	return c.finalizeAttempt(attempt, apiv2.PipelineTask_SKIPPED, task.StatusMetadata, nil)
}

func (c *Coordinator) finishCancellation(runID, message string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := c.cancelRuntimeAttempts(ctx, runID); err != nil {
		return err
	}
	if err := c.markRunTasksForState(runID, model.RuntimeStateCanceled, message); err != nil {
		return err
	}
	return c.updateRunState(runID, model.RuntimeStateCanceled, c.time.Now().Unix())
}
