package poc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"google.golang.org/protobuf/types/known/structpb"
)

const runtimeTaskFinalizing = model.RuntimeTaskFinalizing
const runtimeTaskFailureFinalizing = model.RuntimeTaskFailureFinalizing

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

func logsOnlyInput(input *pipelinespec.ExecutorInput) *pipelinespec.ExecutorInput {
	outputs := &pipelinespec.ExecutorInput_Outputs{Artifacts: map[string]*pipelinespec.ArtifactList{}}
	if logs := input.GetOutputs().GetArtifacts()["executor-logs"]; logs != nil {
		outputs.Artifacts["executor-logs"] = logs
	}
	return &pipelinespec.ExecutorInput{Outputs: outputs}
}

func (c *Coordinator) finishFailedAttempt(ctx context.Context, item *queuedRun, executor WorkloadExecutorPlugin, attempt *model.RuntimeTask, request TaskExecutionRequest, result *TaskExecutionResult) error {
	ctx = c.executionContext(ctx, request, nil, false)
	if err := cleanupExecution(ctx, executor, attempt); err != nil {
		return pendingAttempt(err)
	}
	return workloadFailure(result)
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
	artifactID := attemptArtifactID(attemptID, key, index)
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

func attemptArtifactID(attemptID, key string, index int) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s/%s/%d", attemptID, key, index))).String()
}

func (c *Coordinator) finalizeSuccessfulAttempt(ctx context.Context, executor WorkloadExecutorPlugin, attempt *model.RuntimeTask) error {
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
