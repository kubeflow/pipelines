package poc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

type recoveringIterationExecutor struct {
	fakeExecutorLifecycle
	mu          sync.Mutex
	submissions map[int]int
	resumed     bool
	firstFails  bool
}

func (*recoveringIterationExecutor) Type() string { return "scripted" }
func (*recoveringIterationExecutor) Capabilities() ExecutorCapabilities {
	return ExecutorCapabilities{
		RecoverLostTasks: true, SupportsContainers: true,
		SupportedWorkspaceProvisioners: []string{WorkspaceTypeHostPath},
	}
}
func (e *recoveringIterationExecutor) Submit(_ context.Context, request TaskExecutionRequest) (ExecutionReference, error) {
	index := int(request.ExecutorInput.GetInputs().GetParameterValues()["item"].GetNumberValue())
	e.mu.Lock()
	e.submissions[index]++
	e.mu.Unlock()
	return fakeSubmit(request)
}
func (e *recoveringIterationExecutor) Wait(_ context.Context, reference ExecutionReference) (TaskExecutionResult, error) {
	var request TaskExecutionRequest
	if err := fromJSONData(reference.Data, &request); err != nil {
		return TaskExecutionResult{}, err
	}
	index := int(request.ExecutorInput.GetInputs().GetParameterValues()["item"].GetNumberValue())
	e.mu.Lock()
	defer e.mu.Unlock()
	if index == 0 && e.firstFails {
		return TaskExecutionResult{Outcome: TaskOutcomeFailure, Failure: &TaskFailure{Class: "program", Message: "first iteration failed"}}, nil
	}
	if index == 1 && !e.resumed {
		return TaskExecutionResult{}, fmt.Errorf("lost observation of the second iteration")
	}
	return TaskExecutionResult{Outcome: TaskOutcomeSuccess}, nil
}
func (*recoveringIterationExecutor) Recover(_ context.Context, _ TaskAttemptIdentity, reference *ExecutionReference) (RecoveryResult, error) {
	if reference == nil {
		return RecoveryResult{State: RecoveryUnknown}, nil
	}
	return RecoveryResult{State: RecoveryExisting, Reference: reference}, nil
}

func iterationFixture(t *testing.T) (*Coordinator, *queuedRun, RuntimeTask, *pipelinespec.PipelineTaskSpec, *pipelinespec.ComponentSpec) {
	t.Helper()
	coordinator, item, preparation := attemptFixture(t)
	item.spec = &pipelinespec.PipelineSpec{}
	require.NoError(t, protojson.Unmarshal([]byte(`{
		"components":{"component":{"executorLabel":"executor"}},
		"deploymentSpec":{"executors":{"executor":{"container":{"image":"test"}}}}
	}`), item.spec))
	task := &pipelinespec.PipelineTaskSpec{}
	require.NoError(t, protojson.Unmarshal([]byte(`{
		"parameterIterator":{"itemInput":"item","items":{"raw":"[0,1]"}}
	}`), task))
	return coordinator, item, RuntimeTask{
		TaskID: preparation.Task.UUID, Name: preparation.Task.Name, ScopePath: preparation.Task.ScopePath,
	}, task, item.spec.Components["component"]
}

func TestIterationFailureDoesNotHideUnresolvedWorkload(t *testing.T) {
	coordinator, item, loop, spec, component := iterationFixture(t)
	executor := &recoveringIterationExecutor{submissions: map[int]int{}, firstFails: true}
	err := coordinator.executeParameterIteratorTask(context.Background(), item, executor, loop, spec, component, loop.ScopePath, nil)
	var unresolved *unresolvedAttempt
	require.True(t, errors.As(err, &unresolved), "a known failure must not authorize cleanup while another iteration is unresolved")
	attempts, err := coordinator.runtimeStore.ListByRun(item.run.UUID)
	require.NoError(t, err)
	require.Len(t, attempts, 2, "both the unfinished loop and uncertain workload retain scheduling state")
	var unresolvedAttempts int
	for _, attempt := range attempts {
		if attempt.AttemptUUID != nil {
			unresolvedAttempts++
			require.NotEmpty(t, attempt.ExecutionReference)
		}
	}
	require.Equal(t, 1, unresolvedAttempts, "the uncertain workload must retain exactly its original attempt")
}

func TestFailedCompositeIterationFinalizesItsFrameworkSchedulingRow(t *testing.T) {
	coordinator, item, loop, task, _ := iterationFixture(t)
	require.NoError(t, protojson.Unmarshal([]byte(`{
		"parameterIterator":{"itemInput":"item","items":{"raw":"[0]"}}
	}`), task))
	component := &pipelinespec.ComponentSpec{}
	require.NoError(t, protojson.Unmarshal([]byte(`{"dag":{"tasks":{"leaf":{"componentRef":{"name":"leaf"}}}}}`), component))
	item.spec.Components["leaf"] = item.spec.Components["component"]
	item.spec.Components["component"] = component
	executor := &scriptedExecutor{outcomes: []TaskOutcome{TaskOutcomeFailure}}
	err := coordinator.executeParameterIteratorTask(context.Background(), item, executor, loop, task, component, loop.ScopePath, nil)
	require.Error(t, err)
	var unresolved *unresolvedAttempt
	require.False(t, errors.As(err, &unresolved))
	remaining, err := coordinator.runtimeStore.ListByRun(item.run.UUID)
	require.NoError(t, err)
	require.Len(t, remaining, 1, "only the caller-owned outer loop row may remain; known-failed iterations must be retryable")
	require.Equal(t, loop.TaskID, remaining[0].TaskUUID)
	require.Equal(t, model.RuntimeTaskWaiting, remaining[0].State)
}

func TestShorterRetriedLoopFinalizesObsoleteNeverStartedIterations(t *testing.T) {
	coordinator, item, loop, task, component := iterationFixture(t)
	executor := &recoveringIterationExecutor{resumed: true, submissions: map[int]int{}}
	require.NoError(t, coordinator.executeParameterIteratorTask(context.Background(), item, executor, loop, task, component, loop.ScopePath, nil))
	_, err := coordinator.updateTaskState(loop.TaskID, apiv2.PipelineTask_SUCCEEDED, nil, nil)
	require.NoError(t, err)
	children, err := coordinator.taskStore.GetChildTasks(loop.TaskID)
	require.NoError(t, err)
	require.Len(t, children, 2)
	var obsoleteID string
	for _, child := range children {
		index, err := materializedIterationIndex(child)
		require.NoError(t, err)
		if index == 1 {
			obsoleteID = child.UUID
		}
	}
	tasks, _, _, err := coordinator.taskStore.ListTasks(&model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: item.run.UUID},
	}, list.EmptyOptions())
	require.NoError(t, err)
	var ids []string
	for _, current := range tasks {
		ids = append(ids, current.UUID)
	}
	kind := "scripted"
	item.run.ExecutorType, item.run.State = &kind, model.RuntimeStateFailed
	manifest, err := item.runtimeManifest.ToJSON()
	require.NoError(t, err)
	item.run.PipelineRuntimeManifest = model.LargeText(manifest)
	require.NoError(t, coordinator.runStore.UpdateRun(item.run))
	require.NoError(t, coordinator.runtimeStore.RetryRun(item.run.UUID, 0, ids, model.LargeText(manifest), false))
	require.NoError(t, protojson.Unmarshal([]byte(`{
		"parameterIterator":{"itemInput":"item","items":{"raw":"[0]"}}
	}`), task))
	require.NoError(t, coordinator.executeParameterIteratorTask(context.Background(), item, executor, loop, task, component, loop.ScopePath, nil))
	_, err = coordinator.updateTaskState(loop.TaskID, apiv2.PipelineTask_SUCCEEDED, nil, nil)
	require.NoError(t, err)
	remaining, err := coordinator.runtimeStore.ListByRun(item.run.UUID)
	require.NoError(t, err)
	require.Empty(t, remaining, "obsolete reset iterations cannot outlive a completed loop")
	obsolete, err := coordinator.taskStore.GetTask(obsoleteID)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(apiv2.PipelineTask_SKIPPED), obsolete.State)
	require.NotEmpty(t, obsolete.StateHistory, "old successful execution diagnostics must remain historical")
}

func TestIterationRecoveryDoesNotReplayCompletedContainer(t *testing.T) {
	coordinator, item, loop, spec, component := iterationFixture(t)
	executor := &recoveringIterationExecutor{submissions: map[int]int{}}
	err := coordinator.executeParameterIteratorTask(context.Background(), item, executor, loop, spec, component, loop.ScopePath, nil)
	var unresolved *unresolvedAttempt
	require.True(t, errors.As(err, &unresolved))
	executor.mu.Lock()
	executor.resumed = true
	executor.mu.Unlock()
	require.NoError(t, coordinator.executeParameterIteratorTask(context.Background(), item, executor, loop, spec, component, loop.ScopePath, nil))
	require.Equal(t, 1, executor.submissions[0], "successful computation must not be replayed during loop recovery")
	require.Equal(t, 1, executor.submissions[1], "the unfinished attempt must be recovered, not resubmitted")
}
