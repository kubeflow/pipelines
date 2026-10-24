package poc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
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
	return ExecutorCapabilities{RecoverLostTasks: true, SupportedWorkloadKinds: []WorkloadKind{WorkloadKindContainer}}
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
	require.Len(t, attempts, 1, "the uncertain workload must retain its recovery state")
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
