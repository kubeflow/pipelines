package poc

import (
	"context"
	"errors"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"google.golang.org/protobuf/types/known/structpb"
)

func (c *Coordinator) resumeRuntimeTask(
	ctx context.Context, item *queuedRun, executor WorkloadExecutorPlugin,
	task *model.Task, runtimeTask RuntimeTask, taskSpec *pipelinespec.PipelineTaskSpec,
	componentSpec *pipelinespec.ComponentSpec, parameterValues map[string]*structpb.Value, iteratorOutput bool,
) (bool, error) {
	if c.runtimeStore == nil {
		return false, nil
	}
	attempt, err := c.runtimeStore.Get(task.UUID)
	if runtimeTaskNotFound(err) {
		return false, nil
	}
	if err != nil {
		return true, pendingAttempt(err)
	}
	var request TaskExecutionRequest
	if err := fromJSONData(attempt.ExecutionRequest, &request); err != nil {
		return true, pendingAttempt(err)
	}
	if len(attempt.ExecutionRequest) == 0 {
		request.Container, err = containerForComponent(item.spec, componentSpec)
		if err != nil {
			return true, err
		}
		request.ExecutorInput, request.OutputParameterTypes, err = c.buildExecutorInput(item, runtimeTask.ScopePath, taskSpec, componentSpec, parameterValues)
		if err != nil {
			return true, err
		}
		request.ArtifactLocalRoot = c.artifactLocalRoot(item.run, executor.Type())
		if driver.NeedsWorkspaceMount(request.ExecutorInput) {
			request.Workspace = item.workspace
		}
	}
	kubernetesConfig, err := loadKubernetesExecutorConfig(componentSpec, item.kubernetesSpec)
	if err != nil {
		return true, err
	}
	objectStore, err := c.newObjectStoreClient(ctx, item.run.Namespace)
	if err != nil {
		return true, err
	}
	preparation := &TaskPreparationRequest{
		Run: item.run, Task: task, Container: request.Container, ExecutorInput: request.ExecutorInput,
		OutputParameterTypes: request.OutputParameterTypes, Workspace: request.Workspace,
		ArtifactLocalRoot: request.ArtifactLocalRoot, ComponentSpec: componentSpec, TaskSpec: taskSpec,
		ParentTaskID: runtimeTask.ParentTaskID, PipelineName: item.spec.GetPipelineInfo().GetName(),
		TaskDisplayName: taskDisplayName(taskSpec), KubernetesExecutorConfig: kubernetesConfig,
		MLPipelineTLSEnabled: c.mlPipelineTLSEnabled, ObjectStore: objectStore,
	}
	if executor.Type() == ExecutorKubernetes {
		preparation.ResolverRunTasks, preparation.ResolverParentTask, err = c.buildResolverTaskDetails(item.runtimeManifest, runtimeTask.ParentTaskID)
		if err != nil {
			return true, err
		}
	}
	return true, c.executePreparedTask(ctx, item, executor, preparation, taskSpec, iteratorOutput)
}

func (c *Coordinator) executePreparedTask(ctx context.Context, item *queuedRun, executor WorkloadExecutorPlugin, preparation *TaskPreparationRequest, taskSpec *pipelinespec.PipelineTaskSpec, iteratorOutput bool) error {
	_, err := c.executeAttempt(ctx, item, executor, preparation, taskSpec, iteratorOutput)
	if err == nil || ctx.Err() != nil {
		return err
	}
	var unresolved *unresolvedAttempt
	if errors.As(err, &unresolved) {
		return err
	}
	c.recordTaskExecutionError(item.run.UUID, preparation.Task.UUID, err)
	return err
}
