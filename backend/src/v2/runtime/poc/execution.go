package poc

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	corev1 "k8s.io/api/core/v1"
)

type executionFailure struct {
	Outcome TaskOutcome  `json:"outcome"`
	Failure *TaskFailure `json:"failure"`
}

func (c *Coordinator) runAttempt(
	ctx context.Context, item *queuedRun, executor WorkloadExecutorPlugin,
	preparation *TaskPreparationRequest, attempt *model.RuntimeTask,
	taskSpec *pipelinespec.PipelineTaskSpec, iteratorOutput bool,
) (*TaskExecutionResult, error) {
	if attempt.AttemptUUID == nil {
		return nil, fmt.Errorf("task %s has no durable attempt identity", attempt.TaskUUID)
	}
	var request TaskExecutionRequest
	if len(attempt.ExecutionRequest) > 0 {
		if err := fromJSONData(attempt.ExecutionRequest, &request); err != nil {
			return nil, pendingAttempt(err)
		}
	}
	if attempt.State == runtimeTaskFinalizing || attempt.State == model.RuntimeTaskFailed || attempt.State == runtimeTaskFailureFinalizing {
		result, err := c.checkpointedResult(attempt)
		if err != nil {
			return nil, pendingAttempt(err)
		}
		if attempt.State == model.RuntimeTaskFailed || attempt.State == runtimeTaskFailureFinalizing {
			return result, c.finishFailedAttempt(ctx, item, executor, attempt, request, result)
		}
		ctx = c.executionContext(ctx, request, taskSpec, iteratorOutput)
		return result, c.finalizeSuccessfulAttempt(ctx, executor, attempt)
	}
	if len(attempt.ExecutionRequest) == 0 {
		var err error
		request, err = c.prepareAttempt(ctx, item, executor, preparation, attempt)
		if err != nil {
			return c.checkpointPreparationFailure(ctx, item, executor, attempt, err)
		}
		attempt.ExecutionRequest, err = toJSONData(request)
		if err != nil {
			return nil, err
		}
		if err := c.runtimeStore.Update(attempt, *attempt.AttemptUUID); err != nil {
			return nil, pendingAttempt(err)
		}
	}
	ctx = c.executionContext(ctx, request, taskSpec, iteratorOutput)
	release, err := c.acquireAttempt(ctx, executor, *attempt.AttemptUUID, attempt.State == model.RuntimeTaskInfrastructureBackoff)
	if err != nil {
		return nil, pendingAttempt(err)
	}
	stopped := false
	defer func() {
		if stopped {
			release()
		}
	}()
	var reference ExecutionReference
	var result *TaskExecutionResult
	if attempt.State == model.RuntimeTaskSubmitting || attempt.State == model.RuntimeTaskRunning || attempt.State == model.RuntimeTaskCanceling {
		reference, result, err = recoverAttempt(ctx, executor, request.Identity, attempt.ExecutionReference)
		if err != nil {
			return nil, err
		}
	} else {
		attempt.State = model.RuntimeTaskSubmitting
		if err := c.runtimeStore.Update(attempt, *attempt.AttemptUUID); err != nil {
			return nil, pendingAttempt(err)
		}
		reference, err = executor.Submit(ctx, request)
		if err != nil {
			var rejected *ExecutionRejectedError
			if errors.As(err, &rejected) {
				result = &TaskExecutionResult{Outcome: TaskOutcomeError, Failure: &rejected.Failure}
			} else {
				submissionErr := err
				reference, result, err = recoverAttempt(ctx, executor, request.Identity, nil)
				if err != nil {
					return nil, err
				}
				if result != nil && result.Failure != nil {
					result.Failure.Message = fmt.Sprintf("submission failed (%v); %s", submissionErr, result.Failure.Message)
				}
			}
		}
	}
	if result == nil {
		attempt.ExecutionReference, err = toJSONData(reference)
		if err != nil {
			return nil, pendingAttempt(err)
		}
		attempt.State = model.RuntimeTaskRunning
		if err := c.runtimeStore.Update(attempt, *attempt.AttemptUUID); err != nil {
			return nil, pendingAttempt(err)
		}
		if executor.Type() == ExecutorKubernetes {
			if err := c.checkpointPodReference(attempt, reference); err != nil {
				return nil, pendingAttempt(err)
			}
		}
		completed, err := executor.Wait(ctx, reference)
		if err != nil {
			return nil, pendingAttempt(fmt.Errorf("observe task %s attempt %s: %w", attempt.TaskUUID, *attempt.AttemptUUID, err))
		}
		result = &completed
	}
	stopped = true
	release()
	current, err := c.runtimeStore.Get(attempt.TaskUUID)
	if err != nil {
		return nil, pendingAttempt(err)
	}
	attempt.ExecutionReference = current.ExecutionReference
	if canceled, err := c.runCancellationRequested(attempt.RunUUID); err != nil {
		return nil, pendingAttempt(err)
	} else if canceled {
		return nil, context.Canceled
	}
	if result.Outcome == "" {
		return nil, pendingAttempt(fmt.Errorf("executor %q returned no terminal outcome for task %s", executor.Type(), attempt.TaskUUID))
	}
	result.StatusMetadata = mergeJSONData(result.StatusMetadata, customStatusMetadata("", map[string]any{"attemptID": request.Identity.AttemptID}))
	if logs := request.ExecutorInput.GetOutputs().GetArtifacts()["executor-logs"].GetArtifacts(); len(logs) == 1 {
		result.StatusMetadata = mergeJSONData(result.StatusMetadata, customStatusMetadata("", map[string]any{"attemptLogURI": logs[0].GetUri()}))
	}
	task, err := c.taskStore.GetTask(attempt.TaskUUID)
	if err != nil {
		return nil, pendingAttempt(err)
	}
	metadata := mergeJSONData(task.StatusMetadata, result.StatusMetadata)
	if metadata == nil {
		metadata = model.JSONData{}
	}
	outputs := outputParametersToJSON(result.OutputParameters, taskSpec, iteratorOutput)
	attempt.State = runtimeTaskFinalizing
	if result.Outcome != TaskOutcomeSuccess {
		attempt.State = runtimeTaskFailureFinalizing
		failure, err := toJSONData(executionFailure{Outcome: result.Outcome, Failure: result.Failure})
		if err != nil {
			return nil, err
		}
		metadata = mergeJSONData(metadata, customStatusMetadata("", map[string]any{"executionFailure": failure}))
	}
	update := &model.Task{UUID: attempt.TaskUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING), OutputParameters: outputs, StatusMetadata: metadata}
	if err := c.runtimeStore.Checkpoint(attempt, update); err != nil {
		if errors.Is(err, storage.ErrRuntimeRunCanceled) {
			return nil, context.Canceled
		}
		return nil, pendingAttempt(err)
	}
	if result.Outcome != TaskOutcomeSuccess {
		return result, c.finishFailedAttempt(ctx, item, executor, attempt, request, result)
	}
	return result, c.finalizeSuccessfulAttempt(ctx, executor, attempt)
}

func (c *Coordinator) checkpointPodReference(attempt *model.RuntimeTask, reference ExecutionReference) error {
	podName, nameOK := reference.Data["podName"].(string)
	podUID, uidOK := reference.Data["podUID"].(string)
	if !nameOK || !uidOK || podName == "" || podUID == "" {
		return fmt.Errorf("kubernetes execution reference for task %s has no Pod identity", attempt.TaskUUID)
	}
	pods, err := model.ProtoSliceToJSONSlice([]*apiv2.PipelineTask_TaskPod{{
		Name: podName, Uid: podUID, Type: apiv2.PipelineTask_EXECUTOR,
	}})
	if err != nil {
		return err
	}
	return c.runtimeStore.Checkpoint(attempt, &model.Task{
		UUID: attempt.TaskUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING), Pods: pods,
	})
}

func (c *Coordinator) prepareAttempt(ctx context.Context, item *queuedRun, executor WorkloadExecutorPlugin, preparation *TaskPreparationRequest, attempt *model.RuntimeTask) (TaskExecutionRequest, error) {
	if executor.Type() == ExecutorDocker {
		preparation.ArtifactLocalRoot = filepath.Join(c.artifactLocalRoot(item.run, ExecutorDocker), *attempt.AttemptUUID)
	}
	if attempt.RetryCount > 0 || executor.Type() == ExecutorDocker {
		pipelineRoot, err := c.getPipelineRoot(item.run)
		if err != nil {
			return TaskExecutionRequest{}, err
		}
		if err := withArtifactLocalRootEnv(preparation.ArtifactLocalRoot, func() error {
			preparation.ExecutorInput.Outputs = driver.BuildProvisionedOutputs(pipelineRoot, preparation.Task.Name, preparation.ComponentSpec.GetOutputDefinitions(), *attempt.AttemptUUID, "true")
			return nil
		}); err != nil {
			return TaskExecutionRequest{}, err
		}
	}
	if preparation.Workspace != nil {
		provisioner, err := c.workspaceProvisioner(item.run, executor.Type(), preparation.Workspace)
		if err != nil {
			return TaskExecutionRequest{}, err
		}
		attachment, err := provisioner.ResolveAttachment(ctx, preparation.Workspace, ExecutionTarget{Executor: executor.Type(), Namespace: item.run.Namespace})
		if err != nil {
			return TaskExecutionRequest{}, err
		}
		preparation.Workspace = attachment
	}
	request := TaskExecutionRequest{
		Identity:  TaskAttemptIdentity{RunID: item.run.UUID, TaskID: attempt.TaskUUID, AttemptID: *attempt.AttemptUUID, Namespace: item.run.Namespace},
		Namespace: item.run.Namespace, TaskName: preparation.Task.Name,
		Container: preparation.Container, ExecutorInput: preparation.ExecutorInput,
		OutputParameterTypes: preparation.OutputParameterTypes, Workspace: preparation.Workspace,
		ArtifactLocalRoot: preparation.ArtifactLocalRoot,
	}
	if executor.Type() == ExecutorDocker {
		if err := PrepareDockerTask(ctx, preparation); err != nil {
			return TaskExecutionRequest{}, err
		}
	}
	if executor.Type() == ExecutorKubernetes {
		pod, err := buildCoordinatorExecutorPod(preparation)
		if err != nil {
			return TaskExecutionRequest{}, err
		}
		if len(pod.Spec.Containers) > 0 {
			pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env,
				corev1.EnvVar{Name: component.EnvRetryIndex, Value: strconv.Itoa(attempt.RetryCount)},
				corev1.EnvVar{Name: apiclient.KFPTaskIDEnvVar, Value: attempt.TaskUUID},
				corev1.EnvVar{Name: apiclient.KFPTaskAttemptIDEnvVar, Value: *attempt.AttemptUUID},
			)
		}
		request.Pod = pod
		request.ArtifactLocalRoot = filepath.Join(c.artifactLocalRoot(item.run, ExecutorDocker), *attempt.AttemptUUID)
	}
	return request, nil
}

func recoverAttempt(ctx context.Context, executor WorkloadExecutorPlugin, identity TaskAttemptIdentity, data model.JSONData) (ExecutionReference, *TaskExecutionResult, error) {
	recoverer, ok := executor.(LostTaskRecoverer)
	if !ok || !executor.Capabilities().RecoverLostTasks {
		return ExecutionReference{}, nil, pendingAttempt(fmt.Errorf("executor %q cannot establish whether attempt %s stopped; inspect the workload before retrying", executor.Type(), identity.AttemptID))
	}
	var reference *ExecutionReference
	if len(data) > 0 {
		reference = &ExecutionReference{}
		if err := fromJSONData(data, reference); err != nil {
			return ExecutionReference{}, nil, pendingAttempt(err)
		}
	}
	recovery, err := recoverer.Recover(ctx, identity, reference)
	if err != nil {
		return ExecutionReference{}, nil, pendingAttempt(err)
	}
	switch recovery.State {
	case RecoveryExisting:
		if recovery.Reference == nil {
			return ExecutionReference{}, nil, pendingAttempt(fmt.Errorf("executor %q recovered no reference for attempt %s", executor.Type(), identity.AttemptID))
		}
		return *recovery.Reference, nil, nil
	case RecoveryCompleted:
		if recovery.Result == nil {
			return ExecutionReference{}, nil, pendingAttempt(fmt.Errorf("executor %q recovered no durable result for attempt %s", executor.Type(), identity.AttemptID))
		}
		if recovery.Reference != nil {
			return *recovery.Reference, recovery.Result, nil
		}
		return ExecutionReference{}, recovery.Result, nil
	case RecoveryStopped:
		return ExecutionReference{}, &TaskExecutionResult{
			Outcome: TaskOutcomeError, Failure: &TaskFailure{Class: "interrupted", Message: "previous workload is confirmed stopped"},
		}, nil
	default:
		return ExecutionReference{}, nil, pendingAttempt(fmt.Errorf("executor %q cannot establish the outcome of attempt %s; work has not been resubmitted", executor.Type(), identity.AttemptID))
	}
}

func workloadFailure(result *TaskExecutionResult) error {
	if result != nil && result.Failure != nil && result.Failure.Message != "" {
		return fmt.Errorf("%s: %s", result.Failure.Class, result.Failure.Message)
	}
	return fmt.Errorf("workload failed without diagnostics")
}

func shouldRetryTask(policy *pipelinespec.PipelineTaskSpec_RetryPolicy, retryCount int, result *TaskExecutionResult) bool {
	if policy == nil || retryCount >= int(policy.GetMaxRetryCount()) || result == nil {
		return false
	}
	if result.Failure != nil && result.Failure.Fatal {
		return false
	}
	switch policy.GetPolicy() {
	case pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ALWAYS:
		return result.Outcome == TaskOutcomeFailure || result.Outcome == TaskOutcomeError
	case pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_FAILURE:
		return result.Outcome == TaskOutcomeFailure
	case pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_TRANSIENT_ERROR:
		return result.Outcome == TaskOutcomeError && result.Failure != nil && result.Failure.Transient
	default:
		return result.Outcome == TaskOutcomeError
	}
}

func cleanupExecution(ctx context.Context, executor WorkloadExecutorPlugin, attempt *model.RuntimeTask) error {
	if len(attempt.ExecutionReference) == 0 {
		return nil
	}
	var reference ExecutionReference
	if err := fromJSONData(attempt.ExecutionReference, &reference); err != nil {
		return err
	}
	return executor.Cleanup(ctx, reference)
}
