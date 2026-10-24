package poc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

type publicationCleanupExecutor struct {
	fakeExecutorLifecycle
	cleanupErr error
	cleanups   int
}

func (*publicationCleanupExecutor) Type() string { return "scripted" }
func (*publicationCleanupExecutor) Submit(context.Context, TaskExecutionRequest) (ExecutionReference, error) {
	return ExecutionReference{}, fmt.Errorf("publication must not resubmit computation")
}
func (*publicationCleanupExecutor) Wait(context.Context, ExecutionReference) (TaskExecutionResult, error) {
	return TaskExecutionResult{}, fmt.Errorf("publication must use its checkpointed result")
}
func (e *publicationCleanupExecutor) Cleanup(context.Context, ExecutionReference) error {
	e.cleanups++
	return e.cleanupErr
}

func TestPublicationExhaustionRetainsRecoveryUntilCleanupSucceeds(t *testing.T) {
	coordinator, item, preparation := attemptFixture(t)
	identity := TaskAttemptIdentity{RunID: item.run.UUID, TaskID: preparation.Task.UUID, AttemptID: "publication-attempt"}
	request := TaskExecutionRequest{Identity: identity, ExecutorInput: preparation.ExecutorInput}
	requestData, err := toJSONData(request)
	require.NoError(t, err)
	referenceData, err := toJSONData(ExecutionReference{Executor: "scripted", Version: 1, Identity: identity})
	require.NoError(t, err)
	attempt := &model.RuntimeTask{
		TaskUUID: preparation.Task.UUID, RunUUID: item.run.UUID, AttemptUUID: util.StringPointer(identity.AttemptID),
		State: model.RuntimeTaskPublishing, PublicationRetryCount: publicationRetryLimit,
		ExecutionRequest: requestData, ExecutionReference: referenceData,
	}
	require.NoError(t, coordinator.runtimeStore.Create(attempt))
	executor := &publicationCleanupExecutor{cleanupErr: fmt.Errorf("temporary cleanup failure")}
	err = coordinator.publishAttempt(context.Background(), item, executor, attempt, request, &TaskExecutionResult{Outcome: TaskOutcomeSuccess}, nil, false)
	var pending *unresolvedAttempt
	require.True(t, errors.As(err, &pending), "cleanup failure must remain recoverable")
	require.Equal(t, 1, executor.cleanups)
	stored, err := coordinator.runtimeStore.Get(attempt.TaskUUID)
	require.NoError(t, err)
	require.Equal(t, model.RuntimeTaskFailureFinalizing, stored.State)
	require.Equal(t, identity.AttemptID, *stored.AttemptUUID)
	require.NotEmpty(t, stored.ExecutionReference)
	executor.cleanupErr = nil
	_, err = coordinator.executeAttempt(context.Background(), item, executor, preparation, &pipelinespec.PipelineTaskSpec{
		RetryPolicy: &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 3, Policy: pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ALWAYS},
	}, false)
	var finalized *finalizedAttemptError
	require.True(t, errors.As(err, &finalized), "publication exhaustion must not consume computation retries")
	require.Equal(t, 2, executor.cleanups)
	task, err := coordinator.taskStore.GetTask(attempt.TaskUUID)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(apiv2.PipelineTask_FAILED), task.State)
	_, err = coordinator.runtimeStore.Get(attempt.TaskUUID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
