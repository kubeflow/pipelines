package poc

import (
	"context"
	"fmt"
	"sort"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
)

// RetryManagedRun explicitly reopens cleaned-up failed work without replaying successful siblings.
func (c *Coordinator) RetryManagedRun(ctx context.Context, run *model.Run) error {
	if run == nil || !IsManagedRun(run) || run.ExecutorType == nil {
		return util.NewInvalidInputError("A coordinator-managed run with a persisted executor is required")
	}
	unlock, err := c.lockRunLifecycle(ctx, run.UUID)
	if err != nil {
		return err
	}
	defer unlock()
	if *run.ExecutorType != RequestedExecutor() {
		return util.NewInvalidInputError("Run %s requires executor %s; restore that worker configuration before retrying", run.UUID, *run.ExecutorType)
	}
	if c.usesRunLeases() && c.ownership == nil {
		return c.runOwnedMaintenance(ctx, run.UUID, func(owned *Coordinator) error {
			return owned.retryManagedRun(ctx, run.UUID)
		})
	}
	if err := c.retryManagedRun(ctx, run.UUID); err != nil {
		return err
	}
	if common.GetBoolConfigWithDefault(common.V2RuntimeAutoExecute, true) {
		current, err := c.runStore.GetRun(run.UUID, false)
		if err != nil {
			return err
		}
		item, err := c.rebuildQueuedRun(current)
		if err != nil {
			return err
		}
		return c.enqueueRun(ctx, item)
	}
	return nil
}

func (c *Coordinator) retryManagedRun(ctx context.Context, runID string) error {
	c.activeRunMu.Lock()
	active := c.activeRuns[runID] != nil || c.pendingRuns[runID]
	c.activeRunMu.Unlock()
	if active {
		return util.NewFailedPreconditionError(fmt.Errorf("previous coordinator is still draining"), "Run %s cleanup is still in progress; retry after it completes", runID)
	}
	run, err := c.runStore.GetRun(runID, false)
	if err != nil {
		return err
	}
	if run.State != model.RuntimeStateFailed && run.State != model.RuntimeStateCanceled {
		return util.NewInvalidInputError("Run %s must be FAILED or CANCELED before retrying", runID)
	}
	attempts, err := c.runtimeStore.ListByRun(runID)
	if err != nil {
		return err
	}
	if len(attempts) != 0 {
		return util.NewFailedPreconditionError(fmt.Errorf("retained execution or cleanup records remain"), "Run %s must finish cleanup before retrying; use the runtime cleanup command without forcing unknown workloads", runID)
	}
	item, err := c.rebuildQueuedRun(run)
	if err != nil {
		return err
	}
	executor, ok := c.executors[item.runtimeManifest.Executor]
	if !ok {
		return util.NewInvalidInputError("Executor %s is unavailable for run %s", item.runtimeManifest.Executor, runID)
	}
	if err := validateExecutor(executor, common.IsMultiUserMode()); err != nil {
		return util.NewInvalidInputError("Run %s cannot be retried: %v", runID, err)
	}
	workspaceType := ""
	if run.WorkspaceProvisioner != nil {
		workspaceType = *run.WorkspaceProvisioner
	}
	if err := validatePipelineCapabilities(item.spec, item.kubernetesSpec, executor.Capabilities(), workspaceType); err != nil {
		return util.NewInvalidInputError("Run %s cannot be retried with executor %s: %v", runID, executor.Type(), err)
	}
	tasks, _, _, err := c.taskStore.ListTasks(&model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: runID},
	}, list.EmptyOptions())
	if err != nil {
		return err
	}
	recreateWorkspace := item.kubernetesSpec.GetPipelineConfig().GetWorkspace() != nil || run.WorkspaceProvisioner != nil
	dependencies, err := materializedRetryDependencies(tasks, item.spec)
	if err != nil {
		return err
	}
	reset := tasksForManagedRetry(tasks, dependencies, recreateWorkspace)
	if len(reset) == 0 {
		return util.NewInvalidInputError("Run %s has no failed or canceled work to retry", runID)
	}
	item.runtimeManifest.WorkspaceRetry = recreateWorkspace
	if recreateWorkspace {
		item.runtimeManifest.Workspace = nil
	}
	item.runtimeManifest.RetryGeneration = run.RetryGeneration + 1
	manifest, err := item.runtimeManifest.ToJSON()
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.runtimeStore.RetryRun(runID, run.RetryGeneration, reset, model.LargeText(manifest), recreateWorkspace)
}

func tasksForManagedRetry(tasks []*model.Task, dependencies map[string][]string, workspace bool) []string {
	reset := map[string]bool{}
	invalidated := map[string]bool{}
	parents := map[string]string{}
	for _, task := range tasks {
		if task.ParentTaskUUID != nil {
			parents[task.UUID] = *task.ParentTaskUUID
		}
		if workspace || (task.State != model.TaskStatus(apiv2.PipelineTask_SUCCEEDED) && task.State != model.TaskStatus(apiv2.PipelineTask_CACHED)) {
			reset[task.UUID] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for child, parent := range parents {
			if reset[child] && !reset[parent] {
				reset[parent], changed = true, true
			}
			if invalidated[parent] && !invalidated[child] {
				reset[child], invalidated[child], changed = true, true, true
			}
		}
		for taskID, sources := range dependencies {
			for _, source := range sources {
				if reset[source] && !invalidated[taskID] {
					reset[taskID], invalidated[taskID], changed = true, true, true
				}
			}
		}
	}
	ids := make([]string, 0, len(reset))
	for id := range reset {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (c *Coordinator) reprovisionRetryWorkspace(ctx context.Context, item *queuedRun) error {
	if !item.runtimeManifest.WorkspaceRetry || item.workspace != nil {
		return nil
	}
	requirements := item.kubernetesSpec.GetPipelineConfig().GetWorkspace()
	if requirements == nil {
		return fmt.Errorf("run %s lost its workspace requirements; cannot safely retry", item.run.UUID)
	}
	provisioner, err := c.workspaceProvisioner(item.run, item.runtimeManifest.Executor, nil)
	if err != nil {
		return err
	}
	spec, err := config.WorkspacePVCSpec(requirements, c.defaultWorkspaceSpec)
	if err != nil {
		return err
	}
	workspace, err := provisioner.Provision(ctx, WorkspaceRequest{
		RunID: item.run.UUID, RunName: item.run.K8SName, Namespace: item.run.Namespace, Requirements: spec,
	})
	if err != nil {
		return err
	}
	data, err := toJSONData(workspace)
	if err != nil {
		return err
	}
	item.runtimeManifest.Workspace = workspace
	item.run.WorkspaceReference = &data
	manifest, err := item.runtimeManifest.ToJSON()
	if err != nil {
		return err
	}
	item.run.PipelineRuntimeManifest = model.LargeText(manifest)
	if err := c.updateRun(item.run); err != nil {
		return err
	}
	item.workspace = workspace
	return nil
}
