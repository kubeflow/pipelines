package poc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type scriptedExecutor struct {
	fakeExecutorLifecycle
	mu       sync.Mutex
	outcomes []TaskOutcome
	attempts []TaskAttemptIdentity
}

type setupFailureExecutor struct{ scriptedExecutor }

func (*setupFailureExecutor) Type() string { return ExecutorDocker }

func (*scriptedExecutor) Type() string { return "scripted" }

func (s *scriptedExecutor) Submit(_ context.Context, request TaskExecutionRequest) (ExecutionReference, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts = append(s.attempts, request.Identity)
	return fakeSubmit(request)
}

func (s *scriptedExecutor) Wait(context.Context, ExecutionReference) (TaskExecutionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	outcome := s.outcomes[len(s.attempts)-1]
	result := TaskExecutionResult{Outcome: outcome}
	if outcome != TaskOutcomeSuccess {
		result.Failure = &TaskFailure{Class: "program", Message: "reported workload failure"}
	}
	return result, nil
}

func attemptFixture(t *testing.T) (*Coordinator, *queuedRun, *TaskPreparationRequest) {
	t.Helper()
	db, dialect, err := storage.NewFakeDB()
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	clock := util.NewFakeTimeForEpoch()
	runs := storage.NewRunStore(db, clock, dialect)
	tasks := storage.NewTaskStore(db, clock, util.NewUUIDGenerator(), dialect)
	run, err := runs.CreateRun(&model.Run{
		UUID: "attempt-run", Namespace: "ns", K8SName: "attempt-run",
		RunDetails: model.RunDetails{State: model.RuntimeStateRunning},
	})
	require.NoError(t, err)
	coordinator := NewCoordinator(runs, tasks, nil, nil, clock, nil, nil, false)
	task, err := coordinator.createTask(run, nil, "task", "task", apiv2.PipelineTask_RUNTIME, "root.task", nil)
	require.NoError(t, err)
	return coordinator, &queuedRun{run: run, runtimeManifest: NewRuntimeManifest("scripted")}, &TaskPreparationRequest{
		Run: run, Task: task,
		Container:     &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "test"},
		ComponentSpec: &pipelinespec.ComponentSpec{},
		ExecutorInput: &pipelinespec.ExecutorInput{Outputs: &pipelinespec.ExecutorInput_Outputs{}},
	}
}

func TestAttemptRetryRetainsTaskIdentityAndChangesAttempt(t *testing.T) {
	coordinator, item, preparation := attemptFixture(t)
	executor := &scriptedExecutor{outcomes: []TaskOutcome{TaskOutcomeFailure, TaskOutcomeSuccess}}
	taskSpec := &pipelinespec.PipelineTaskSpec{RetryPolicy: &pipelinespec.PipelineTaskSpec_RetryPolicy{
		MaxRetryCount: 1, Policy: pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_FAILURE,
	}}
	_, err := coordinator.executeAttempt(context.Background(), item, executor, preparation, taskSpec, false)
	require.NoError(t, err)
	require.Len(t, executor.attempts, 2)
	require.Equal(t, executor.attempts[0].TaskID, executor.attempts[1].TaskID)
	require.NotEqual(t, executor.attempts[0].AttemptID, executor.attempts[1].AttemptID)
	task, err := coordinator.taskStore.GetTask(preparation.Task.UUID)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), task.State)
	_, err = coordinator.runtimeStore.Get(task.UUID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestSetupFailureUsesComputationRetryPolicy(t *testing.T) {
	coordinator, item, preparation := attemptFixture(t)
	preparation.Container = nil
	executor := &setupFailureExecutor{}
	_, err := coordinator.executeAttempt(context.Background(), item, executor, preparation, &pipelinespec.PipelineTaskSpec{
		RetryPolicy: &pipelinespec.PipelineTaskSpec_RetryPolicy{
			MaxRetryCount: 1, Policy: pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_ERROR,
		},
	}, false)
	require.Error(t, err)
	require.Empty(t, executor.attempts, "failed preparation must never submit a workload")
	task, err := coordinator.taskStore.GetTask(preparation.Task.UUID)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(apiv2.PipelineTask_FAILED), task.State)
	history, err := json.Marshal(task.StateHistory)
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(history), "Previous execution attempt diagnostics"), "one allowed setup retry must be consumed")
}

func TestPublicationRecoveryDoesNotExecuteWorkloadAgain(t *testing.T) {
	coordinator, item, preparation := attemptFixture(t)
	executor := &scriptedExecutor{}
	request := TaskExecutionRequest{
		Identity:      TaskAttemptIdentity{RunID: item.run.UUID, TaskID: preparation.Task.UUID, AttemptID: "durable-attempt"},
		ExecutorInput: preparation.ExecutorInput,
	}
	data, err := toJSONData(request)
	require.NoError(t, err)
	attempt := &model.RuntimeTask{
		TaskUUID: preparation.Task.UUID, RunUUID: item.run.UUID,
		AttemptUUID: util.StringPointer(request.Identity.AttemptID),
		State:       model.RuntimeTaskPublishing, ExecutionRequest: data,
	}
	require.NoError(t, coordinator.runtimeStore.Create(attempt))
	outputs := outputParametersToJSON(map[string]*structpb.Value{"value": structpb.NewNumberValue(7)}, nil, false)
	require.NoError(t, coordinator.runtimeStore.Checkpoint(attempt, &model.Task{
		UUID: attempt.TaskUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING), OutputParameters: outputs,
	}))
	result, err := coordinator.executeAttempt(context.Background(), item, executor, preparation, &pipelinespec.PipelineTaskSpec{}, false)
	require.NoError(t, err)
	require.Empty(t, executor.attempts)
	require.Equal(t, float64(7), result.OutputParameters["value"].GetNumberValue())
}

func TestRetryPolicyClassification(t *testing.T) {
	tests := []struct {
		name      string
		policy    pipelinespec.PipelineTaskSpec_RetryPolicy_Policy
		outcome   TaskOutcome
		transient bool
		want      bool
	}{
		{"default excludes program failure", pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_UNSPECIFIED, TaskOutcomeFailure, false, false},
		{"default retries setup error", pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_UNSPECIFIED, TaskOutcomeError, false, true},
		{"failure retries program failure", pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_FAILURE, TaskOutcomeFailure, false, true},
		{"failure excludes setup error", pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_FAILURE, TaskOutcomeError, false, false},
		{"transient excludes permanent error", pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_TRANSIENT_ERROR, TaskOutcomeError, false, false},
		{"transient includes known transient", pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_TRANSIENT_ERROR, TaskOutcomeError, true, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := &TaskExecutionResult{Outcome: test.outcome, Failure: &TaskFailure{Transient: test.transient}}
			policy := &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 2, Policy: test.policy}
			require.Equal(t, test.want, shouldRetryTask(policy, 0, result))
			require.False(t, shouldRetryTask(policy, 2, result), "retry budget must be enforced")
			policy.MaxRetryCount = 0
			require.False(t, shouldRetryTask(policy, 0, result), "zero means no retries")
		})
	}
	require.False(t, shouldRetryTask(nil, 0, nil))
}

func TestUnresolvedAttemptIsNotAWorkloadFailure(t *testing.T) {
	err := pendingAttempt(context.Canceled)
	var pending *unresolvedAttempt
	require.True(t, errors.As(err, &pending))
	require.ErrorIs(t, err, context.Canceled)
}

func TestExistingAttemptResumesBeforeCacheHit(t *testing.T) {
	coordinator, item, preparation := attemptFixture(t)
	item.spec = &pipelinespec.PipelineSpec{}
	require.NoError(t, protojson.Unmarshal([]byte(`{
		"components":{"component":{"executorLabel":"executor"}},
		"deploymentSpec":{"executors":{"executor":{"container":{"image":"test"}}}}
	}`), item.spec))
	component := item.spec.Components["component"]
	taskSpec := &pipelinespec.PipelineTaskSpec{}
	require.NoError(t, protojson.Unmarshal([]byte(`{"cachingOptions":{"enableCache":true}}`), taskSpec))
	input, _, err := coordinator.buildExecutorInput(item, preparation.Task.ScopePath, taskSpec, component, nil)
	require.NoError(t, err)
	preparation.ExecutorInput = input
	fingerprint, _, err := coordinator.lookupCachedTask(component, taskSpec, preparation.Container, preparation.ExecutorInput, item.run.Namespace, nil)
	require.NoError(t, err)
	_, err = coordinator.taskStore.CreateTask(&model.Task{
		UUID: "cached-task", RunUUID: item.run.UUID, Namespace: item.run.Namespace,
		Name: "cached", ScopePath: "root.cached", Fingerprint: fingerprint,
		State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED),
	})
	require.NoError(t, err)
	data, err := toJSONData(TaskExecutionRequest{
		Identity:      TaskAttemptIdentity{RunID: item.run.UUID, TaskID: preparation.Task.UUID, AttemptID: "existing-attempt"},
		ExecutorInput: preparation.ExecutorInput,
	})
	require.NoError(t, err)
	require.NoError(t, coordinator.runtimeStore.Create(&model.RuntimeTask{
		TaskUUID: preparation.Task.UUID, RunUUID: item.run.UUID,
		AttemptUUID: util.StringPointer("existing-attempt"),
		State:       model.RuntimeTaskPublishing, ExecutionRequest: data,
	}))
	err = coordinator.executeRuntimeTask(context.Background(), item, &scriptedExecutor{}, RuntimeTask{
		TaskID: preparation.Task.UUID, Name: preparation.Task.Name, ScopePath: preparation.Task.ScopePath,
	}, taskSpec, component, nil, false)
	require.NoError(t, err)
	task, err := coordinator.taskStore.GetTask(preparation.Task.UUID)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), task.State, "an existing attempt must finish publication instead of taking a cache hit")
	_, err = coordinator.runtimeStore.Get(task.UUID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
