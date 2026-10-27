package poc

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PruneOptions controls offline terminal-resource reconciliation.
type PruneOptions struct {
	KeepWorkspaces bool
}

// PruneResult counts completed work, not estimated reclaimed bytes.
type PruneResult struct {
	RunsExamined       int `json:"runsExamined"`
	RunsSkipped        int `json:"runsSkipped"`
	RunsCleaned        int `json:"runsCleaned"`
	AttemptsFinalized  int `json:"attemptsFinalized"`
	WorkspacesRetained int `json:"workspacesRetained"`
	Failures           int `json:"failures"`
}

// Prune cleans only terminal managed runs without starting a worker or canceling
// workloads. Docker requires exclusive local installation ownership; Kubernetes
// mutations and backend operations use the existing per-run lease fences.
func (c *Coordinator) Prune(ctx context.Context, options PruneOptions) (PruneResult, error) {
	result := PruneResult{}
	if c.runtimeStore == nil {
		return result, fmt.Errorf("cleanup requires durable runtime storage")
	}
	if c.workerCancel != nil || c.workerLock != nil {
		return result, fmt.Errorf("stop this coordinator's worker before cleanup")
	}
	executor, ok := c.executors[RequestedExecutor()]
	if !ok {
		return result, fmt.Errorf("cleanup executor %q is not configured", RequestedExecutor())
	}
	if err := validateExecutor(executor, common.IsMultiUserMode()); err != nil {
		return result, err
	}
	switch executor.Type() {
	case ExecutorDocker:
		if err := c.lockInstallation(); err != nil {
			return result, err
		}
		defer func() {
			_ = c.workerLock.Close()
			c.workerLock = nil
		}()
	case ExecutorKubernetes:
		if !c.usesRunLeases() || c.leaseProvider == nil {
			return result, fmt.Errorf("kubernetes cleanup requires lease-scoped runtime stores")
		}
	default:
		return result, fmt.Errorf("cleanup supports only Docker and Kubernetes executors")
	}
	if err := executor.CheckRequirements(ctx); err != nil {
		return result, err
	}
	opts, err := list.NewOptions(&model.Run{}, 100, "id", nil)
	if err != nil {
		return result, err
	}
	opts.SkipCount = true
	var failures []error
	for {
		runs, _, next, err := c.runStore.ListRuns(&model.FilterContext{}, opts, false)
		if err != nil {
			return result, errors.Join(append(failures, err)...)
		}
		for _, candidate := range runs {
			if err := ctx.Err(); err != nil {
				return result, errors.Join(append(failures, err)...)
			}
			result.RunsExamined++
			if !isManagedRunTerminal(candidate.State) {
				result.RunsSkipped++
				continue
			}
			// ListRuns intentionally omits manifests. Never infer ownership from
			// an empty list projection or remove resources using it.
			run, err := c.runStore.GetRun(candidate.UUID, false)
			if err == nil && (!IsManagedRun(run) || run.ExecutorType == nil || *run.ExecutorType != executor.Type()) {
				result.RunsSkipped++
				continue
			}
			if err == nil {
				if c.usesRunLeases() {
					token, claimErr := c.leaseStore.Acquire(run.UUID, c.ownerID, c.leaseDuration, true)
					err = claimErr
					if err == nil {
						err = c.withRunOwnership(ctx, *token, func(owned *Coordinator, lifetime context.Context) error {
							return owned.pruneRun(lifetime, run.UUID, options, &result)
						})
					}
				} else {
					err = c.pruneRun(ctx, run.UUID, options, &result)
				}
			}
			if err != nil {
				result.Failures++
				failures = append(failures, fmt.Errorf("run %s: %w; recovery references retained; retry cleanup after resolving the error", candidate.UUID, err))
			}
		}
		if next == "" {
			return result, errors.Join(failures...)
		}
		opts, err = list.NewOptionsFromToken(next, 100)
		if err != nil {
			return result, errors.Join(append(failures, err)...)
		}
		opts.SkipCount = true
	}
}

func (c *Coordinator) pruneRun(ctx context.Context, runID string, options PruneOptions, result *PruneResult) error {
	run, err := c.runStore.GetRun(runID, false)
	if err != nil {
		return err
	}
	if !isManagedRunTerminal(run.State) || !IsManagedRun(run) {
		return fmt.Errorf("run is no longer terminal and managed; refusing cleanup")
	}
	manifest, err := ParseManifest(string(run.PipelineRuntimeManifest))
	if err != nil {
		return err
	}
	if manifest == nil || manifest.Kind != runtimeManifestKind || manifest.Mode != RuntimeModeCoordinator ||
		manifest.APIVersion != runtimeManifestAPIVersion || manifest.Executor != RequestedExecutor() ||
		run.ExecutorType == nil || *run.ExecutorType != manifest.Executor {
		return fmt.Errorf("run has no consistent supported runtime manifest and executor identity")
	}
	attempts, err := c.runtimeStore.ListByRun(runID)
	if err != nil {
		return err
	}
	var failures []error
	for _, attempt := range attempts {
		if err := c.pruneAttempt(ctx, run, manifest.Executor, attempt); err != nil {
			failures = append(failures, fmt.Errorf("task %s: %w", attempt.TaskUUID, err))
		} else {
			result.AttemptsFinalized++
		}
	}
	if len(failures) != 0 {
		return errors.Join(failures...)
	}
	// No workspace or staging may disappear while an unresolved task still
	// needs it, even if a run was marked terminal prematurely.
	remaining, err := c.runtimeStore.ListByRun(runID)
	if err != nil {
		return err
	}
	if len(remaining) != 0 {
		return fmt.Errorf("%d runtime tasks still require recovery", len(remaining))
	}
	for _, entry := range manifest.Tasks {
		task, err := c.taskStore.GetTask(entry.TaskID)
		if err != nil {
			return err
		}
		if task.RunUUID != runID || !isTerminalTaskState(task.State) {
			return fmt.Errorf("task %s has no confirmed terminal history", entry.TaskID)
		}
	}
	if run.WorkspaceReference != nil {
		var workspace WorkspaceHandle
		if err := fromJSONData(*run.WorkspaceReference, &workspace); err != nil {
			return err
		}
		if manifest.Workspace != nil && !reflect.DeepEqual(manifest.Workspace, &workspace) {
			return fmt.Errorf("workspace references disagree; refusing cleanup")
		}
		manifest.Workspace = &workspace
	}
	if manifest.Workspace != nil {
		if manifest.Workspace.Metadata["runID"] != runID ||
			(manifest.Workspace.Type == WorkspaceTypePVC && manifest.Workspace.Metadata["namespace"] != run.Namespace) {
			return fmt.Errorf("workspace belongs to a different run or namespace; refusing cleanup")
		}
	}
	if options.KeepWorkspaces {
		if manifest.Workspace != nil || run.WorkspaceProvisioner != nil {
			result.WorkspacesRetained++
		}
	} else {
		if manifest.Workspace == nil && run.WorkspaceProvisioner != nil && *run.WorkspaceProvisioner == WorkspaceTypePVC {
			manifest.Workspace, err = (pvcWorkspaceProvisioner{coreClient: c.kubernetesCoreClient}).recoverForCleanup(ctx, WorkspaceRequest{
				RunID: runID, RunName: run.K8SName, Namespace: run.Namespace,
			})
			if err != nil {
				return err
			}
		}
		if manifest.Workspace != nil {
			if err := c.cleanupWorkspaceForRun(ctx, run, manifest); err != nil {
				return err
			}
		}
		if err := c.confirmPrunedWorkspace(ctx, manifest.Workspace); err != nil {
			return err
		}
	}
	if err := c.cleanupRunStaging(ctx, runID, manifest.Executor); err != nil {
		return err
	}
	result.RunsCleaned++
	return nil
}

func (c *Coordinator) pruneAttempt(ctx context.Context, run *model.Run, executorType string, attempt *model.RuntimeTask) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if attempt.AttemptUUID == nil {
		return c.cleanupNeverDispatchedTask(run, attempt)
	}
	if len(attempt.ExecutionRequest) == 0 || len(attempt.ExecutionReference) == 0 {
		return fmt.Errorf("attempt outcome or execution identity is unconfirmed")
	}
	var request TaskExecutionRequest
	var reference ExecutionReference
	if err := fromJSONData(attempt.ExecutionRequest, &request); err != nil {
		return err
	}
	if err := fromJSONData(attempt.ExecutionReference, &reference); err != nil {
		return err
	}
	if err := validateAttempt(reference.Identity); err != nil {
		return err
	}
	if reference.Executor != executorType || reference.Identity != request.Identity ||
		reference.Identity.RunID != run.UUID || reference.Identity.TaskID != attempt.TaskUUID ||
		reference.Identity.AttemptID != *attempt.AttemptUUID || attempt.RunUUID != run.UUID ||
		request.Namespace != run.Namespace || (reference.Identity.Namespace != "" && reference.Identity.Namespace != run.Namespace) {
		return fmt.Errorf("execution reference does not match the current run/task/attempt identity")
	}
	task, err := c.taskStore.GetTask(attempt.TaskUUID)
	if err != nil {
		return err
	}
	if task.RunUUID != run.UUID || task.Namespace != run.Namespace {
		return fmt.Errorf("public task belongs to a different run or namespace")
	}
	checkpoint, err := loadExecutorCompletion(reference)
	if err != nil {
		return err
	}
	var state apiv2.PipelineTask_TaskState
	switch attempt.State {
	case model.RuntimeTaskFinalizing, model.RuntimeTaskFailureFinalizing, model.RuntimeTaskFailed:
		if !checkpoint.Terminal || checkpoint.Result == nil || !checkpoint.LogsRetained {
			return fmt.Errorf("terminal outcome and durable logs are not confirmed")
		}
		switch checkpoint.Result.Outcome {
		case TaskOutcomeSuccess:
			if attempt.State != model.RuntimeTaskFinalizing {
				return fmt.Errorf("terminal result disagrees with finalization phase")
			}
			state = apiv2.PipelineTask_SUCCEEDED
		case TaskOutcomeFailure, TaskOutcomeError:
			if attempt.State == model.RuntimeTaskFinalizing {
				return fmt.Errorf("terminal result disagrees with finalization phase")
			}
			state = apiv2.PipelineTask_FAILED
		default:
			return fmt.Errorf("terminal outcome is unknown")
		}
	case model.RuntimeTaskCancelFinalizing:
		logs := request.ExecutorInput.GetOutputs().GetArtifacts()["executor-logs"].GetArtifacts()
		properties, _ := task.StatusMetadata["customProperties"].(map[string]any)
		if len(logs) != 1 || logs[0].GetUri() == "" || properties["attemptID"] != reference.Identity.AttemptID ||
			properties["attemptLogURI"] != logs[0].GetUri() ||
			properties["logsUnavailable"] != nil || properties["logPublicationFailure"] != nil {
			return fmt.Errorf("canceled attempt's durable logs are not confirmed")
		}
		state = apiv2.PipelineTask_SKIPPED
	default:
		return fmt.Errorf("phase %s still requires worker recovery, not cleanup", attempt.State)
	}
	if isTerminalTaskState(task.State) && task.State != model.TaskStatus(state) {
		return fmt.Errorf("terminal task history disagrees with retained finalization outcome")
	}
	// Built-in Cleanup verifies backend identity and refuses live workloads.
	// Do not use Cancel or Recover here: pruning cannot change an outcome.
	if err := cleanupExecution(ctx, c.executors[executorType], attempt); err != nil {
		return err
	}
	finished := task.FinishedInSec
	if finished == 0 {
		finished = c.time.Now().Unix()
	}
	return c.runtimeStore.Finalize(attempt.TaskUUID, *attempt.AttemptUUID, &model.Task{
		UUID: task.UUID, State: model.TaskStatus(state), FinishedInSec: finished,
		StatusMetadata: task.StatusMetadata, OutputParameters: task.OutputParameters,
	})
}

func (c *Coordinator) confirmPrunedWorkspace(ctx context.Context, workspace *WorkspaceHandle) error {
	if workspace == nil || workspace.Type != WorkspaceTypePVC {
		return nil
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		pvc, err := c.kubernetesCoreClient.GetClientSet().CoreV1().PersistentVolumeClaims(workspace.Metadata["namespace"]).
			Get(ctx, workspace.Metadata["pvcName"], metav1.GetOptions{})
		if apierrors.IsNotFound(err) || (err == nil && string(pvc.UID) != workspace.Metadata["pvcUID"]) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := waitForPluginPoll(ctx); err != nil {
			return err
		}
	}
}
