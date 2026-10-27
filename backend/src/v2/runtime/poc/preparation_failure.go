package poc

import (
	"context"
	"errors"
	"net"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (c *Coordinator) checkpointPreparationFailure(ctx context.Context, item *queuedRun, executor WorkloadExecutorPlugin, attempt *model.RuntimeTask, preparationErr error) (*TaskExecutionResult, error) {
	if ctx.Err() != nil {
		return nil, pendingAttempt(ctx.Err())
	}
	var networkError net.Error
	transient := errors.As(preparationErr, &networkError) && networkError.Timeout()
	transient = transient || apierrors.IsTimeout(preparationErr) || apierrors.IsServerTimeout(preparationErr) || apierrors.IsServiceUnavailable(preparationErr)
	result := &TaskExecutionResult{
		Outcome: TaskOutcomeError,
		Failure: &TaskFailure{Class: "setup", Message: preparationErr.Error(), Transient: transient},
	}
	diagnostic, err := toJSONData(executionFailure{Outcome: result.Outcome, Failure: result.Failure})
	if err != nil {
		return nil, err
	}
	attempt.State = model.RuntimeTaskFailed
	if err := c.runtimeStore.Checkpoint(attempt, &model.Task{
		UUID: attempt.TaskUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING),
		StatusMetadata: customStatusMetadata(preparationErr.Error(), map[string]any{"executionFailure": diagnostic}),
	}); err != nil {
		return nil, pendingAttempt(err)
	}
	return result, c.finishFailedAttempt(ctx, item, executor, attempt, TaskExecutionRequest{}, result)
}
