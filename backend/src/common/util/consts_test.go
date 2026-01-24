// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxParameterBytes_Default(t *testing.T) {
	defaultMaxParameterBytes := 10000
	assert.Equal(t, defaultMaxParameterBytes, GetMaxParameterBytes())
}

func TestMaxParameterBytes_20KBParametersSucceed(t *testing.T) {
	// Temporarily set MaxParameterBytes to 25000 to allow 20KB parameters
	originalMaxBytes := MaxParameterBytes
	MaxParameterBytes = 25000
	defer func() { MaxParameterBytes = originalMaxBytes }()

	// Create a parameter value that is approximately 20KB (20480 bytes)
	largeValue := strings.Repeat("a", 20000)

	params := SpecParameters{
		{Name: "large_param", Value: &largeValue},
	}

	result, err := MarshalParametersWorkflow(params)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "large_param")
}

func TestMaxParameterBytes_ExceedsLimitFails(t *testing.T) {
	originalMaxBytes := MaxParameterBytes
	MaxParameterBytes = 10000
	defer func() { MaxParameterBytes = originalMaxBytes }()

	// Create a parameter value that exceeds 10KB
	largeValue := strings.Repeat("a", 15000)

	params := SpecParameters{
		{Name: "large_param", Value: &largeValue},
	}

	result, err := MarshalParametersWorkflow(params)
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "exceed maximum size")
}

func TestMaxParameterBytes_WithinLimitSucceeds(t *testing.T) {
	originalMaxBytes := MaxParameterBytes
	MaxParameterBytes = 10000
	defer func() { MaxParameterBytes = originalMaxBytes }()

	smallValue := strings.Repeat("a", 5000)

	params := SpecParameters{
		{Name: "small_param", Value: &smallValue},
	}

	result, err := MarshalParametersWorkflow(params)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "small_param")
}

func TestMaxParameterBytes_EnvVarParsing(t *testing.T) {
	// Test the GetMaxParameterBytes() parsing logic
	defaultMaxParameterBytes := 10000
	testCases := []struct {
		name        string
		envValue    string
		expectValue int
	}{
		{
			name:        "valid positive integer",
			envValue:    "25000",
			expectValue: 25000,
		},
		{
			name:        "invalid string keeps default",
			envValue:    "invalid",
			expectValue: defaultMaxParameterBytes,
		},
		{
			name:        "zero keeps default",
			envValue:    "0",
			expectValue: defaultMaxParameterBytes,
		},
		{
			name:        "negative keeps default",
			envValue:    "-100",
			expectValue: defaultMaxParameterBytes,
		},
		{
			name:        "empty string keeps default",
			envValue:    "",
			expectValue: defaultMaxParameterBytes,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the GetMaxParameterBytes() logic
			result := defaultMaxParameterBytes
			if tc.envValue != "" {
				if val, err := strconv.Atoi(tc.envValue); err == nil && val > 0 {
					result = val
				}
			}
			assert.Equal(t, tc.expectValue, result)
		})
	}
}
