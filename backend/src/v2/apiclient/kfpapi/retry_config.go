package kfpapi

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
)

const (
	rpcAttemptTimeoutEnv = "KFP_RUNTIME_RPC_ATTEMPT_TIMEOUT"
	rpcTotalTimeoutEnv   = "KFP_RUNTIME_RPC_TOTAL_TIMEOUT"
	rpcMaxAttemptsEnv    = "KFP_RUNTIME_RPC_MAX_ATTEMPTS"
	rpcInitialBackoffEnv = "KFP_RUNTIME_RPC_INITIAL_BACKOFF"
	rpcMaxBackoffEnv     = "KFP_RUNTIME_RPC_MAX_BACKOFF"
)

// NewFromEnv creates an API adapter using the retry policy configured in the environment.
func NewFromEnv(c *apiclient.Client) (API, error) {
	policy, err := RetryPolicyFromEnv()
	if err != nil {
		return nil, err
	}
	return NewWithRetryPolicy(c, policy), nil
}

// RetryPolicyFromEnv returns the default retry policy with environment overrides.
func RetryPolicyFromEnv() (RetryPolicy, error) {
	policy := defaultRetryPolicy

	parseDuration := func(name string, target *time.Duration) error {
		value := os.Getenv(name)
		if value == "" {
			return nil
		}
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("%s must be a duration: %w", name, err)
		}
		*target = parsed
		return nil
	}

	for _, item := range []struct {
		name   string
		target *time.Duration
	}{
		{rpcAttemptTimeoutEnv, &policy.AttemptTimeout},
		{rpcTotalTimeoutEnv, &policy.TotalTimeout},
		{rpcInitialBackoffEnv, &policy.InitialBackoff},
		{rpcMaxBackoffEnv, &policy.MaxBackoff},
	} {
		if err := parseDuration(item.name, item.target); err != nil {
			return RetryPolicy{}, err
		}
	}

	if value := os.Getenv(rpcMaxAttemptsEnv); value != "" {
		attempts, err := strconv.Atoi(value)
		if err != nil {
			return RetryPolicy{}, fmt.Errorf("%s must be an integer: %w", rpcMaxAttemptsEnv, err)
		}
		policy.MaxAttempts = attempts
	}

	if policy.AttemptTimeout <= 0 ||
		policy.TotalTimeout <= 0 ||
		policy.MaxAttempts < 1 ||
		policy.InitialBackoff < 0 ||
		policy.MaxBackoff < policy.InitialBackoff {
		return RetryPolicy{}, fmt.Errorf("invalid runtime RPC retry configuration")
	}

	return policy, nil
}
