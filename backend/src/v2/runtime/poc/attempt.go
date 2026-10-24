package poc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/grpc/codes"
)

const publicationRetryLimit = 5

func runtimeTaskNotFound(err error) bool {
	return util.IsUserErrorCodeMatch(err, codes.NotFound) || errors.Is(err, sql.ErrNoRows)
}

// unresolvedAttempt keeps an uncertain workload or unfinished publication recoverable.
type unresolvedAttempt struct{ err error }

func (e *unresolvedAttempt) Error() string { return e.err.Error() }
func (e *unresolvedAttempt) Unwrap() error { return e.err }

func pendingAttempt(err error) error {
	return &unresolvedAttempt{err: err}
}

func (c *Coordinator) executeAttempt(
	ctx context.Context, item *queuedRun, executor WorkloadExecutorPlugin,
	preparation *TaskPreparationRequest, taskSpec *pipelinespec.PipelineTaskSpec, iteratorOutput bool,
) (*TaskExecutionResult, error) {
	if c.runtimeStore == nil {
		return nil, fmt.Errorf("executor %q requires a durable runtime task store", executor.Type())
	}
	attempt, err := c.runtimeStore.Get(preparation.Task.UUID)
	if err != nil {
		if !runtimeTaskNotFound(err) {
			return nil, pendingAttempt(err)
		}
		attempt = &model.RuntimeTask{
			TaskUUID: preparation.Task.UUID, RunUUID: item.run.UUID,
			AttemptUUID: util.StringPointer(uuid.NewString()), State: model.RuntimeTaskReady,
		}
		if err := c.runtimeStore.Create(attempt); err != nil {
			return nil, pendingAttempt(err)
		}
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, pendingAttempt(err)
		}
		if attempt.NextEligibleAt != nil {
			delay := time.Duration(*attempt.NextEligibleAt-c.time.Now().Unix()) * time.Second
			if err := waitForAttempt(ctx, delay); err != nil {
				return nil, pendingAttempt(err)
			}
		}
		result, err := c.runAttempt(ctx, item, executor, preparation, attempt, taskSpec, iteratorOutput)
		if err == nil {
			return result, nil
		}
		var unresolved *unresolvedAttempt
		if errors.As(err, &unresolved) || ctx.Err() != nil {
			return nil, err
		}
		var finalized *finalizedAttemptError
		if errors.As(err, &finalized) {
			return result, err
		}
		if !shouldRetryTask(taskSpec.GetRetryPolicy(), attempt.RetryCount, result) {
			if finalizeErr := c.finalizeAttempt(attempt, apiv2.PipelineTask_FAILED, model.JSONData{"message": err.Error()}, nil); finalizeErr != nil {
				return nil, pendingAttempt(finalizeErr)
			}
			return nil, err
		}
		next := &model.RuntimeTask{
			TaskUUID: attempt.TaskUUID, RunUUID: attempt.RunUUID,
			AttemptUUID: util.StringPointer(uuid.NewString()), State: model.RuntimeTaskBackoff,
			RetryCount: attempt.RetryCount + 1,
		}
		eligible := c.time.Now().Add(taskRetryDelay(taskSpec.GetRetryPolicy(), attempt.RetryCount)).Unix()
		next.NextEligibleAt = &eligible
		if err := c.runtimeStore.BeginRetry(attempt.TaskUUID, *attempt.AttemptUUID, next); err != nil {
			return nil, pendingAttempt(err)
		}
		attempt = next
	}
}

func waitForAttempt(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func taskRetryDelay(policy *pipelinespec.PipelineTaskSpec_RetryPolicy, retryCount int) time.Duration {
	if policy == nil || policy.GetBackoffDuration() == nil {
		return 0
	}
	factor := policy.GetBackoffFactor()
	if factor <= 0 {
		factor = 2
	}
	seconds := float64(policy.GetBackoffDuration().GetSeconds()) * math.Pow(float64(factor), float64(retryCount))
	maximum := float64(time.Duration(1<<63-1) / time.Second)
	if configured := policy.GetBackoffMaxDuration().GetSeconds(); configured > 0 && float64(configured) < maximum {
		maximum = float64(configured)
	}
	if seconds > maximum || math.IsInf(seconds, 1) {
		seconds = maximum
	}
	return time.Duration(seconds) * time.Second
}

func (c *Coordinator) finalizeAttempt(attempt *model.RuntimeTask, state apiv2.PipelineTask_TaskState, metadata model.JSONData, outputs model.JSONSlice) error {
	update := &model.Task{
		UUID: attempt.TaskUUID, State: model.TaskStatus(state),
		StatusMetadata: metadata, OutputParameters: outputs, FinishedInSec: c.time.Now().Unix(),
	}
	return c.runtimeStore.Finalize(attempt.TaskUUID, *attempt.AttemptUUID, update)
}
