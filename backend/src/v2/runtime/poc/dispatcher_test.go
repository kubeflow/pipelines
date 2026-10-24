package poc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDispatcherCapacityIncludesWaitingWorkloads(t *testing.T) {
	dispatcher := &taskDispatcher{limit: 1}
	release, err := dispatcher.acquire(context.Background(), false)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		free, err := dispatcher.acquire(ctx, true)
		if free != nil {
			free()
		}
		done <- err
	}()
	require.Eventually(t, func() bool {
		dispatcher.mu.Lock()
		defer dispatcher.mu.Unlock()
		return len(dispatcher.waiting) == 1
	}, time.Second, time.Millisecond)
	dispatcher.mu.Lock()
	require.Equal(t, 1, dispatcher.active)
	dispatcher.mu.Unlock()
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	release()
	release()
	require.Zero(t, dispatcher.active, "release is idempotent and cancellation does not leak capacity")
}

func TestDispatcherRetriesHaveBoundedPriority(t *testing.T) {
	dispatcher := &taskDispatcher{limit: 1, active: 1}
	fresh := &dispatchTicket{ready: make(chan struct{})}
	dispatcher.waiting = append(dispatcher.waiting, fresh)
	for index := 0; index < 4; index++ {
		dispatcher.waiting = append(dispatcher.waiting, &dispatchTicket{retry: true, ready: make(chan struct{})})
	}
	for index := 0; index < 3; index++ {
		dispatcher.release()
		select {
		case <-fresh.ready:
			t.Fatal("eligible retries must have priority over fresh work")
		default:
		}
	}
	dispatcher.release()
	select {
	case <-fresh.ready:
	default:
		t.Fatal("retry traffic must not indefinitely starve fresh work")
	}
}

func TestDispatcherUnlimitedCapacity(t *testing.T) {
	dispatcher := &taskDispatcher{}
	for index := 0; index < 20; index++ {
		release, err := dispatcher.acquire(context.Background(), false)
		require.NoError(t, err)
		defer release()
	}
	require.Equal(t, 20, dispatcher.active)
}

func TestCancellationReleasesObservedAndRecoveredCapacityExactlyOnce(t *testing.T) {
	coordinator := NewCoordinator(nil, nil, nil, nil, nil, nil, nil, false)
	executor := &barrierExecutor{}
	coordinator.reserveRecoveredAttempt(executor, "attempt")
	release, err := coordinator.acquireAttempt(context.Background(), executor, "attempt", false)
	require.NoError(t, err)
	again, err := coordinator.acquireAttempt(context.Background(), executor, "attempt", false)
	require.NoError(t, err)
	dispatcher := coordinator.dispatcher(executor)
	require.Equal(t, 1, dispatcher.active)
	coordinator.releaseAttempt("attempt")
	require.Zero(t, dispatcher.active)
	release()
	again()
	require.Zero(t, dispatcher.active, "late observers must not release a canceled slot twice")
	require.Empty(t, coordinator.reservations)
}

func TestWorkerLockRejectsSecondWorkerAndReleasesOnClose(t *testing.T) {
	t.Setenv("KFP_V2_RUNTIME_WORKSPACE_ROOT", t.TempDir())
	first := &Coordinator{}
	require.NoError(t, first.lockInstallation())
	second := &Coordinator{}
	require.ErrorContains(t, second.lockInstallation(), "another worker")
	require.NoError(t, first.workerLock.Close())
	require.NoError(t, second.lockInstallation())
	require.NoError(t, second.workerLock.Close())
}
