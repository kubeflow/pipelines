package poc

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

type recordingExecutionIO struct {
	stage         func(context.Context, TaskExecutionRequest) error
	lastReference ExecutionReference
	checkpointErr error
	outputErr     error
	logErr        error
	outputs       int
	logs          int
	records       int
}

func (s *recordingExecutionIO) Stage(ctx context.Context, request TaskExecutionRequest) error {
	if s.stage != nil {
		return s.stage(ctx, request)
	}
	return nil
}

func (s *recordingExecutionIO) Checkpoint(_ context.Context, reference ExecutionReference) error {
	if s.checkpointErr != nil {
		return s.checkpointErr
	}
	data, err := toJSONData(reference)
	if err != nil {
		return err
	}
	return fromJSONData(data, &s.lastReference)
}

func (*recordingExecutionIO) Reconcile(_ context.Context, result *TaskExecutionResult) (*TaskExecutionResult, error) {
	return result, nil
}

func (s *recordingExecutionIO) PublishLogs(context.Context, WorkloadExecutorPlugin, ExecutionReference, *TaskExecutionResult) error {
	s.logs++
	return s.logErr
}

func (s *recordingExecutionIO) PublishOutputs(context.Context, *TaskExecutionResult) error {
	s.outputs++
	return s.outputErr
}

func (s *recordingExecutionIO) Record(context.Context, *TaskExecutionResult, bool) error {
	s.records++
	return nil
}

func completionContext(service taskExecutionIO) context.Context {
	return context.WithValue(context.Background(), executionIOKey{}, service)
}

func publicationFixture(t *testing.T) (*Coordinator, storage.ArtifactStoreInterface, storage.ArtifactTaskStoreInterface) {
	t.Helper()
	db, dialect, err := storage.NewFakeDB()
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	clock := util.NewFakeTimeForEpoch()
	generator := util.NewUUIDGenerator()
	runs := storage.NewRunStore(db, clock, dialect)
	tasks := storage.NewTaskStore(db, clock, generator, dialect)
	artifacts := storage.NewArtifactStore(db, clock, generator, dialect)
	links := storage.NewArtifactTaskStore(db, generator, dialect)
	return NewCoordinator(runs, tasks, artifacts, links, clock, nil, nil, false), artifacts, links
}

func publishTestExecutorArtifacts(t *testing.T, coordinator *Coordinator, run *model.Run, taskName, root string, input *pipelinespec.ExecutorInput) error {
	t.Helper()
	run.State = model.RuntimeStateRunning
	_, err := coordinator.runStore.CreateRun(run)
	require.NoError(t, err)
	task, err := coordinator.runtimeStore.CreateTask(&model.Task{
		UUID: "task-1", RunUUID: run.UUID, Namespace: run.Namespace, Name: taskName,
		State: model.TaskStatus(apiv2.PipelineTask_RUNNING),
	})
	require.NoError(t, err)
	request := TaskExecutionRequest{
		Identity:  TaskAttemptIdentity{RunID: run.UUID, TaskID: task.UUID, AttemptID: "attempt-1"},
		Namespace: run.Namespace, TaskName: taskName, ArtifactLocalRoot: root, ExecutorInput: input,
	}
	data, err := toJSONData(request)
	require.NoError(t, err)
	require.NoError(t, coordinator.runtimeStore.Update(&model.RuntimeTask{
		TaskUUID: task.UUID, RunUUID: run.UUID, AttemptUUID: &request.Identity.AttemptID,
		State: model.RuntimeTaskRunning, ExecutionRequest: data,
	}, ""))
	ctx := coordinator.executionContext(context.Background(), request, nil, false)
	result, err := completeExecution(ctx, dockerExecutor{}, ExecutionReference{
		Executor: ExecutorDocker, Version: 1, Identity: request.Identity,
	}, TaskExecutionResult{Outcome: TaskOutcomeSuccess})
	if err == nil {
		require.Equal(t, TaskOutcomeSuccess, result.Outcome)
	}
	return err
}

func TestExecutorPublicationExhaustionIsDurableFatalAndNotAWorkloadRetry(t *testing.T) {
	service := &recordingExecutionIO{outputErr: errors.New("object store unavailable")}
	reference := ExecutionReference{Executor: "test", Version: 1, Identity: TaskAttemptIdentity{
		RunID: "run", TaskID: "task", AttemptID: "attempt",
	}}
	result, err := completeExecution(completionContext(service), &barrierExecutor{}, reference, TaskExecutionResult{Outcome: TaskOutcomeSuccess})
	require.NoError(t, err)
	require.Equal(t, TaskOutcomeError, result.Outcome)
	require.True(t, result.Failure.Fatal)
	require.Equal(t, "publication", result.Failure.Class)
	require.Equal(t, publicationRetryLimit, service.outputs)
	checkpoint, err := loadExecutorCompletion(service.lastReference)
	require.NoError(t, err)
	require.True(t, checkpoint.Terminal)
	require.True(t, checkpoint.LogsRetained)
	require.Equal(t, publicationRetryLimit, checkpoint.Attempts)

	restarted := &recordingExecutionIO{}
	recovered, found, err := resumeExecutorCompletion(completionContext(restarted), &barrierExecutor{}, service.lastReference)
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, recovered.Failure.Fatal)
	require.Zero(t, restarted.outputs, "a restart must not reset transfer retries or replay successful computation")
}

func TestInvalidDeclaredOutputIsNotRetriedAsAStorageOutage(t *testing.T) {
	service := &recordingExecutionIO{outputErr: &invalidDeclaredOutput{err: fmt.Errorf("required output is missing")}}
	result, err := completeExecution(completionContext(service), &barrierExecutor{}, ExecutionReference{}, TaskExecutionResult{Outcome: TaskOutcomeSuccess})
	require.NoError(t, err)
	require.Equal(t, TaskOutcomeFailure, result.Outcome)
	require.False(t, result.Failure.Fatal)
	require.Equal(t, "output-contract", result.Failure.Class)
	require.Equal(t, 1, service.outputs)
}

func TestExecutorPublicationRejectsStaleReportingBeforeTransfers(t *testing.T) {
	service := &recordingExecutionIO{checkpointErr: storage.ErrRunLeaseConflict}
	result, err := completeExecution(completionContext(service), &barrierExecutor{}, ExecutionReference{}, TaskExecutionResult{Outcome: TaskOutcomeSuccess})
	require.ErrorIs(t, err, storage.ErrRunLeaseConflict)
	require.Empty(t, result.Outcome, "failure to observe/report is not a known task outcome")
	require.Zero(t, service.outputs)
	require.Zero(t, service.logs)
}

func TestCleanupRetainsBackendUntilLogsAreDurable(t *testing.T) {
	service := &recordingExecutionIO{logErr: errors.New("log store unavailable")}
	reference := ExecutionReference{Data: map[string]interface{}{}}
	checkpoint := executorCompletion{
		Terminal: true, Result: &TaskExecutionResult{Outcome: TaskOutcomeError, Failure: &TaskFailure{Class: "publication", Fatal: true}},
	}
	require.NoError(t, saveExecutorCompletion(completionContext(service), service, &reference, checkpoint))
	require.ErrorContains(t, retainCleanupLogs(completionContext(service), &barrierExecutor{}, reference), "log store unavailable")
	service.logErr = nil
	require.NoError(t, retainCleanupLogs(completionContext(service), &barrierExecutor{}, reference))
	checkpoint, err := loadExecutorCompletion(service.lastReference)
	require.NoError(t, err)
	require.True(t, checkpoint.LogsRetained)
	require.True(t, checkpoint.Result.Failure.Fatal, "cleanup must not change the task outcome")
}

func TestDockerResultDoesNotEmbedLogBytesInMetadata(t *testing.T) {
	engine := &fakeDockerEngine{}
	executor := dockerExecutor{command: engine.run}
	reference, err := executor.Submit(context.Background(), dockerTestRequest(t))
	require.NoError(t, err)
	result, err := executor.Wait(context.Background(), reference)
	require.NoError(t, err)
	data, err := toJSONData(result)
	require.NoError(t, err)
	require.NotContains(t, fmt.Sprint(data), "durable stdout")
	require.NotContains(t, fmt.Sprint(data), "durable stderr")
}
