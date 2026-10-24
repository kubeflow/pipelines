package poc

import (
	"context"
	"io"
)

type fakeExecutorLifecycle struct{}

func (fakeExecutorLifecycle) Capabilities() ExecutorCapabilities {
	return ExecutorCapabilities{
		SupportsContainers: true, SupportsKubernetes: true,
		SupportedWorkspaceProvisioners: []string{WorkspaceTypeHostPath, WorkspaceTypePVC},
	}
}

func (fakeExecutorLifecycle) Capacity() ExecutorCapacity              { return ExecutorCapacity{} }
func (fakeExecutorLifecycle) CheckRequirements(context.Context) error { return nil }
func (fakeExecutorLifecycle) Start(context.Context) error             { return nil }
func (fakeExecutorLifecycle) Shutdown(context.Context) error          { return nil }
func (fakeExecutorLifecycle) Cancel(context.Context, ExecutionReference) (CancellationResult, error) {
	return CancellationResult{State: CancellationConfirmed}, nil
}
func (fakeExecutorLifecycle) Cleanup(context.Context, ExecutionReference) error {
	return nil
}
func (fakeExecutorLifecycle) Logs(_ context.Context, reference ExecutionReference) (ExecutionLogSource, error) {
	return ExecutionLogSource{Identity: reference.Identity, stream: func(context.Context, bool, io.Writer) error {
		return nil
	}}, nil
}

func fakeSubmit(request TaskExecutionRequest) (ExecutionReference, error) {
	data, err := toJSONData(request)
	return ExecutionReference{Executor: ExecutorDocker, Version: 1, Identity: request.Identity, Data: data}, err
}

func fakeWait(ctx context.Context, reference ExecutionReference, execute func(context.Context, *TaskExecutionRequest) (*TaskExecutionResult, error)) (TaskExecutionResult, error) {
	var request TaskExecutionRequest
	if err := fromJSONData(reference.Data, &request); err != nil {
		return TaskExecutionResult{}, err
	}
	result, err := execute(ctx, &request)
	if result == nil {
		return TaskExecutionResult{}, err
	}
	if result.Outcome == "" && err == nil {
		result.Outcome = TaskOutcomeSuccess
	}
	return *result, err
}

func (f *fakeDockerExecutor) Submit(_ context.Context, request TaskExecutionRequest) (ExecutionReference, error) {
	return fakeSubmit(request)
}
func (f *fakeDockerExecutor) Wait(ctx context.Context, reference ExecutionReference) (TaskExecutionResult, error) {
	return fakeWait(ctx, reference, f.ExecuteTask)
}
func (f *fakeArtifactExecutor) Submit(_ context.Context, request TaskExecutionRequest) (ExecutionReference, error) {
	return fakeSubmit(request)
}
func (f *fakeArtifactExecutor) Wait(ctx context.Context, reference ExecutionReference) (TaskExecutionResult, error) {
	result, err := fakeWait(ctx, reference, f.ExecuteTask)
	if err != nil {
		return TaskExecutionResult{}, err
	}
	return completeExecution(ctx, f, reference, result)
}
func (b *blockingExecutor) Submit(_ context.Context, request TaskExecutionRequest) (ExecutionReference, error) {
	return fakeSubmit(request)
}
func (b *blockingExecutor) Wait(ctx context.Context, reference ExecutionReference) (TaskExecutionResult, error) {
	return fakeWait(ctx, reference, b.ExecuteTask)
}
func (b *barrierExecutor) Submit(_ context.Context, request TaskExecutionRequest) (ExecutionReference, error) {
	return fakeSubmit(request)
}
func (b *barrierExecutor) Wait(ctx context.Context, reference ExecutionReference) (TaskExecutionResult, error) {
	return fakeWait(ctx, reference, b.ExecuteTask)
}
