package poc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
)

type runLifecycleLock struct {
	mutex sync.Mutex
	users int
}

func (c *Coordinator) lockRunLifecycle(ctx context.Context, runID string) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.lifecycleMu.Lock()
	if c.lifecycleLocks == nil {
		c.lifecycleLocks = map[string]*runLifecycleLock{}
	}
	lock := c.lifecycleLocks[runID]
	if lock == nil {
		lock = &runLifecycleLock{}
		c.lifecycleLocks[runID] = lock
	}
	lock.users++
	c.lifecycleMu.Unlock()
	for !lock.mutex.TryLock() {
		if err := waitForAttempt(ctx, time.Millisecond); err != nil {
			c.releaseRunLifecycle(runID, lock)
			return nil, err
		}
	}
	return func() {
		lock.mutex.Unlock()
		c.releaseRunLifecycle(runID, lock)
	}, nil
}

func (c *Coordinator) releaseRunLifecycle(runID string, lock *runLifecycleLock) {
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()
	lock.users--
	if lock.users == 0 {
		delete(c.lifecycleLocks, runID)
	}
}

// DeleteManagedRun keeps termination, cleanup and metadata deletion in one lifecycle scope.
func (c *Coordinator) DeleteManagedRun(ctx context.Context, runID string) error {
	unlock, err := c.lockRunLifecycle(ctx, runID)
	if err != nil {
		return err
	}
	defer unlock()
	run, err := c.runStore.GetRun(runID, false)
	if err != nil {
		return err
	}
	if c.leaseStore != nil && run.ExecutorType != nil && *run.ExecutorType == ExecutorKubernetes && c.ownership == nil {
		for {
			if err := c.requestDeletionCancellation(runID); err != nil {
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
			return c.withRunOwnership(ctx, *token, func(owned *Coordinator, operation context.Context) error {
				return owned.deleteManagedRun(operation, runID)
			})
		}
	}
	return c.deleteManagedRun(ctx, runID)
}

func (c *Coordinator) requestDeletionCancellation(runID string) error {
	current, err := c.runStore.GetRun(runID, false)
	if err != nil {
		return err
	}
	if !IsManagedRun(current) {
		return fmt.Errorf("run %s is not managed by the coordinator", runID)
	}
	if !isManagedRunTerminal(current.State) && current.State != model.RuntimeStateCancelling {
		if err := c.runStore.TerminateRun(runID); err != nil {
			return err
		}
		c.cancelActiveRun(runID)
	}
	return nil
}

func (c *Coordinator) deleteManagedRun(ctx context.Context, runID string) error {
	if err := c.requestDeletionCancellation(runID); err != nil {
		return err
	}
	current, err := c.runStore.GetRun(runID, false)
	if err != nil {
		return err
	}
	if !isManagedRunTerminal(current.State) {
		if err := c.TerminateManagedRun(ctx, current); err != nil {
			return err
		}
		current, err = c.runStore.GetRun(runID, false)
		if err != nil {
			return err
		}
	}
	if !isManagedRunTerminal(current.State) {
		return fmt.Errorf("run %s has not confirmed termination; recovery metadata was retained", runID)
	}
	attempts, err := c.runtimeStore.ListByRun(runID)
	if err != nil {
		return err
	}
	if len(attempts) != 0 {
		for _, attempt := range attempts {
			if err := c.cleanupNeverDispatchedTask(current, attempt); err != nil {
				return fmt.Errorf("run %s retains unfinished execution cleanup; recovery metadata was not deleted: %w", runID, err)
			}
		}
	}
	if err := c.CleanupManagedRun(ctx, current); err != nil {
		return err
	}
	return c.runtimeStore.DeleteRun(runID, current.RetryGeneration)
}
