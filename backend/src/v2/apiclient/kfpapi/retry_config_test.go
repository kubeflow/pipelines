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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRetryPolicyFromEnvDefaults(t *testing.T) {
	for _, name := range []string{
		rpcAttemptTimeoutEnv,
		rpcTotalTimeoutEnv,
		rpcMaxAttemptsEnv,
		rpcInitialBackoffEnv,
		rpcMaxBackoffEnv,
	} {
		t.Setenv(name, "")
	}

	got, err := RetryPolicyFromEnv()
	require.NoError(t, err)
	require.Equal(t, defaultRetryPolicy, got)
}

func TestRetryPolicyFromEnvOverrides(t *testing.T) {
	t.Setenv(rpcAttemptTimeoutEnv, "2s")
	t.Setenv(rpcTotalTimeoutEnv, "8s")
	t.Setenv(rpcMaxAttemptsEnv, "4")
	t.Setenv(rpcInitialBackoffEnv, "50ms")
	t.Setenv(rpcMaxBackoffEnv, "500ms")

	got, err := RetryPolicyFromEnv()
	require.NoError(t, err)
	require.Equal(t, 2*time.Second, got.AttemptTimeout)
	require.Equal(t, 8*time.Second, got.TotalTimeout)
	require.Equal(t, 4, got.MaxAttempts)
	require.Equal(t, 50*time.Millisecond, got.InitialBackoff)
	require.Equal(t, 500*time.Millisecond, got.MaxBackoff)
}

func TestRetryPolicyFromEnvRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{rpcAttemptTimeoutEnv, "invalid"},
		{rpcTotalTimeoutEnv, "invalid"},
		{rpcMaxAttemptsEnv, "invalid"},
		{rpcInitialBackoffEnv, "invalid"},
		{rpcMaxBackoffEnv, "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(rpcAttemptTimeoutEnv, "")
			t.Setenv(rpcTotalTimeoutEnv, "")
			t.Setenv(rpcMaxAttemptsEnv, "")
			t.Setenv(rpcInitialBackoffEnv, "")
			t.Setenv(rpcMaxBackoffEnv, "")
			t.Setenv(tt.name, tt.value)

			_, err := RetryPolicyFromEnv()
			require.Error(t, err)
		})
	}
}

func TestRetryPolicyFromEnvRejectsInconsistentValues(t *testing.T) {
	t.Setenv(rpcAttemptTimeoutEnv, "10s")
	t.Setenv(rpcTotalTimeoutEnv, "1s")
	t.Setenv(rpcMaxAttemptsEnv, "0")
	t.Setenv(rpcInitialBackoffEnv, "2s")
	t.Setenv(rpcMaxBackoffEnv, "1s")

	_, err := RetryPolicyFromEnv()
	require.Error(t, err)
}
