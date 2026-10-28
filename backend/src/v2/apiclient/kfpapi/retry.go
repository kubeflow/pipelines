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
// writes get a single attempt bounded by TotalTimeout and are never retried.
type RetryPolicy struct {
	// AttemptTimeout is the deadline of one read attempt.
	AttemptTimeout time.Duration
	// TotalTimeout is the deadline of a whole read, including retries and
	// backoff, and of a single write attempt.
	TotalTimeout time.Duration
	// MaxAttempts is the maximum number of read attempts, including the first.
	MaxAttempts int
	// InitialBackoff is the first upper bound of the randomized retry delay.
	InitialBackoff time.Duration
	// MaxBackoff caps the upper bound of the randomized retry delay.
	MaxBackoff time.Duration
}

var defaultRetryPolicy = RetryPolicy{
	AttemptTimeout: 5 * time.Second,
	TotalTimeout:   15 * time.Second,
	MaxAttempts:    3,
	InitialBackoff: 100 * time.Millisecond,
	MaxBackoff:     time.Second,
}

func (p RetryPolicy) validate() error {
	if p.AttemptTimeout <= 0 ||
		p.TotalTimeout <= 0 ||
		p.MaxAttempts < 1 ||
		p.InitialBackoff < 0 ||
		p.MaxBackoff < p.InitialBackoff {
		return status.Error(codes.InvalidArgument, "invalid RPC retry policy")
	}
	return nil
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
		if err := overallCtx.Err(); err != nil {
			glog.Warningf(
				"Runtime RPC %s stopped before attempt %d: reason=%v elapsed=%s",
				operation, attempt, err, time.Since(started),
			)
			return zero, err
		}

		attemptCtx, attemptCancel := context.WithTimeout(overallCtx, policy.AttemptTimeout)
		result, err := call(attemptCtx)
		attemptTimedOut := errors.Is(attemptCtx.Err(), context.DeadlineExceeded) &&
			ctx.Err() == nil && overallCtx.Err() == nil
		attemptCancel()

		// The caller's cancellation or deadline takes precedence.
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}

		// Do not accept a result after the overall budget has expired.
		if overallCtx.Err() != nil {
			glog.Warningf(
				"Runtime RPC %s exhausted its time budget: attempts=%d reason=%v elapsed=%s",
				operation, attempt, overallCtx.Err(), time.Since(started),
			)
			return zero, overallCtx.Err()
		}

		if err == nil {
			if attemptTimedOut {
				return zero, context.DeadlineExceeded
			}
			if attempt > 1 {
				glog.V(2).Infof(
					"Runtime RPC %s succeeded after retries: attempts=%d elapsed=%s",
					operation, attempt, time.Since(started),
				)
			}
			return result, nil
		}

		// Retry only explicitly permitted transient failures. A remote
		// DeadlineExceeded status alone is not sufficient to trigger a retry.
		retryable := status.Code(err) == codes.Unavailable || attemptTimedOut

		if !retryable || attempt == policy.MaxAttempts {
			glog.Warningf(
				"Runtime RPC %s stopped: attempts=%d retryable=%t code=%s elapsed=%s",
				operation, attempt, retryable, status.Code(err), time.Since(started),
			)
			return zero, err
		}

		glog.V(2).Infof(
			"Retrying runtime RPC %s: attempt=%d code=%s elapsed=%s",
			operation, attempt, status.Code(err), time.Since(started),
		)

		// Apply bounded jitter while respecting the overall context.
		if sleepErr := env.sleep(overallCtx, env.jitter(backoff)); sleepErr != nil {
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

// boundedWrite calls a write RPC exactly once under policy.TotalTimeout. Writes
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
	callCtx, cancel := context.WithTimeout(ctx, policy.TotalTimeout)
	defer cancel()

	result, err := call(callCtx)
	if err == nil {
		return result, nil
	}

	// The caller's cancellation or deadline takes precedence.
	if ctx.Err() != nil {
		return zero, err
	}

	// Only our own budget is rewritten, so unrelated errors keep their code.
	if errors.Is(callCtx.Err(), context.DeadlineExceeded) &&
		(errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded) {
		glog.Warningf(
			"Runtime RPC %s write exceeded its budget and was not retried: budget=%s elapsed=%s",
			operation, policy.TotalTimeout, time.Since(started),
		)
		return zero, status.Errorf(
			codes.DeadlineExceeded,
			"runtime RPC %s did not complete within %s; the write may or may not have been committed and was not retried",
			operation, policy.TotalTimeout,
		)
	}

	glog.Warningf(
		"Runtime RPC %s write failed and was not retried: code=%s elapsed=%s",
		operation, status.Code(err), time.Since(started),
	)
	return zero, err
}
