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
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
)

const (
	rpcAttemptTimeoutEnv = apiclient.KFPRuntimeRPCAttemptTimeoutEnvVar
	rpcTotalTimeoutEnv   = apiclient.KFPRuntimeRPCTotalTimeoutEnvVar
	rpcMaxAttemptsEnv    = apiclient.KFPRuntimeRPCMaxAttemptsEnvVar
	rpcInitialBackoffEnv = apiclient.KFPRuntimeRPCInitialBackoffEnvVar
	rpcMaxBackoffEnv     = apiclient.KFPRuntimeRPCMaxBackoffEnvVar
)

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

	if policy.validate() != nil {
		return RetryPolicy{}, fmt.Errorf("invalid runtime RPC retry configuration")
	}

	return policy, nil
}
