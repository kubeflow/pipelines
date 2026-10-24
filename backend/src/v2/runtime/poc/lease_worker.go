package poc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	backendcompiler "github.com/kubeflow/pipelines/backend/src/v2/compiler"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
)

const leaseClaimBatchSize = 8

type runLeaseProvider interface {
	RunLeaseStore() storage.RunLeaseStoreInterface
	LeaseScopedStores(storage.RunLeaseToken) (*storage.LeaseScopedStores, error)
}

type ownedRun struct {
	done chan struct{}
}

func (c *Coordinator) usesRunLeases() bool {
	return RequestedExecutor() == ExecutorKubernetes && c.leaseStore != nil
}

// PrepareRunSubmission initializes lease-managed runs before they become visible
// to workers. A true result means CreateRun must not be followed by SubmitRun.
func (c *Coordinator) PrepareRunSubmission(run *model.Run, job *pipelinespec.PipelineJob, platform *pipelinespec.SinglePlatformSpec) (bool, error) {
	if !c.usesRunLeases() {
		return false, nil
	}
	if run == nil || job == nil {
		return false, fmt.Errorf("run and pipeline job are required")
	}
	spec, err := backendcompiler.GetPipelineSpec(job)
	if err != nil {
		return false, err
	}
	executor := c.executors[RequestedExecutor()]
	if err := validateExecutor(executor, common.IsMultiUserMode()); err != nil {
		return false, err
	}
	return true, c.prepareLeasedSubmission(run, spec, platform, executor.Type())
}

func (c *Coordinator) prepareLeasedSubmission(run *model.Run, spec *pipelinespec.PipelineSpec, platform *pipelinespec.SinglePlatformSpec, executor string) error {
	if err := validateSupportedComponent(spec, spec.GetRoot()); err != nil {
		return err
	}
	if workspace := platform.GetPipelineConfig().GetWorkspace(); workspace != nil {
		if _, err := config.WorkspacePVCSpec(workspace, c.defaultWorkspaceSpec); err != nil {
			return err
		}
		provisioner, err := c.workspaceProvisioner(run, executor, nil)
		if err != nil {
			return err
		}
		run.WorkspaceProvisioner = util.StringPointer(provisioner.Type())
	}
	run.ExecutorType = util.StringPointer(executor)
	if run.PipelineRoot == "" {
		root, err := c.getPipelineRoot(run)
		if err != nil {
			return err
		}
		run.PipelineRoot = model.LargeText(root)
	}
	manifest, err := NewRuntimeManifest(executor).ToJSON()
	if err != nil {
		return err
	}
	run.PipelineRuntimeManifest = model.LargeText(manifest)
	return nil
}

func (c *Coordinator) runLeaseLoop(ctx context.Context) {
	defer c.runWorkers.Wait()
	ticker := time.NewTicker(c.leasePollInterval)
	defer ticker.Stop()
	for {
		if err := c.admitLeasedRuns(ctx); err != nil && ctx.Err() == nil {
			glog.Errorf("Runtime worker %s could not claim runs: %v", c.ownerID, err)
		}
		select {
		case <-ctx.Done():
			// Drain any concurrent startup reconciliation before waiting for its
			// already registered run goroutines.
			c.admissionMu.Lock()
			c.admissionMu.Unlock()
			return
		case <-ticker.C:
		}
	}
}

func (c *Coordinator) availableLeaseClaims() int {
	executor := c.executors[RequestedExecutor()]
	dispatcher := c.dispatcher(executor)
	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	if dispatcher.limit == 0 {
		return leaseClaimBatchSize
	}
	available := dispatcher.limit - dispatcher.active - len(dispatcher.waiting)
	if available < 0 {
		return 0
	}
	if available > leaseClaimBatchSize {
		return leaseClaimBatchSize
	}
	return available
}

func (c *Coordinator) admitLeasedRuns(ctx context.Context) error {
	c.admissionMu.Lock()
	defer c.admissionMu.Unlock()
	if !common.GetBoolConfigWithDefault(common.V2RuntimeAutoExecute, true) ||
		c.workerContext == nil || c.workerContext.Err() != nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	budget := c.availableLeaseClaims()
	candidateBudget := budget
	if budget == 0 {
		candidateBudget = leaseClaimBatchSize
	}
	preferRetry := c.claimRetryStreak < 3
	runIDs, err := c.leaseStore.ListEligible(RequestedExecutor(), candidateBudget, preferRetry)
	if err != nil {
		return err
	}
	for _, runID := range runIDs {
		if c.workerContext.Err() != nil {
			return c.workerContext.Err()
		}
		c.activeRunMu.Lock()
		existing := c.ownedRuns[runID]
		c.activeRunMu.Unlock()
		if existing != nil {
			continue
		}
		run, err := c.runStore.GetRun(runID, false)
		if storage.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		if isManagedRunTerminal(run.State) {
			continue
		}
		cleanup := run.State == model.RuntimeStateCancelling
		if !cleanup && budget == 0 {
			continue
		}
		token, err := c.leaseStore.Acquire(runID, c.ownerID, c.leaseDuration, cleanup)
		if errors.Is(err, storage.ErrRunLeaseConflict) || errors.Is(err, storage.ErrRunLeaseIneligible) {
			continue
		}
		if err != nil {
			return err
		}
		active := &ownedRun{done: make(chan struct{})}
		c.activeRunMu.Lock()
		c.ownedRuns[runID] = active
		c.activeRunMu.Unlock()
		c.runWorkers.Add(1)
		go func() {
			defer c.runWorkers.Done()
			defer close(active.done)
			defer func() {
				c.activeRunMu.Lock()
				delete(c.ownedRuns, runID)
				c.activeRunMu.Unlock()
			}()
			glog.Infof("Runtime worker %s acquired run %s generation %d", token.OwnerID, runID, token.Generation)
			err := c.withRunOwnership(c.workerContext, *token, func(owned *Coordinator, lifetime context.Context) error {
				return owned.executeLeasedRun(lifetime, runID)
			})
			if err != nil && c.workerContext.Err() == nil && !errors.Is(err, storage.ErrRunLeaseConflict) {
				glog.Errorf("Runtime worker %s stopped coordinating run %s: %v", c.ownerID, runID, err)
			}
		}()
	}
	if preferRetry {
		c.claimRetryStreak++
	} else {
		c.claimRetryStreak = 0
	}
	return nil
}

func (c *Coordinator) withRunOwnership(ctx context.Context, token storage.RunLeaseToken, operation func(*Coordinator, context.Context) error) error {
	stores, err := c.leaseProvider.LeaseScopedStores(token)
	if err != nil {
		releaseErr := c.leaseStore.Release(token)
		if releaseErr != nil && !errors.Is(releaseErr, storage.ErrRunLeaseConflict) {
			return errors.Join(err, releaseErr)
		}
		return err
	}
	lifetime, cancel := context.WithCancelCause(ctx)
	guard := &ownershipGuard{
		lifetime: lifetime, attempts: map[string]struct{}{},
		check: func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return c.leaseStore.Check(token)
		},
	}
	owned := *c
	owned.runStore, owned.taskStore, owned.runtimeStore = stores.Runs, stores.Tasks, stores.Runtime
	if c.artifactStore != nil {
		owned.artifactStore = stores.Artifacts
	}
	if c.artifactTaskStore != nil {
		owned.artifactTaskStore = stores.ArtifactTasks
	}
	owned.ownership = guard
	owned.executors = make(map[string]WorkloadExecutorPlugin, len(c.executors))
	for name, executor := range c.executors {
		owned.executors[name] = ownedExecutor{WorkloadExecutorPlugin: executor, guard: guard}
	}
	owned.workspaceProvisioners = make(map[string]WorkspaceProvisionerPlugin, len(c.workspaceProvisioners))
	for name, provisioner := range c.workspaceProvisioners {
		owned.workspaceProvisioners[name] = ownedWorkspaceProvisioner{WorkspaceProvisionerPlugin: provisioner, guard: guard}
	}
	renewed := make(chan struct{})
	go func() {
		defer close(renewed)
		c.renewRunOwnership(lifetime, cancel, token)
	}()
	result := operation(&owned, lifetime)
	if lifetime.Err() != nil {
		result = context.Cause(lifetime)
	}
	cancel(context.Canceled)
	<-renewed
	guard.releaseAttempts(c)
	releaseErr := c.leaseStore.Release(token)
	if releaseErr != nil && !errors.Is(releaseErr, storage.ErrRunLeaseConflict) {
		return errors.Join(result, fmt.Errorf("release run %s ownership: %w", token.RunUUID, releaseErr))
	}
	return result
}

func (c *Coordinator) renewRunOwnership(ctx context.Context, cancel context.CancelCauseFunc, token storage.RunLeaseToken) {
	ticker := time.NewTicker(c.leaseRenewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if err := c.leaseStore.Renew(token, c.leaseDuration); err != nil {
			glog.Warningf("Runtime worker %s lost run %s generation %d: %v", token.OwnerID, token.RunUUID, token.Generation, err)
			cancel(fmt.Errorf("renew run %s ownership: %w", token.RunUUID, err))
			return
		}
		run, err := c.runStore.GetRun(token.RunUUID, false)
		if err != nil {
			cancel(fmt.Errorf("observe owned run %s: %w", token.RunUUID, err))
			return
		}
		if run.State == model.RuntimeStateCancelling {
			c.cancelActiveRun(token.RunUUID)
		}
	}
}

func (c *Coordinator) executeLeasedRun(ctx context.Context, runID string) error {
	run, err := c.runStore.GetRun(runID, false)
	if err != nil {
		return err
	}
	if isManagedRunTerminal(run.State) {
		return nil
	}
	if run.State == model.RuntimeStateCancelling {
		return retryOwnedOperation(ctx, func() error {
			return c.finishCancellation(runID, "run cancellation was recovered by its lease holder")
		})
	}
	item, err := c.rebuildQueuedRun(run)
	if err != nil {
		return err
	}
	if len(item.runtimeManifest.Tasks) == 0 {
		if err := c.SubmitRun(ctx, run, item.job, item.kubernetesSpec); err != nil {
			return err
		}
		item, err = c.rebuildQueuedRun(run)
		if err != nil {
			return err
		}
	}
	attempts, err := c.runtimeStore.ListByRun(runID)
	if err != nil {
		return err
	}
	executor := c.executors[item.runtimeManifest.Executor]
	for _, attempt := range attempts {
		if attempt.AttemptUUID != nil && (attempt.State == model.RuntimeTaskSubmitting ||
			attempt.State == model.RuntimeTaskRunning || attempt.State == model.RuntimeTaskCanceling) {
			c.reserveRecoveredAttempt(executor, *attempt.AttemptUUID)
		}
	}
	return retryOwnedOperation(ctx, func() error { return c.executeRun(ctx, item) })
}

func retryOwnedOperation(ctx context.Context, operation func() error) error {
	for {
		err := operation()
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		if errors.Is(err, storage.ErrRunLeaseConflict) {
			return err
		}
		var unresolved *unresolvedAttempt
		if !errors.As(err, &unresolved) {
			return err
		}
		if err := waitForAttempt(ctx, time.Second); err != nil {
			return err
		}
	}
}

func (c *Coordinator) runOwnedMaintenance(ctx context.Context, runID string, operation func(*Coordinator) error) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		token, err := c.leaseStore.Acquire(runID, c.ownerID, c.leaseDuration, true)
		if errors.Is(err, storage.ErrRunLeaseConflict) {
			if err := waitForAttempt(ctx, time.Second); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		return c.withRunOwnership(ctx, *token, func(owned *Coordinator, _ context.Context) error {
			return operation(owned)
		})
	}
}

func (c *Coordinator) terminateLeasedRun(ctx context.Context, run *model.Run) error {
	current, err := c.runStore.GetRun(run.UUID, false)
	if err != nil {
		return err
	}
	if isManagedRunTerminal(current.State) {
		return nil
	}
	if current.State != model.RuntimeStateCancelling {
		if err := c.runStore.TerminateRun(run.UUID); err != nil {
			return err
		}
	}
	c.cancelActiveRun(run.UUID)
	return c.runOwnedMaintenance(ctx, run.UUID, func(owned *Coordinator) error {
		current, err := owned.runStore.GetRun(run.UUID, false)
		if err != nil {
			return err
		}
		if isManagedRunTerminal(current.State) {
			return nil
		}
		if err := owned.finishCancellation(run.UUID, "run execution was canceled"); err != nil {
			return err
		}
		return owned.CleanupManagedRun(ctx, current)
	})
}

func isManagedRunTerminal(state model.RuntimeState) bool {
	switch state {
	case model.RuntimeStateSucceeded, model.RuntimeStateFailed, model.RuntimeStateCanceled, model.RuntimeStateSkipped:
		return true
	default:
		return false
	}
}
