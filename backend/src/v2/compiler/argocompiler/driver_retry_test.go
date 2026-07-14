// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argocompiler

import (
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDriverRetryStrategy(t *testing.T) {
	tests := []struct {
		name string
		// nil means the environment variable is unset
		envValue  *string
		wantNil   bool
		wantLimit int32
	}{
		{
			name:      "default when unset",
			wantLimit: DefaultDriverRetryLimit,
		},
		{
			name:      "explicit limit",
			envValue:  stringPtr("7"),
			wantLimit: 7,
		},
		{
			name:     "zero disables retries",
			envValue: stringPtr("0"),
			wantNil:  true,
		},
		{
			name:      "invalid value falls back to default",
			envValue:  stringPtr("not-a-number"),
			wantLimit: DefaultDriverRetryLimit,
		},
		{
			name:      "negative value falls back to default",
			envValue:  stringPtr("-2"),
			wantLimit: DefaultDriverRetryLimit,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.envValue != nil {
				t.Setenv(DriverRetryLimitEnvVar, *test.envValue)
			}
			strategy := getDriverRetryStrategy()
			if test.wantNil {
				assert.Nil(t, strategy)
				return
			}
			require.NotNil(t, strategy)
			require.NotNil(t, strategy.Limit)
			assert.Equal(t, test.wantLimit, strategy.Limit.IntVal)
			assert.Equal(t, wfapi.RetryPolicyAlways, strategy.RetryPolicy)
			require.NotNil(t, strategy.Backoff)
			assert.Equal(t, "5s", strategy.Backoff.Duration)
			assert.Equal(t, "1m", strategy.Backoff.MaxDuration)
		})
	}
}

func stringPtr(value string) *string {
	return &value
}
