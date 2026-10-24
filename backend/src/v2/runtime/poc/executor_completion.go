package poc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"google.golang.org/protobuf/proto"
)

const executorCompletionKey = "completion"

// executorCompletion is opaque backend state, not a coordinator scheduling phase.
type executorCompletion struct {
	Result       *TaskExecutionResult `json:"result,omitempty"`
	Attempts     int                  `json:"attempts"`
	Terminal     bool                 `json:"terminal"`
	LogsRetained bool                 `json:"logsRetained"`
}

type executionIOKey struct{}

// taskExecutionIO exposes only attempt-scoped reporting and shared file transfers.
type taskExecutionIO interface {
	Stage(context.Context, TaskExecutionRequest) error
	Checkpoint(context.Context, ExecutionReference) error
	Reconcile(context.Context, *TaskExecutionResult) (*TaskExecutionResult, error)
	PublishLogs(context.Context, WorkloadExecutorPlugin, ExecutionReference, *TaskExecutionResult) error
	PublishOutputs(context.Context, *TaskExecutionResult) error
	Record(context.Context, *TaskExecutionResult, bool) error
}

func (s *workerExecutionIO) Stage(ctx context.Context, request TaskExecutionRequest) error {
	if request.Identity != s.request.Identity {
		return storage.ErrRuntimeTaskConflict
	}
	store, err := s.objectStore(ctx, request.Namespace)
	if err != nil {
		return err
	}
	return stageDockerRequest(ctx, store, request)
}

type workerExecutionIO struct {
	runtimeStore storage.RuntimeStoreInterface
	taskStore    storage.TaskStoreInterface
	request      TaskExecutionRequest
	taskSpec     *pipelinespec.PipelineTaskSpec
	iterator     bool
	artifacts    bool
	objectStore  func(context.Context, string) (component.ObjectStoreClientInterface, error)
}

func (c *Coordinator) executionContext(ctx context.Context, request TaskExecutionRequest, taskSpec *pipelinespec.PipelineTaskSpec, iterator bool) context.Context {
	service := &workerExecutionIO{
		runtimeStore: c.runtimeStore, taskStore: c.taskStore, request: request,
		taskSpec: taskSpec, iterator: iterator,
		artifacts:   c.artifactStore != nil && c.artifactTaskStore != nil,
		objectStore: c.newObjectStoreClient,
	}
	return context.WithValue(ctx, executionIOKey{}, taskExecutionIO(service))
}

func scopedExecutionIO(ctx context.Context) taskExecutionIO {
	service, _ := ctx.Value(executionIOKey{}).(taskExecutionIO)
	return service
}

func (s *workerExecutionIO) current() (*model.RuntimeTask, error) {
	attempt, err := s.runtimeStore.Get(s.request.Identity.TaskID)
	if err != nil {
		return nil, err
	}
	if attempt.AttemptUUID == nil || *attempt.AttemptUUID != s.request.Identity.AttemptID || attempt.RunUUID != s.request.Identity.RunID {
		return nil, storage.ErrRuntimeTaskConflict
	}
	return attempt, nil
}

func (s *workerExecutionIO) Checkpoint(ctx context.Context, reference ExecutionReference) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if reference.Identity != s.request.Identity {
		return storage.ErrRuntimeTaskConflict
	}
	attempt, err := s.current()
	if err != nil {
		return err
	}
	attempt.ExecutionReference, err = toJSONData(reference)
	if err != nil {
		return err
	}
	if attempt.State == model.RuntimeTaskSubmitting {
		attempt.State = model.RuntimeTaskRunning
	}
	return s.runtimeStore.Update(attempt, s.request.Identity.AttemptID)
}

func (s *workerExecutionIO) Reconcile(_ context.Context, result *TaskExecutionResult) (*TaskExecutionResult, error) {
	task, err := s.taskStore.GetTask(s.request.Identity.TaskID)
	if err != nil {
		return nil, err
	}
	return reconcileLauncherCheckpoint(task, s.request.Identity.AttemptID, result)
}

func (s *workerExecutionIO) PublishLogs(ctx context.Context, executor WorkloadExecutorPlugin, reference ExecutionReference, result *TaskExecutionResult) error {
	if len(s.request.ExecutorInput.GetOutputs().GetArtifacts()["executor-logs"].GetArtifacts()) == 0 {
		return nil
	}
	if _, err := writeExecutionLogs(ctx, executor, reference, s.request); err != nil {
		return err
	}
	store, err := s.objectStore(ctx, s.request.Namespace)
	if err != nil {
		return err
	}
	if err := uploadExecutionArtifacts(ctx, store, s.request.ArtifactLocalRoot, logsOnlyInput(s.request.ExecutorInput)); err != nil {
		return err
	}
	return s.Record(ctx, result, true)
}

func (s *workerExecutionIO) PublishOutputs(ctx context.Context, result *TaskExecutionResult) error {
	if result.Outcome != TaskOutcomeSuccess {
		return s.Record(ctx, result, true)
	}
	if s.request.Pod == nil {
		store, err := s.objectStore(ctx, s.request.Namespace)
		if err != nil {
			return err
		}
		if err := uploadExecutionArtifacts(ctx, store, s.request.ArtifactLocalRoot, s.request.ExecutorInput); err != nil {
			return err
		}
	}
	return s.Record(ctx, result, s.request.Pod != nil)
}

func (s *workerExecutionIO) Record(ctx context.Context, result *TaskExecutionResult, logsOnly bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	attempt, err := s.current()
	if err != nil {
		return err
	}
	task, err := s.taskStore.GetTask(attempt.TaskUUID)
	if err != nil {
		return err
	}
	metadata := mergeJSONData(task.StatusMetadata, result.StatusMetadata)
	metadata = mergeJSONData(metadata, customStatusMetadata("", map[string]any{"attemptID": s.request.Identity.AttemptID}))
	if result.Failure != nil {
		diagnostic, err := toJSONData(executionFailure{Outcome: result.Outcome, Failure: result.Failure})
		if err != nil {
			return err
		}
		metadata = mergeJSONData(metadata, customStatusMetadata(result.Failure.Message, map[string]any{"executionFailure": diagnostic}))
	}
	if logs := s.request.ExecutorInput.GetOutputs().GetArtifacts()["executor-logs"].GetArtifacts(); len(logs) == 1 {
		metadata = mergeJSONData(metadata, customStatusMetadata("", map[string]any{"attemptLogURI": logs[0].GetUri()}))
	}
	outputs := outputParametersToJSON(result.OutputParameters, s.taskSpec, s.iterator)
	if s.request.Pod != nil && len(task.OutputParameters) > 0 {
		outputs = task.OutputParameters
		if s.iterator {
			outputs, err = iteratorOutputParameters(outputs)
			if err != nil {
				return err
			}
		}
	}
	update := &model.Task{UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING), StatusMetadata: metadata, OutputParameters: outputs}
	if !s.artifacts {
		return s.runtimeStore.Checkpoint(attempt, update)
	}
	input := s.request.ExecutorInput
	if logsOnly {
		input = logsOnlyInput(input)
	} else if input != nil {
		input = proto.Clone(input).(*pipelinespec.ExecutorInput)
	}
	run := &model.Run{UUID: s.request.Identity.RunID, Namespace: s.request.Namespace}
	artifacts, links, err := publicationRecords(run, task.UUID, s.request.TaskName, s.request.Identity.AttemptID, input)
	if err != nil {
		return err
	}
	return s.runtimeStore.CheckpointArtifacts(attempt, update, artifacts, links)
}

func loadExecutorCompletion(reference ExecutionReference) (executorCompletion, error) {
	var checkpoint executorCompletion
	if raw, ok := reference.Data[executorCompletionKey]; ok {
		data, err := toJSONData(raw)
		if err != nil {
			return checkpoint, err
		}
		if err := fromJSONData(data, &checkpoint); err != nil {
			return checkpoint, err
		}
	}
	if checkpoint.Attempts < 0 || checkpoint.Attempts > publicationRetryLimit {
		return checkpoint, fmt.Errorf("invalid executor publication retry checkpoint")
	}
	if checkpoint.Terminal && checkpoint.Result == nil {
		return checkpoint, fmt.Errorf("terminal execution checkpoint has no task result")
	}
	return checkpoint, nil
}

func saveExecutorCompletion(ctx context.Context, service taskExecutionIO, reference *ExecutionReference, checkpoint executorCompletion) error {
	data, err := toJSONData(checkpoint)
	if err != nil {
		return err
	}
	if reference.Data == nil {
		reference.Data = model.JSONData{}
	}
	reference.Data[executorCompletionKey] = data
	return service.Checkpoint(ctx, *reference)
}

// completeExecution is owned by the executor's Wait/Recover lifecycle.
func completeExecution(ctx context.Context, executor WorkloadExecutorPlugin, reference ExecutionReference, result TaskExecutionResult) (TaskExecutionResult, error) {
	checkpoint, err := loadExecutorCompletion(reference)
	if err != nil {
		return TaskExecutionResult{}, err
	}
	if checkpoint.Terminal {
		return *checkpoint.Result, nil
	}
	service := scopedExecutionIO(ctx)
	if service == nil {
		if checkpoint.Result != nil {
			return TaskExecutionResult{}, fmt.Errorf("resuming execution publication requires an attempt-scoped reporting context")
		}
		if reference.Executor == ExecutorDocker {
			data, err := parseDockerReference(reference)
			if err != nil {
				return TaskExecutionResult{}, err
			}
			if len(data.ExecutorInput.GetOutputs().GetArtifacts()) > 0 {
				return TaskExecutionResult{}, fmt.Errorf("docker artifact publication requires an attempt-scoped reporting context")
			}
		}
		return result, nil
	}
	if checkpoint.Result == nil {
		if reference.Executor == ExecutorKubernetes {
			reconciled, err := service.Reconcile(ctx, &result)
			if err != nil {
				return TaskExecutionResult{}, err
			}
			result = *reconciled
		}
		checkpoint.Result = &result
		if err := saveExecutorCompletion(ctx, service, &reference, checkpoint); err != nil {
			return TaskExecutionResult{}, err
		}
	}
	var publicationErr error
	for checkpoint.Attempts < publicationRetryLimit {
		checkpoint.Attempts++
		if err := saveExecutorCompletion(ctx, service, &reference, checkpoint); err != nil {
			return TaskExecutionResult{}, err
		}
		publicationErr = nil
		if !checkpoint.LogsRetained {
			publicationErr = service.PublishLogs(ctx, executor, reference, checkpoint.Result)
			if publicationErr == nil {
				checkpoint.LogsRetained = true
				if err := saveExecutorCompletion(ctx, service, &reference, checkpoint); err != nil {
					return TaskExecutionResult{}, err
				}
			}
		}
		if publicationErr == nil {
			publicationErr = service.PublishOutputs(ctx, checkpoint.Result)
		}
		if publicationErr == nil {
			checkpoint.Terminal = true
			if err := saveExecutorCompletion(ctx, service, &reference, checkpoint); err != nil {
				return TaskExecutionResult{}, err
			}
			return *checkpoint.Result, nil
		}
		var invalid *invalidDeclaredOutput
		if errors.As(publicationErr, &invalid) {
			failed := *checkpoint.Result
			failed.Outcome = TaskOutcomeFailure
			failed.Failure = &TaskFailure{Class: "output-contract", Message: invalid.Error()}
			checkpoint.Result, checkpoint.Terminal = &failed, true
			if err := service.Record(ctx, &failed, true); err != nil {
				return TaskExecutionResult{}, err
			}
			if err := saveExecutorCompletion(ctx, service, &reference, checkpoint); err != nil {
				return TaskExecutionResult{}, err
			}
			return failed, nil
		}

		if err := waitForAttempt(ctx, time.Duration(checkpoint.Attempts)*200*time.Millisecond); err != nil {
			return TaskExecutionResult{}, err
		}
	}
	message := "executor publication retry budget was exhausted"
	if publicationErr != nil {
		message += ": " + publicationErr.Error()
	}
	failed := *checkpoint.Result
	failed.Outcome = TaskOutcomeError
	failed.Failure = &TaskFailure{Class: "publication", Message: message, Fatal: true}
	checkpoint.Result, checkpoint.Terminal = &failed, true
	if err := service.Record(ctx, &failed, true); err != nil {
		return TaskExecutionResult{}, err
	}
	if err := saveExecutorCompletion(ctx, service, &reference, checkpoint); err != nil {
		return TaskExecutionResult{}, err
	}
	return failed, nil
}

func resumeExecutorCompletion(ctx context.Context, executor WorkloadExecutorPlugin, reference ExecutionReference) (TaskExecutionResult, bool, error) {
	checkpoint, err := loadExecutorCompletion(reference)
	if err != nil {
		return TaskExecutionResult{}, true, err
	}
	if checkpoint.Result == nil {
		return TaskExecutionResult{}, false, nil
	}
	result, err := completeExecution(ctx, executor, reference, *checkpoint.Result)
	return result, true, err
}

func retainCleanupLogs(ctx context.Context, executor WorkloadExecutorPlugin, reference ExecutionReference) error {
	checkpoint, err := loadExecutorCompletion(reference)
	if err != nil {
		return err
	}
	if checkpoint.Result == nil || checkpoint.LogsRetained {
		return nil
	}
	service := scopedExecutionIO(ctx)
	if service == nil {
		return fmt.Errorf("cleanup requires an attempt-scoped reporting context to retain execution logs")
	}
	if err := service.PublishLogs(ctx, executor, reference, checkpoint.Result); err != nil {
		return fmt.Errorf("retain execution logs before cleanup: %w", err)
	}
	checkpoint.LogsRetained = true
	return saveExecutorCompletion(ctx, service, &reference, checkpoint)
}
