// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func handoffWaitResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("handoff waiter did not finish")
		return nil
	}
}

func TestScheduleWriterWaiterSharesSuccess(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var checks atomic.Int32
	wait := newScheduleWriterHandoffWaiter(func(ctx context.Context) error {
		if checks.Add(1) == 1 {
			close(entered)
		}
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, time.Second, time.Millisecond)
	results := make(chan error, 12)
	go func() { results <- wait(context.Background()) }()
	<-entered
	for i := 1; i < cap(results); i++ {
		go func() { results <- wait(context.Background()) }()
	}
	close(release)
	for i := 0; i < cap(results); i++ {
		require.NoError(t, handoffWaitResult(t, results))
	}
	require.NoError(t, wait(context.Background()))
	require.EqualValues(t, 1, checks.Load())
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, wait(canceled), context.Canceled)
	require.EqualValues(t, 1, checks.Load())
}

func TestScheduleWriterWaiterLeaderCancellationAllowsFollower(t *testing.T) {
	entered := make(chan struct{})
	var checks atomic.Int32
	wait := newScheduleWriterHandoffWaiter(func(ctx context.Context) error {
		if checks.Add(1) == 1 {
			close(entered)
			<-ctx.Done()
			return ctx.Err()
		}
		return nil
	}, time.Second, time.Millisecond)
	leaderCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	leader := make(chan error, 1)
	follower := make(chan error, 1)
	go func() { leader <- wait(leaderCtx) }()
	<-entered
	go func() { follower <- wait(context.Background()) }()
	cancel()
	require.ErrorIs(t, handoffWaitResult(t, leader), context.Canceled)
	require.NoError(t, handoffWaitResult(t, follower))
	require.EqualValues(t, 2, checks.Load())
}

func TestScheduleWriterWaiterFollowerCancellationDoesNotCancelLeader(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var checks atomic.Int32
	wait := newScheduleWriterHandoffWaiter(func(ctx context.Context) error {
		checks.Add(1)
		close(entered)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, time.Second, time.Millisecond)
	leader := make(chan error, 1)
	go func() { leader <- wait(context.Background()) }()
	<-entered
	followerCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, wait(followerCtx), context.DeadlineExceeded)
	require.EqualValues(t, 1, checks.Load())
	close(release)
	require.NoError(t, handoffWaitResult(t, leader))
}

func TestScheduleWriterWaiterQueueTimeCountsTowardBudget(t *testing.T) {
	// A shorter-lived leader holds the semaphore. The follower's budget must
	// already be running when it becomes the next polling caller.
	const budget = 300 * time.Millisecond
	entered := make(chan struct{})
	remaining := make(chan time.Duration, 1)
	var checks atomic.Int32
	wait := newScheduleWriterHandoffWaiter(func(ctx context.Context) error {
		if checks.Add(1) == 1 {
			close(entered)
		} else {
			deadline, _ := ctx.Deadline()
			remaining <- time.Until(deadline)
		}
		<-ctx.Done()
		return ctx.Err()
	}, budget, time.Millisecond)
	leaderCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	leader := make(chan error, 1)
	go func() { leader <- wait(leaderCtx) }()
	<-entered
	// Cancel the leader while the synchronous follower is waiting on its gate.
	timer := time.AfterFunc(100*time.Millisecond, cancel)
	defer timer.Stop()
	require.ErrorIs(t, wait(context.Background()), context.DeadlineExceeded)
	require.ErrorIs(t, handoffWaitResult(t, leader), context.Canceled)
	select {
	case left := <-remaining:
		require.Less(t, left, budget-50*time.Millisecond)
	default:
		t.Fatal("follower never acquired the handoff gate")
	}
	require.EqualValues(t, 2, checks.Load())
}

func TestScheduleWriterWaiterRetriesTransientErrors(t *testing.T) {
	for _, initial := range []error{
		errors.New("old writer remains"),
		apierrors.NewNotFound(schema.GroupResource{Resource: "deployments"}, "ml-pipeline"),
	} {
		t.Run(initial.Error(), func(t *testing.T) {
			var checks int
			wait := newScheduleWriterHandoffWaiter(func(context.Context) error {
				checks++
				if checks == 1 {
					return initial
				}
				return nil
			}, time.Second, time.Millisecond)
			require.NoError(t, wait(context.Background()))
			require.Equal(t, 2, checks)
		})
	}
}

func TestScheduleWriterWaiterRejectsConfigurationErrors(t *testing.T) {
	for _, failure := range []error{
		apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "writer", errors.New("missing RBAC")),
		apierrors.NewUnauthorized("missing credentials"),
		apierrors.NewBadRequest("invalid writer identity"),
		apierrors.NewInvalid(schema.GroupKind{Kind: "Pod"}, "writer", field.ErrorList{field.Invalid(field.NewPath("identity"), "", "required")}),
	} {
		t.Run(failure.Error(), func(t *testing.T) {
			var checks int
			wait := newScheduleWriterHandoffWaiter(func(context.Context) error { checks++; return failure }, time.Second, time.Millisecond)
			require.ErrorIs(t, wait(context.Background()), failure)
			require.Equal(t, 1, checks)
		})
	}
}

func TestScheduleWriterWaiterBoundsPersistentFailure(t *testing.T) {
	var checks int
	wait := newScheduleWriterHandoffWaiter(func(context.Context) error {
		checks++
		return errors.New("old writer remains")
	}, 40*time.Millisecond, time.Millisecond)
	require.ErrorIs(t, wait(context.Background()), context.DeadlineExceeded)
	require.Greater(t, checks, 1)
}

func TestScheduleWriterWaiterDoesNotReturnSuccessAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wait := newScheduleWriterHandoffWaiter(func(context.Context) error {
		cancel()
		return nil
	}, time.Second, time.Millisecond)
	require.ErrorIs(t, wait(ctx), context.Canceled)
	// A completed check can still be retained for independent live callers.
	require.NoError(t, wait(context.Background()))
}
