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

type RetryPolicy struct {
	AttemptTimeout time.Duration
	TotalTimeout   time.Duration
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

var defaultRetryPolicy = RetryPolicy{
	AttemptTimeout: 5 * time.Second,
	TotalTimeout:   15 * time.Second,
	MaxAttempts:    3,
	InitialBackoff: 100 * time.Millisecond,
	MaxBackoff:     time.Second,
}

func retryRead[T any](
	ctx context.Context,
	operation string,
	policy RetryPolicy,
	call func(context.Context) (T, error),
) (T, error) {
	var zero T

	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if policy.AttemptTimeout <= 0 ||
		policy.TotalTimeout <= 0 ||
		policy.MaxAttempts < 1 ||
		policy.InitialBackoff < 0 ||
		policy.MaxBackoff < policy.InitialBackoff {
		return zero, status.Error(codes.InvalidArgument, "invalid RPC retry policy")
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

		// The caller's cancellation or deadline takes precedence.
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}

		// Stop if the overall retry budget has expired.
		if overallCtx.Err() != nil {
			glog.Warningf(
				"Runtime RPC %s exhausted its time budget: attempts=%d reason=%v elapsed=%s",
				operation, attempt, overallCtx.Err(), time.Since(started),
			)
			return zero, overallCtx.Err()
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
		delay := backoff
		if delay > 0 {
			delay = time.Duration(rand.Int63n(int64(delay) + 1))
		}

		timer := time.NewTimer(delay)
		select {
		case <-overallCtx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return zero, overallCtx.Err()
		case <-timer.C:
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
