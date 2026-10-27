package poc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
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
	if attempt.AttemptUUID == nil {
		attempt.AttemptUUID = util.StringPointer(uuid.NewString())
		attempt.State = model.RuntimeTaskReady
		if err := c.runtimeStore.Update(attempt, ""); err != nil {
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
		if !shouldRetryTask(taskSpec.GetRetryPolicy(), attempt.RetryCount, result) {
			task, getErr := c.taskStore.GetTask(attempt.TaskUUID)
			if getErr != nil {
				return nil, pendingAttempt(getErr)
			}
			metadata := mergeJSONData(task.StatusMetadata, model.JSONData{"message": err.Error()})
			if finalizeErr := c.finalizeAttempt(attempt, apiv2.PipelineTask_FAILED, metadata, nil); finalizeErr != nil {
				return nil, pendingAttempt(finalizeErr)
			}
			return nil, err
		}
		next := &model.RuntimeTask{
			TaskUUID: attempt.TaskUUID, RunUUID: attempt.RunUUID,
			AttemptUUID: util.StringPointer(uuid.NewString()), State: model.RuntimeTaskBackoff,
			RetryCount: attempt.RetryCount + 1,
		}
		infrastructure := result != nil && result.Outcome == TaskOutcomeError
		if infrastructure {
			next.State = model.RuntimeTaskInfrastructureBackoff
		}
		delay := retryEligibilityDelay(taskSpec.GetRetryPolicy(), attempt.RetryCount, infrastructure, rand.Float64())
		deadline := c.time.Now().Add(delay)
		eligible := deadline.Unix()
		if delay > 0 && deadline.Nanosecond() != 0 {
			eligible++
		}
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
	if policy.GetBackoffDuration().AsDuration() == 0 {
		return 0
	}
	factor := policy.GetBackoffFactor()
	if factor <= 0 {
		factor = 2
	}
	duration := float64(policy.GetBackoffDuration().AsDuration()) * math.Pow(float64(factor), float64(retryCount))
	maximum := time.Duration(1<<63 - 1)
	if configured := policy.GetBackoffMaxDuration().AsDuration(); configured > 0 && configured < maximum {
		maximum = configured
	}
	if duration >= float64(maximum) || math.IsInf(duration, 1) {
		return maximum
	}
	return time.Duration(duration)
}

// retryEligibilityDelay adds bounded jitter without shortening the pipeline's backoff.
func retryEligibilityDelay(policy *pipelinespec.PipelineTaskSpec_RetryPolicy, retryCount int, infrastructure bool, jitter float64) time.Duration {
	configured := taskRetryDelay(policy, retryCount)
	if !infrastructure {
		return configured
	}
	exponent := min(max(retryCount, 0), 6)
	window := min(time.Second<<exponent, time.Minute)
	infrastructureDelay := window/2 + time.Duration(float64(window/2)*jitter)
	return max(configured, infrastructureDelay)
}

func (c *Coordinator) finalizeAttempt(attempt *model.RuntimeTask, state apiv2.PipelineTask_TaskState, metadata model.JSONData, outputs model.JSONSlice) error {
	update := &model.Task{
		UUID: attempt.TaskUUID, State: model.TaskStatus(state),
		StatusMetadata: metadata, OutputParameters: outputs, FinishedInSec: c.time.Now().Unix(),
	}
	return c.runtimeStore.Finalize(attempt.TaskUUID, *attempt.AttemptUUID, update)
}
