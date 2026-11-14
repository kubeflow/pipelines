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
	"math/rand"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetryPolicy bounds runtime API RPCs. Reads are retried within these limits;
// writes get a single attempt bounded by WriteTimeout and are never retried.
type RetryPolicy struct {
	// AttemptTimeout is the deadline of one read attempt.
	AttemptTimeout time.Duration
	// TotalTimeout is the deadline of a whole read, including retries and backoff.
	TotalTimeout time.Duration
	// MaxAttempts is the maximum number of read attempts, including the first.
	MaxAttempts int
	// InitialBackoff is the first upper bound of the randomized retry delay.
	InitialBackoff time.Duration
	// MaxBackoff caps the upper bound of the randomized retry delay.
	MaxBackoff time.Duration
	// WriteTimeout is the deadline of the single attempt of a write. It is
	// longer than the read budget because a timed-out write is not retried and
	// leaves its outcome unknown.
	WriteTimeout time.Duration
}

var defaultRetryPolicy = RetryPolicy{
	AttemptTimeout: 5 * time.Second,
	TotalTimeout:   15 * time.Second,
	MaxAttempts:    3,
	InitialBackoff: 100 * time.Millisecond,
	MaxBackoff:     time.Second,
	WriteTimeout:   30 * time.Second,
}

func (p RetryPolicy) validate() error {
	var problem string
	switch {
	case p.AttemptTimeout <= 0:
		problem = "attempt timeout must be positive"
	case p.TotalTimeout <= 0:
		problem = "total timeout must be positive"
	case p.MaxAttempts < 1:
		problem = "max attempts must be at least 1"
	case p.InitialBackoff < 0:
		problem = "initial backoff must not be negative"
	case p.MaxBackoff < p.InitialBackoff:
		problem = "max backoff must not be smaller than initial backoff"
	case p.WriteTimeout <= 0:
		problem = "write timeout must be positive"
	default:
		return nil
	}
	return status.Errorf(codes.InvalidArgument, "invalid RPC retry policy: %s", problem)
}

// retryEnv holds the non-deterministic collaborators of the retry loop so that
// tests can replace them. Attempt deadlines still use real contexts.
type retryEnv struct {
	// sleep blocks for d, or returns ctx.Err() as soon as ctx is done.
	sleep func(ctx context.Context, d time.Duration) error
	// jitter returns a duration in [0, upperBound].
	jitter func(upperBound time.Duration) time.Duration
}

var defaultRetryEnv = retryEnv{sleep: sleepContext, jitter: fullJitter}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func fullJitter(upperBound time.Duration) time.Duration {
	if upperBound <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(upperBound) + 1))
}

// expiredErr returns ctx.Err(), or context.DeadlineExceeded once the deadline
// of ctx has passed even if its timer has not fired yet. gRPC can return
// DEADLINE_EXCEEDED before that timer fires when the server-side deadline,
// derived from the same deadline, expires first.
func expiredErr(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return context.DeadlineExceeded
	}
	return nil
}

// retryRead calls a read-only RPC under policy. Only codes.Unavailable and a
// locally enforced attempt timeout are retried; every other error is returned
// immediately. The caller's cancellation and deadline always take precedence.
func retryRead[T any](
	ctx context.Context,
	operation string,
	policy RetryPolicy,
	call func(context.Context) (T, error),
) (T, error) {
	return retryReadWithEnv(ctx, defaultRetryEnv, operation, policy, call)
}

func retryReadWithEnv[T any](
	ctx context.Context,
	env retryEnv,
	operation string,
	policy RetryPolicy,
	call func(context.Context) (T, error),
) (T, error) {
	var zero T

	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if err := policy.validate(); err != nil {
		return zero, err
	}

	started := time.Now()
	overallCtx, cancel := context.WithTimeout(ctx, policy.TotalTimeout)
	defer cancel()

	backoff := policy.InitialBackoff

	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if err := expiredErr(overallCtx); err != nil {
			glog.Warningf(
				"Runtime RPC %s stopped before attempt %d: reason=%v elapsed=%s",
				operation, attempt, err, time.Since(started),
			)
			return zero, err
		}

		attemptCtx, attemptCancel := context.WithTimeout(overallCtx, policy.AttemptTimeout)
		result, err := call(attemptCtx)
		callerErr := expiredErr(ctx)
		budgetErr := expiredErr(overallCtx)
		attemptTimedOut := callerErr == nil && budgetErr == nil &&
			errors.Is(expiredErr(attemptCtx), context.DeadlineExceeded)
		attemptCancel()

		// The caller's cancellation or deadline takes precedence.
		if callerErr != nil {
			glog.Warningf(
				"Runtime RPC %s stopped by its caller: attempts=%d reason=%v elapsed=%s",
				operation, attempt, callerErr, time.Since(started),
			)
			return zero, callerErr
		}

		// Do not accept a result after the overall budget has expired.
		if budgetErr != nil {
			glog.Warningf(
				"Runtime RPC %s exhausted its time budget: attempts=%d reason=%v elapsed=%s",
				operation, attempt, budgetErr, time.Since(started),
			)
			return zero, budgetErr
		}

		if err == nil {
			if attemptTimedOut {
				return zero, context.DeadlineExceeded
			}
			if attempt > 1 {
				glog.Infof(
					"Runtime RPC %s succeeded after retries: attempts=%d elapsed=%s",
					operation, attempt, time.Since(started),
				)
			}
			return result, nil
		}

		// Retry only explicitly permitted transient failures. A remote
		// DeadlineExceeded status alone is not sufficient to trigger a retry, and a
		// permanent status that arrives after the attempt deadline is not retried.
		deadlineErr := errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded
		retryable := status.Code(err) == codes.Unavailable || (attemptTimedOut && deadlineErr)

		if !retryable || attempt == policy.MaxAttempts {
			glog.Warningf(
				"Runtime RPC %s stopped: attempts=%d retryable=%t code=%s elapsed=%s",
				operation, attempt, retryable, status.Code(err), time.Since(started),
			)
			return zero, err
		}

		glog.Infof(
			"Retrying runtime RPC %s: attempt=%d code=%s elapsed=%s",
			operation, attempt, status.Code(err), time.Since(started),
		)

		// Apply bounded jitter while respecting the overall context.
		if sleepErr := env.sleep(overallCtx, env.jitter(backoff)); sleepErr != nil {
			stoppedBy := "its time budget"
			if expiredErr(ctx) != nil {
				stoppedBy = "its caller"
			}
			glog.Warningf(
				"Runtime RPC %s stopped by %s during backoff: attempts=%d reason=%v elapsed=%s",
				operation, stoppedBy, attempt, sleepErr, time.Since(started),
			)
			return zero, sleepErr
		}

		if backoff < policy.MaxBackoff {
			if backoff > policy.MaxBackoff/2 {
				backoff = policy.MaxBackoff
			} else {
				backoff *= 2
			}
		}
	}

	return zero, status.Error(codes.Unavailable, "RPC retry budget exhausted")
}

// boundedWrite calls a write RPC exactly once under policy.WriteTimeout. Writes
// are never replayed here: a timeout or Unavailable error does not prove that
// the write failed to commit, so retrying is only safe for writes whose replay
// contract has been established separately. A write that reports success is
// returned as such even if the deadline raced with the response.
func boundedWrite[T any](
	ctx context.Context,
	operation string,
	policy RetryPolicy,
	call func(context.Context) (T, error),
) (T, error) {
	var zero T

	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if err := policy.validate(); err != nil {
		return zero, err
	}

	started := time.Now()
	callCtx, cancel := context.WithTimeout(ctx, policy.WriteTimeout)
	defer cancel()

	result, err := call(callCtx)
	if err == nil {
		return result, nil
	}

	// The caller's cancellation or deadline takes precedence.
	if callerErr := expiredErr(ctx); callerErr != nil {
		glog.Warningf(
			"Runtime RPC %s write stopped by its caller and was not retried: reason=%v elapsed=%s",
			operation, callerErr, time.Since(started),
		)
		return zero, callerErr
	}

	// Only our own budget is rewritten, so unrelated errors keep their code.
	if errors.Is(expiredErr(callCtx), context.DeadlineExceeded) &&
		(errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded) {
		glog.Warningf(
			"Runtime RPC %s write exceeded its budget and was not retried: budget=%s elapsed=%s",
			operation, policy.WriteTimeout, time.Since(started),
		)
		return zero, status.Errorf(
			codes.DeadlineExceeded,
			"runtime RPC %s did not complete within %s; the write may or may not have been committed and was not retried",
			operation, policy.WriteTimeout,
		)
	}

	glog.Warningf(
		"Runtime RPC %s write failed and was not retried: code=%s elapsed=%s",
		operation, status.Code(err), time.Since(started),
	)
	return zero, err
}
