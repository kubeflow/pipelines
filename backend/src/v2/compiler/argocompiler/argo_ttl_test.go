// Copyright 2025 The Kubeflow Authors
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

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTTLStrategy(t *testing.T) {
	tests := []struct {
		name           string
		pipelineConfig *pipelinespec.PipelineConfig
		// nil means the whole strategy should be nil
		expectedNil         bool
		wantAfterCompletion *int32
		wantAfterSuccess    *int32
		wantAfterFailure    *int32
	}{
		{
			name:        "nil config returns nil",
			expectedNil: true,
		},
		{
			name: "all-zero config returns nil",
			pipelineConfig: &pipelinespec.PipelineConfig{
				ResourceTtlOnCompletion: 0,
				ResourceTtlOnSuccess:    0,
				ResourceTtlOnFailure:    0,
			},
			expectedNil: true,
		},
		{
			name:                "resource_ttl_on_completion only → SecondsAfterCompletion",
			pipelineConfig:      &pipelinespec.PipelineConfig{ResourceTtlOnCompletion: 300},
			wantAfterCompletion: int32Ptr(300),
		},
		{
			name:             "resource_ttl_on_success only → SecondsAfterSuccess",
			pipelineConfig:   &pipelinespec.PipelineConfig{ResourceTtlOnSuccess: 600},
			wantAfterSuccess: int32Ptr(600),
		},
		{
			name:             "resource_ttl_on_failure only → SecondsAfterFailure",
			pipelineConfig:   &pipelinespec.PipelineConfig{ResourceTtlOnFailure: 120},
			wantAfterFailure: int32Ptr(120),
		},
		{
			name: "all three fields set independently",
			pipelineConfig: &pipelinespec.PipelineConfig{
				ResourceTtlOnCompletion: 300,
				ResourceTtlOnSuccess:    3600,
				ResourceTtlOnFailure:    60,
			},
			wantAfterCompletion: int32Ptr(300),
			wantAfterSuccess:    int32Ptr(3600),
			wantAfterFailure:    int32Ptr(60),
		},
		{
			name: "success and failure without completion",
			pipelineConfig: &pipelinespec.PipelineConfig{
				ResourceTtlOnSuccess: 7200,
				ResourceTtlOnFailure: 1800,
			},
			wantAfterSuccess: int32Ptr(7200),
			wantAfterFailure: int32Ptr(1800),
		},
		{
			name:           "negative resource_ttl_on_completion is treated as zero (not set)",
			pipelineConfig: &pipelinespec.PipelineConfig{ResourceTtlOnCompletion: -1},
			expectedNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := buildTTLStrategy(tt.pipelineConfig)

			if tt.expectedNil {
				assert.Nil(t, strategy)
				return
			}

			require.NotNil(t, strategy)

			if tt.wantAfterCompletion != nil {
				require.NotNil(t, strategy.SecondsAfterCompletion)
				assert.Equal(t, *tt.wantAfterCompletion, *strategy.SecondsAfterCompletion)
			} else {
				assert.Nil(t, strategy.SecondsAfterCompletion)
			}

			if tt.wantAfterSuccess != nil {
				require.NotNil(t, strategy.SecondsAfterSuccess)
				assert.Equal(t, *tt.wantAfterSuccess, *strategy.SecondsAfterSuccess)
			} else {
				assert.Nil(t, strategy.SecondsAfterSuccess)
			}

			if tt.wantAfterFailure != nil {
				require.NotNil(t, strategy.SecondsAfterFailure)
				assert.Equal(t, *tt.wantAfterFailure, *strategy.SecondsAfterFailure)
			} else {
				assert.Nil(t, strategy.SecondsAfterFailure)
			}
		})
	}
}

func TestBuildActiveDeadlineSeconds(t *testing.T) {
	tests := []struct {
		name           string
		pipelineConfig *pipelinespec.PipelineConfig
		expected       *int64
	}{
		{
			name:     "nil config returns 14-day default",
			expected: int64Ptr(defaultActiveDeadlineSeconds),
		},
		{
			name:           "zero value returns 14-day default",
			pipelineConfig: &pipelinespec.PipelineConfig{ActiveDeadlineSeconds: 0},
			expected:       int64Ptr(defaultActiveDeadlineSeconds),
		},
		{
			name:           "positive value overrides default",
			pipelineConfig: &pipelinespec.PipelineConfig{ActiveDeadlineSeconds: 3600},
			expected:       int64Ptr(3600),
		},
		{
			name:           "negative value disables deadline",
			pipelineConfig: &pipelinespec.PipelineConfig{ActiveDeadlineSeconds: -1},
			expected:       nil,
		},
		{
			name:           "large positive value is honored",
			pipelineConfig: &pipelinespec.PipelineConfig{ActiveDeadlineSeconds: 2147483647},
			expected:       int64Ptr(2147483647),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildActiveDeadlineSeconds(tt.pipelineConfig)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func int32Ptr(v int32) *int32 { return &v }
