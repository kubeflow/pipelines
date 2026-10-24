package poc

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type ownershipExecutor struct {
	leaseLifecycleExecutor
	waiting chan struct{}
	cancels atomic.Int32
	cleanup atomic.Int32
	submits atomic.Int32
}

func (e *ownershipExecutor) Submit(ctx context.Context, request TaskExecutionRequest) (ExecutionReference, error) {
	e.submits.Add(1)
	return e.leaseLifecycleExecutor.Submit(ctx, request)
}

func (e *ownershipExecutor) Wait(ctx context.Context, _ ExecutionReference) (TaskExecutionResult, error) {
	close(e.waiting)
	<-ctx.Done()
	return TaskExecutionResult{}, ctx.Err()
}

func (e *ownershipExecutor) Cancel(context.Context, ExecutionReference) error {
	e.cancels.Add(1)
	return nil
}

func (e *ownershipExecutor) Cleanup(context.Context, ExecutionReference) error {
	e.cleanup.Add(1)
	return nil
}

func TestOwnershipLossStopsObservationWithoutCancelingWorkload(t *testing.T) {
	lifetime, loseOwnership := context.WithCancelCause(context.Background())
	guard := &ownershipGuard{
		lifetime: lifetime, check: func(context.Context) error { return nil },
	}
	backend := &ownershipExecutor{waiting: make(chan struct{})}
	executor := ownedExecutor{WorkloadExecutorPlugin: backend, guard: guard}
	done := make(chan error, 1)
	go func() {
		_, err := executor.Wait(context.Background(), ExecutionReference{})
		done <- err
	}()
	select {
	case <-backend.waiting:
	case <-time.After(5 * time.Second):
		t.Fatal("owned executor never began observation")
	}
	lost := errors.New("lease incarnation was lost")
	loseOwnership(lost)
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("ownership loss did not stop observation")
	}
	require.ErrorIs(t, executor.Cancel(context.Background(), ExecutionReference{}), lost)
	require.ErrorIs(t, executor.Cleanup(context.Background(), ExecutionReference{}), lost)
	require.Zero(t, backend.cancels.Load(), "ownership loss must leave work for the new owner to recover")
	require.Zero(t, backend.cleanup.Load(), "ownership loss must preserve backend recovery resources")
}

func TestLeaseValidationRejectsDispatchAndCleanupBeforeRenewalNoticesLoss(t *testing.T) {
	lost := errors.New("lease generation is stale")
	guard := &ownershipGuard{
		lifetime: context.Background(), check: func(context.Context) error { return lost },
	}
	backend := &ownershipExecutor{}
	executor := ownedExecutor{WorkloadExecutorPlugin: backend, guard: guard}
	_, err := executor.Submit(context.Background(), TaskExecutionRequest{})
	require.ErrorIs(t, err, lost)
	require.ErrorIs(t, executor.Cancel(context.Background(), ExecutionReference{}), lost)
	require.ErrorIs(t, executor.Cleanup(context.Background(), ExecutionReference{}), lost)
	require.Zero(t, backend.submits.Load())
	require.Zero(t, backend.cancels.Load())
	require.Zero(t, backend.cleanup.Load())
}

func TestOwnershipLossReleasesOnlyLocalCapacityAccounting(t *testing.T) {
	coordinator := NewCoordinator(nil, nil, nil, nil, nil, nil, nil, false)
	coordinator.ownership = &ownershipGuard{attempts: map[string]struct{}{}}
	backend := &ownershipExecutor{}
	release, err := coordinator.acquireAttempt(context.Background(), backend, "attempt", false)
	require.NoError(t, err)
	dispatcher := coordinator.dispatcher(backend)
	require.Equal(t, 1, dispatcher.active)
	coordinator.ownership.releaseAttempts(coordinator)
	require.Zero(t, dispatcher.active)
	release()
	require.Zero(t, dispatcher.active, "late completion must not release capacity twice")
	require.Zero(t, backend.cancels.Load())
	require.Zero(t, backend.cleanup.Load())
}
