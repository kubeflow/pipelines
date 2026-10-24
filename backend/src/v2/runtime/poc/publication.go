package poc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"google.golang.org/protobuf/types/known/structpb"
)

const runtimeTaskFinalizing = model.RuntimeTaskFinalizing
const runtimeTaskFailureFinalizing = model.RuntimeTaskFailureFinalizing

type finalizedAttemptError struct{ err error }

func (e *finalizedAttemptError) Error() string { return e.err.Error() }
func (e *finalizedAttemptError) Unwrap() error { return e.err }

func (c *Coordinator) checkpointedResult(attempt *model.RuntimeTask) (*TaskExecutionResult, error) {
	task, err := c.taskStore.GetTask(attempt.TaskUUID)
	if err != nil {
		return nil, err
	}
	result := &TaskExecutionResult{Outcome: TaskOutcomeSuccess, StatusMetadata: task.StatusMetadata, OutputParameters: map[string]*structpb.Value{}}
	parameters, err := model.JSONSliceToProtoSlice(task.OutputParameters, func() *apiv2.PipelineTask_InputOutputs_IOParameter {
		return &apiv2.PipelineTask_InputOutputs_IOParameter{}
	})
	if err != nil {
		return nil, err
	}
	for _, parameter := range parameters {
		result.OutputParameters[parameter.GetParameterKey()] = parameter.GetValue()
	}
	if diagnostic, ok, err := failureDiagnostics(task.StatusMetadata); err != nil {
		return nil, err
	} else if ok {
		result.Outcome, result.Failure = diagnostic.Outcome, diagnostic.Failure
	}
	return result, nil
}

func (c *Coordinator) checkpointAttempt(ctx context.Context, item *queuedRun, executor WorkloadExecutorPlugin, attempt *model.RuntimeTask, request TaskExecutionRequest, update *model.Task) error {
	if c.artifactStore == nil || c.artifactTaskStore == nil {
		return c.runtimeStore.Checkpoint(attempt, update)
	}
	input := request.ExecutorInput
	if attempt.State == model.RuntimeTaskFailed || executor.Type() == ExecutorKubernetes {
		input = logsOnlyInput(input)
	}
	artifacts, links, err := publicationRecords(item.run, attempt.TaskUUID, request.TaskName, *attempt.AttemptUUID, input)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.runtimeStore.CheckpointArtifacts(attempt, update, artifacts, links)
}

func logsOnlyInput(input *pipelinespec.ExecutorInput) *pipelinespec.ExecutorInput {
	outputs := &pipelinespec.ExecutorInput_Outputs{Artifacts: map[string]*pipelinespec.ArtifactList{}}
	if logs := input.GetOutputs().GetArtifacts()["executor-logs"]; logs != nil {
		outputs.Artifacts["executor-logs"] = logs
	}
	return &pipelinespec.ExecutorInput{Outputs: outputs}
}

func (c *Coordinator) finishFailedAttempt(ctx context.Context, item *queuedRun, executor WorkloadExecutorPlugin, attempt *model.RuntimeTask, request TaskExecutionRequest, result *TaskExecutionResult) error {
	if attempt.State != runtimeTaskFailureFinalizing {
		if executor.Type() == ExecutorDocker || executor.Type() == ExecutorKubernetes {
			var publicationErr error
			if attempt.PublicationRetryCount >= publicationRetryLimit {
				publicationErr = fmt.Errorf("failed-attempt log publication retry budget was exhausted")
			}
			for attempt.PublicationRetryCount < publicationRetryLimit {
				attempt.PublicationRetryCount++
				if err := c.runtimeStore.Update(attempt, *attempt.AttemptUUID); err != nil {
					return pendingAttempt(err)
				}
				publicationErr = c.ensureExecutionLogs(ctx, item, executor, attempt, request)
				if publicationErr == nil {
					publicationErr = c.publishOutputArtifacts(ctx, item.run, attempt.TaskUUID, request.TaskName, *attempt.AttemptUUID, request.ArtifactLocalRoot, logsOnlyInput(request.ExecutorInput))
				}
				if publicationErr == nil {
					break
				}
				if err := waitForAttempt(ctx, time.Duration(attempt.PublicationRetryCount)*200*time.Millisecond); err != nil {
					return pendingAttempt(err)
				}
			}
			if publicationErr != nil {
				result, err := c.checkpointPublicationFailure(attempt, publicationErr)
				if err != nil {
					return err
				}
				return c.finishFailedAttempt(ctx, item, executor, attempt, request, result)
			}
		}
		attempt.State = runtimeTaskFailureFinalizing
		if err := c.runtimeStore.Update(attempt, *attempt.AttemptUUID); err != nil {
			return pendingAttempt(err)
		}
	}
	if err := cleanupExecution(ctx, executor, attempt); err != nil {
		return pendingAttempt(err)
	}
	if result.Failure != nil && result.Failure.Class == "publication" {
		task, err := c.taskStore.GetTask(attempt.TaskUUID)
		if err != nil {
			return pendingAttempt(err)
		}
		if err := c.finalizeAttempt(attempt, apiv2.PipelineTask_FAILED, task.StatusMetadata, task.OutputParameters); err != nil {
			return pendingAttempt(err)
		}
		return &finalizedAttemptError{err: workloadFailure(result)}
	}
	return workloadFailure(result)
}

func (c *Coordinator) checkpointPublicationFailure(attempt *model.RuntimeTask, publicationErr error) (*TaskExecutionResult, error) {
	task, err := c.taskStore.GetTask(attempt.TaskUUID)
	if err != nil {
		return nil, pendingAttempt(err)
	}
	result := &TaskExecutionResult{
		Outcome: TaskOutcomeError,
		Failure: &TaskFailure{Class: "publication", Message: fmt.Sprintf("publication failed after %d attempts without rerunning computation: %v", publicationRetryLimit, publicationErr)},
	}
	diagnostic, err := toJSONData(executionFailure{Outcome: result.Outcome, Failure: result.Failure})
	if err != nil {
		return nil, pendingAttempt(err)
	}
	result.StatusMetadata = mergeJSONData(task.StatusMetadata, customStatusMetadata(result.Failure.Message, map[string]any{"executionFailure": diagnostic}))
	attempt.State = runtimeTaskFailureFinalizing
	if err := c.runtimeStore.Checkpoint(attempt, &model.Task{
		UUID: attempt.TaskUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING), StatusMetadata: result.StatusMetadata,
	}); err != nil {
		return nil, pendingAttempt(err)
	}
	return result, nil
}

func publicationRecords(run *model.Run, taskID, taskName, attemptID string, input *pipelinespec.ExecutorInput) ([]*model.Artifact, []*model.ArtifactTask, error) {
	if input.GetOutputs() == nil {
		return nil, nil, nil
	}
	updated, err := loadExecutorOutputArtifacts(input.GetOutputs().GetOutputFile())
	if err != nil {
		return nil, nil, err
	}
	var artifacts []*model.Artifact
	var links []*model.ArtifactTask
	for key, list := range input.GetOutputs().GetArtifacts() {
		for index, artifact := range list.GetArtifacts() {
			if index < len(updated[key].GetArtifacts()) {
				mergeRuntimeArtifacts(updated[key].GetArtifacts()[index], artifact)
			}
			record, link := publicationRecord(run, taskID, taskName, attemptID, key, index, artifact)
			artifacts = append(artifacts, record)
			links = append(links, link)
		}
	}
	return artifacts, links, nil
}

func publicationRecord(run *model.Run, taskID, taskName, attemptID, key string, index int, artifact *pipelinespec.RuntimeArtifact) (*model.Artifact, *model.ArtifactTask) {
	identity := fmt.Sprintf("%s/%s/%d", attemptID, key, index)
	artifactID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(identity)).String()
	record := &model.Artifact{
		UUID: artifactID, Namespace: run.Namespace, Name: artifact.GetName(),
		Metadata: runtimeArtifactMetadata(artifact),
	}
	if artifact.GetUri() != "" {
		uri := artifact.GetUri()
		record.URI = &uri
	}
	return record, &model.ArtifactTask{
		UUID:       uuid.NewSHA1(uuid.NameSpaceOID, []byte(identity+"/link")).String(),
		ArtifactID: artifactID, TaskID: taskID, RunUUID: run.UUID,
		Type: model.IOType(apiv2.IOType_OUTPUT), ArtifactKey: key,
		Producer: model.JSONData{"task_name": taskName},
	}
}

func (c *Coordinator) publishAttempt(ctx context.Context, item *queuedRun, executor WorkloadExecutorPlugin, attempt *model.RuntimeTask, request TaskExecutionRequest, result *TaskExecutionResult, taskSpec *pipelinespec.PipelineTaskSpec, iteratorOutput bool) error {
	if attempt.State != runtimeTaskFinalizing {
		var publicationErr error
		if attempt.PublicationRetryCount >= publicationRetryLimit {
			publicationErr = fmt.Errorf("publication retry budget was exhausted before finalization")
		}
		for attempt.PublicationRetryCount < publicationRetryLimit {
			attempt.PublicationRetryCount++
			if err := c.runtimeStore.Update(attempt, *attempt.AttemptUUID); err != nil {
				return pendingAttempt(err)
			}
			if executor.Type() == ExecutorDocker || executor.Type() == ExecutorKubernetes {
				publicationErr = c.ensureExecutionLogs(ctx, item, executor, attempt, request)
				if publicationErr == nil {
					input := request.ExecutorInput
					if executor.Type() == ExecutorKubernetes {
						input = logsOnlyInput(input)
					}
					publicationErr = c.publishOutputArtifacts(ctx, item.run, attempt.TaskUUID, request.TaskName, *attempt.AttemptUUID, request.ArtifactLocalRoot, input)
				}
			}
			if publicationErr == nil {
				break
			}
			if err := waitForAttempt(ctx, time.Duration(attempt.PublicationRetryCount)*200*time.Millisecond); err != nil {
				return pendingAttempt(err)
			}
		}
		if attempt.PublicationRetryCount >= publicationRetryLimit && publicationErr != nil {
			result, err := c.checkpointPublicationFailure(attempt, publicationErr)
			if err != nil {
				return err
			}
			return c.finishFailedAttempt(ctx, item, executor, attempt, request, result)
		}
		attempt.State = runtimeTaskFinalizing
		if err := c.runtimeStore.Update(attempt, *attempt.AttemptUUID); err != nil {
			return pendingAttempt(err)
		}
	}
	if err := cleanupExecution(ctx, executor, attempt); err != nil {
		return pendingAttempt(err)
	}
	task, err := c.taskStore.GetTask(attempt.TaskUUID)
	if err != nil {
		return pendingAttempt(err)
	}
	if err := c.finalizeAttempt(attempt, apiv2.PipelineTask_SUCCEEDED, task.StatusMetadata, task.OutputParameters); err != nil {
		return pendingAttempt(err)
	}
	return nil
}
