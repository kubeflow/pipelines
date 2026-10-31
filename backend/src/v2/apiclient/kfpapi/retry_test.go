// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kfpapi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func testRetryPolicy() RetryPolicy {
	return RetryPolicy{
		AttemptTimeout: 100 * time.Millisecond,
		TotalTimeout:   300 * time.Millisecond,
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     2 * time.Millisecond,
	}
}

func TestRetryReadSucceedsImmediately(t *testing.T) {
	calls := 0
	got, err := retryRead(context.Background(), "TestRetryRead", testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			return "ok", nil
		})

	require.NoError(t, err)
	require.Equal(t, "ok", got)
	require.Equal(t, 1, calls)
}

func TestRetryReadRetriesUnavailable(t *testing.T) {
	calls := 0
	got, err := retryRead(context.Background(), "TestRetryRead", testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			if calls < 3 {
				return "", status.Error(codes.Unavailable, "temporary")
			}
			return "recovered", nil
		})

	require.NoError(t, err)
	require.Equal(t, "recovered", got)
	require.Equal(t, 3, calls)
}

func TestRetryReadDoesNotRetryPermanentError(t *testing.T) {
	calls := 0
	_, err := retryRead(context.Background(), "TestRetryRead", testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			return "", status.Error(codes.PermissionDenied, "denied")
		})

	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.Equal(t, 1, calls)
}

func TestRetryReadRespectsCallerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	_, err := retryRead(ctx, "TestRetryRead", testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			return "", errors.New("must not run")
		})

	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, calls)
}

func TestRetryReadBoundsHangingAttempt(t *testing.T) {
	policy := testRetryPolicy()
	policy.AttemptTimeout = 30 * time.Millisecond
	policy.TotalTimeout = 80 * time.Millisecond
	policy.InitialBackoff = 10 * time.Millisecond
	policy.MaxBackoff = 10 * time.Millisecond

	start := time.Now()
	calls := 0
	_, err := retryRead(context.Background(), "TestRetryRead", policy,
		func(ctx context.Context) (string, error) {
			calls++
			<-ctx.Done()
			return "", ctx.Err()
		})

	require.Error(t, err)
	require.Less(t, time.Since(start), time.Second)
	require.GreaterOrEqual(t, calls, 1)
	require.LessOrEqual(t, calls, policy.MaxAttempts)
}

func TestRetryReadRetriesAttemptTimeout(t *testing.T) {
	policy := testRetryPolicy()
	policy.AttemptTimeout = 10 * time.Millisecond
	policy.TotalTimeout = 200 * time.Millisecond

	calls := 0
	got, err := retryRead(context.Background(), "TestRetryRead", policy,
		func(ctx context.Context) (string, error) {
			calls++
			if calls == 1 {
				<-ctx.Done()
				return "", ctx.Err()
			}
			return "recovered", nil
		})

	require.NoError(t, err)
	require.Equal(t, "recovered", got)
	require.Equal(t, 2, calls)
}

func TestRetryReadRespectsCancellationDuringBackoff(t *testing.T) {
	policy := testRetryPolicy()
	policy.InitialBackoff = 200 * time.Millisecond
	policy.MaxBackoff = 200 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	attemptStarted := make(chan struct{}, 1)
	start := time.Now()

	done := make(chan error, 1)
	go func() {
		_, err := retryRead(ctx, "TestRetryRead", policy, func(context.Context) (string, error) {
			attemptStarted <- struct{}{}
			return "", status.Error(codes.Unavailable, "temporary")
		})
		done <- err
	}()

	select {
	case <-attemptStarted:
	case <-time.After(time.Second):
		t.Fatal("retry attempt did not start")
	}

	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
		require.Less(t, time.Since(start), 150*time.Millisecond)
	case <-time.After(time.Second):
		t.Fatal("retry did not stop promptly after cancellation")
	}
}

func TestRetryReadDoesNotRetryRemoteDeadlineExceeded(t *testing.T) {
	calls := 0
	_, err := retryRead(
		context.Background(),
		"TestRetryRead",
		testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			return "", status.Error(codes.DeadlineExceeded, "remote deadline")
		},
	)

	require.Error(t, err)
	require.Equal(t, codes.DeadlineExceeded, status.Code(err))
	require.Equal(t, 1, calls)
}

func TestRetryReadEnforcesOverallTimeoutAcrossAttempts(t *testing.T) {
	policy := testRetryPolicy()
	policy.AttemptTimeout = 200 * time.Millisecond
	policy.TotalTimeout = 70 * time.Millisecond
	policy.MaxAttempts = 5
	policy.InitialBackoff = 5 * time.Millisecond
	policy.MaxBackoff = 5 * time.Millisecond

	calls := 0
	start := time.Now()
	_, err := retryRead(
		context.Background(),
		"TestRetryRead",
		policy,
		func(ctx context.Context) (string, error) {
			calls++
			<-ctx.Done()
			return "", ctx.Err()
		},
	)
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Less(t, elapsed, time.Second)
	require.Equal(t, 1, calls)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRetryReadRejectsSuccessAfterAttemptTimeout(t *testing.T) {
	policy := testRetryPolicy()
	policy.AttemptTimeout = 10 * time.Millisecond
	policy.TotalTimeout = time.Second
	policy.MaxAttempts = 1

	result, err := retryRead(
		context.Background(),
		"TestLateSuccess",
		policy,
		func(ctx context.Context) (string, error) {
			<-ctx.Done()
			return "late success", nil
		},
	)

	require.Empty(t, result)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// fakeRetryEnv records the delays the retry loop asks for without sleeping.
type fakeRetryEnv struct {
	upperBounds []time.Duration
	slept       []time.Duration
	onSleep     func(ctx context.Context) error
}

func (f *fakeRetryEnv) env() retryEnv {
	return retryEnv{
		jitter: func(upperBound time.Duration) time.Duration {
			f.upperBounds = append(f.upperBounds, upperBound)
			return upperBound / 2
		},
		sleep: func(ctx context.Context, d time.Duration) error {
			f.slept = append(f.slept, d)
			if f.onSleep != nil {
				return f.onSleep(ctx)
			}
			return ctx.Err()
		},
	}
}

func TestRetryReadBackoffScheduleIsCappedAndJittered(t *testing.T) {
	policy := RetryPolicy{
		AttemptTimeout: time.Minute,
		TotalTimeout:   time.Hour,
		MaxAttempts:    6,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     time.Second,
	}
	fake := &fakeRetryEnv{}

	calls := 0
	_, err := retryReadWithEnv(context.Background(), fake.env(), "TestRetryRead", policy,
		func(context.Context) (string, error) {
			calls++
			return "", status.Error(codes.Unavailable, "temporary")
		})

	require.Equal(t, codes.Unavailable, status.Code(err))
	require.Equal(t, 6, calls)
	require.Equal(t, []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		time.Second,
	}, fake.upperBounds)
	for i, upperBound := range fake.upperBounds {
		require.Equal(t, upperBound/2, fake.slept[i])
	}
}

func TestRetryReadStopsWhenCallerCancelsDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fake := &fakeRetryEnv{onSleep: func(sleepCtx context.Context) error {
		cancel()
		return sleepCtx.Err()
	}}

	calls := 0
	_, err := retryReadWithEnv(ctx, fake.env(), "TestRetryRead", testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			return "", status.Error(codes.Unavailable, "temporary")
		})

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, calls)
	require.Len(t, fake.slept, 1)
}

func TestRetryReadDoesNotSleepAfterFinalAttempt(t *testing.T) {
	policy := testRetryPolicy()
	policy.MaxAttempts = 1
	fake := &fakeRetryEnv{}

	_, err := retryReadWithEnv(context.Background(), fake.env(), "TestRetryRead", policy,
		func(context.Context) (string, error) {
			return "", status.Error(codes.Unavailable, "temporary")
		})

	require.Equal(t, codes.Unavailable, status.Code(err))
	require.Empty(t, fake.slept)
}

func TestRetryPolicyValidate(t *testing.T) {
	require.NoError(t, defaultRetryPolicy.validate())
	require.NoError(t, testRetryPolicy().validate())

	for name, mutate := range map[string]func(*RetryPolicy){
		"zero attempt timeout": func(p *RetryPolicy) { p.AttemptTimeout = 0 },
		"zero total timeout":   func(p *RetryPolicy) { p.TotalTimeout = 0 },
		"zero attempts":        func(p *RetryPolicy) { p.MaxAttempts = 0 },
		"negative backoff":     func(p *RetryPolicy) { p.InitialBackoff = -time.Millisecond },
		"max below initial":    func(p *RetryPolicy) { p.MaxBackoff = p.InitialBackoff - time.Nanosecond },
	} {
		policy := testRetryPolicy()
		mutate(&policy)
		require.Equal(t, codes.InvalidArgument, status.Code(policy.validate()), name)
	}
}

func TestRetryReadAndBoundedWriteRejectInvalidPolicy(t *testing.T) {
	readCalls, writeCalls := 0, 0

	_, err := retryRead(context.Background(), "TestRetryRead", RetryPolicy{},
		func(context.Context) (string, error) {
			readCalls++
			return "", nil
		})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = boundedWrite(context.Background(), "TestWrite", RetryPolicy{},
		func(context.Context) (string, error) {
			writeCalls++
			return "", nil
		})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	require.Zero(t, readCalls)
	require.Zero(t, writeCalls)
}

func TestBoundedWriteSucceedsOnce(t *testing.T) {
	calls := 0
	got, err := boundedWrite(context.Background(), "TestWrite", testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			return "created", nil
		})

	require.NoError(t, err)
	require.Equal(t, "created", got)
	require.Equal(t, 1, calls)
}

func TestBoundedWriteNeverRetriesUnavailable(t *testing.T) {
	calls := 0
	_, err := boundedWrite(context.Background(), "TestWrite", testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			return "", status.Error(codes.Unavailable, "temporary")
		})

	require.Equal(t, codes.Unavailable, status.Code(err))
	require.Equal(t, 1, calls)
}

func TestBoundedWriteKeepsUnrelatedErrorCode(t *testing.T) {
	_, err := boundedWrite(context.Background(), "TestWrite", testRetryPolicy(),
		func(context.Context) (string, error) {
			return "", status.Error(codes.PermissionDenied, "denied")
		})

	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestBoundedWriteBoundsHangingCall(t *testing.T) {
	policy := testRetryPolicy()
	policy.TotalTimeout = 30 * time.Millisecond

	start := time.Now()
	calls := 0
	_, err := boundedWrite(context.Background(), "TestWrite", policy,
		func(ctx context.Context) (string, error) {
			calls++
			<-ctx.Done()
			return "", ctx.Err()
		})

	require.Equal(t, codes.DeadlineExceeded, status.Code(err))
	require.Contains(t, err.Error(), "may or may not have been committed")
	require.Equal(t, 1, calls)
	require.Less(t, time.Since(start), time.Second)
}

func TestBoundedWriteKeepsSuccessThatRacesTheDeadline(t *testing.T) {
	policy := testRetryPolicy()
	policy.TotalTimeout = 20 * time.Millisecond

	got, err := boundedWrite(context.Background(), "TestWrite", policy,
		func(ctx context.Context) (string, error) {
			<-ctx.Done()
			// The server committed the write even though the deadline passed.
			return "committed", nil
		})

	require.NoError(t, err)
	require.Equal(t, "committed", got)
}

func TestBoundedWriteRespectsCallerCancellation(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	_, err := boundedWrite(canceled, "TestWrite", testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			return "", errors.New("must not run")
		})
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, calls)

	ctx, cancelDuringCall := context.WithCancel(context.Background())
	defer cancelDuringCall()
	_, err = boundedWrite(ctx, "TestWrite", testRetryPolicy(),
		func(callCtx context.Context) (string, error) {
			cancelDuringCall()
			<-callCtx.Done()
			return "", callCtx.Err()
		})
	require.ErrorIs(t, err, context.Canceled)
	require.NotContains(t, err.Error(), "may or may not have been committed")
}
