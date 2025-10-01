// Copyright 2025 The Kubeflow Authors
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

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDriverPodConfig_Empty(t *testing.T) {
	// Clear environment variables
	os.Unsetenv(EnvDriverPodLabels)
	os.Unsetenv(EnvDriverPodAnnotations)

	config, err := GetDriverPodConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Empty(t, config.Labels)
	assert.Empty(t, config.Annotations)
}

func TestGetDriverPodConfig_JSONFormat(t *testing.T) {
	testCases := []struct {
		name                string
		labelsJSON          string
		annotationsJSON     string
		expectedLabels      map[string]string
		expectedAnnotations map[string]string
	}{
		{
			name:            "Valid JSON labels and annotations",
			labelsJSON:      `{"env":"prod","team":"data"}`,
			annotationsJSON: `{"description":"ML pipeline","owner":"team-a"}`,
			expectedLabels: map[string]string{
				"env":  "prod",
				"team": "data",
			},
			expectedAnnotations: map[string]string{
				"description": "ML pipeline",
				"owner":       "team-a",
			},
		},
		{
			name:                "Empty JSON objects",
			labelsJSON:          `{}`,
			annotationsJSON:     `{}`,
			expectedLabels:      map[string]string{},
			expectedAnnotations: map[string]string{},
		},
		{
			name:            "Labels only",
			labelsJSON:      `{"app":"ml-pipeline","version":"v1"}`,
			annotationsJSON: "",
			expectedLabels: map[string]string{
				"app":     "ml-pipeline",
				"version": "v1",
			},
			expectedAnnotations: map[string]string{},
		},
		{
			name:            "Annotations only",
			labelsJSON:      "",
			annotationsJSON: `{"note":"test-annotation"}`,
			expectedLabels:  map[string]string{},
			expectedAnnotations: map[string]string{
				"note": "test-annotation",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv(EnvDriverPodLabels, tc.labelsJSON)
			os.Setenv(EnvDriverPodAnnotations, tc.annotationsJSON)
			defer func() {
				os.Unsetenv(EnvDriverPodLabels)
				os.Unsetenv(EnvDriverPodAnnotations)
			}()

			config, err := GetDriverPodConfig()
			require.NoError(t, err)
			assert.Equal(t, tc.expectedLabels, config.Labels)
			assert.Equal(t, tc.expectedAnnotations, config.Annotations)
		})
	}
}

func TestGetDriverPodConfig_KeyValueFormat(t *testing.T) {
	testCases := []struct {
		name                string
		labelsKV            string
		annotationsKV       string
		expectedLabels      map[string]string
		expectedAnnotations map[string]string
	}{
		{
			name:          "Single key=value pair",
			labelsKV:      "env=prod",
			annotationsKV: "description=test",
			expectedLabels: map[string]string{
				"env": "prod",
			},
			expectedAnnotations: map[string]string{
				"description": "test",
			},
		},
		{
			name:          "Multiple key=value pairs",
			labelsKV:      "env=prod,team=data,app=ml-pipeline",
			annotationsKV: "owner=team-a,version=1.0.0",
			expectedLabels: map[string]string{
				"env":  "prod",
				"team": "data",
				"app":  "ml-pipeline",
			},
			expectedAnnotations: map[string]string{
				"owner":   "team-a",
				"version": "1.0.0",
			},
		},
		{
			name:          "Key=value with spaces",
			labelsKV:      " env = prod , team = data ",
			annotationsKV: " description = ML Pipeline ",
			expectedLabels: map[string]string{
				"env":  "prod",
				"team": "data",
			},
			expectedAnnotations: map[string]string{
				"description": "ML Pipeline",
			},
		},
		{
			name:          "Values with special characters",
			labelsKV:      "app=ml-pipeline-v1.0",
			annotationsKV: "url=https://example.com/path",
			expectedLabels: map[string]string{
				"app": "ml-pipeline-v1.0",
			},
			expectedAnnotations: map[string]string{
				"url": "https://example.com/path",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv(EnvDriverPodLabels, tc.labelsKV)
			os.Setenv(EnvDriverPodAnnotations, tc.annotationsKV)
			defer func() {
				os.Unsetenv(EnvDriverPodLabels)
				os.Unsetenv(EnvDriverPodAnnotations)
			}()

			config, err := GetDriverPodConfig()
			require.NoError(t, err)
			assert.Equal(t, tc.expectedLabels, config.Labels)
			assert.Equal(t, tc.expectedAnnotations, config.Annotations)
		})
	}
}

func TestGetDriverPodConfig_ReservedLabelsFiltered(t *testing.T) {
	testCases := []struct {
		name           string
		labelsInput    string
		expectedLabels map[string]string
	}{
		{
			name:        "Filter reserved label - JSON format",
			labelsInput: `{"env":"prod","pipelines.kubeflow.org/reserved":"value","team":"data"}`,
			expectedLabels: map[string]string{
				"env":  "prod",
				"team": "data",
			},
		},
		{
			name:        "Filter reserved label - key=value format",
			labelsInput: "env=prod,pipelines.kubeflow.org/reserved=value,team=data",
			expectedLabels: map[string]string{
				"env":  "prod",
				"team": "data",
			},
		},
		{
			name:        "Filter multiple reserved labels",
			labelsInput: "env=prod,pipelines.kubeflow.org/run_id=123,pipelines.kubeflow.org/pipeline_id=456,team=data",
			expectedLabels: map[string]string{
				"env":  "prod",
				"team": "data",
			},
		},
		{
			name:           "All labels are reserved",
			labelsInput:    "pipelines.kubeflow.org/run_id=123,pipelines.kubeflow.org/pipeline_id=456",
			expectedLabels: map[string]string{},
		},
		{
			name:        "Similar but not reserved prefix",
			labelsInput: "env=prod,pipelines.example.org/custom=value",
			expectedLabels: map[string]string{
				"env":                          "prod",
				"pipelines.example.org/custom": "value",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv(EnvDriverPodLabels, tc.labelsInput)
			os.Unsetenv(EnvDriverPodAnnotations)
			defer os.Unsetenv(EnvDriverPodLabels)

			config, err := GetDriverPodConfig()
			require.NoError(t, err)
			assert.Equal(t, tc.expectedLabels, config.Labels)
		})
	}
}

func TestGetDriverPodConfig_InvalidInput(t *testing.T) {
	testCases := []struct {
		name                string
		labelsInput         string
		annotationsInput    string
		expectedLabels      map[string]string
		expectedAnnotations map[string]string
	}{
		{
			name:                "Invalid JSON - fallback to k=v parsing",
			labelsInput:         `{"invalid-json`,
			annotationsInput:    "",
			expectedLabels:      map[string]string{}, // Invalid k=v format, empty result
			expectedAnnotations: map[string]string{},
		},
		{
			name:                "Invalid k=v pairs - missing equals",
			labelsInput:         "invalidpair,env=prod",
			annotationsInput:    "",
			expectedLabels:      map[string]string{"env": "prod"}, // Valid pair accepted
			expectedAnnotations: map[string]string{},
		},
		{
			name:                "Empty key in k=v pair",
			labelsInput:         "=value,env=prod",
			annotationsInput:    "",
			expectedLabels:      map[string]string{"env": "prod"},
			expectedAnnotations: map[string]string{},
		},
		{
			name:                "Empty value in k=v pair",
			labelsInput:         "key=,env=prod",
			annotationsInput:    "",
			expectedLabels:      map[string]string{"env": "prod"},
			expectedAnnotations: map[string]string{},
		},
		{
			name:                "Only commas",
			labelsInput:         ",,,",
			annotationsInput:    "",
			expectedLabels:      map[string]string{},
			expectedAnnotations: map[string]string{},
		},
		{
			name:                "Whitespace only",
			labelsInput:         "   ",
			annotationsInput:    "   ",
			expectedLabels:      map[string]string{},
			expectedAnnotations: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv(EnvDriverPodLabels, tc.labelsInput)
			os.Setenv(EnvDriverPodAnnotations, tc.annotationsInput)
			defer func() {
				os.Unsetenv(EnvDriverPodLabels)
				os.Unsetenv(EnvDriverPodAnnotations)
			}()

			config, err := GetDriverPodConfig()
			require.NoError(t, err) // Should not error, graceful degradation
			assert.Equal(t, tc.expectedLabels, config.Labels)
			assert.Equal(t, tc.expectedAnnotations, config.Annotations)
		})
	}
}

func TestParseKVPairs(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:  "Single pair",
			input: "key=value",
			expected: map[string]string{
				"key": "value",
			},
		},
		{
			name:  "Multiple pairs",
			input: "key1=value1,key2=value2,key3=value3",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			name:  "Pairs with spaces",
			input: " key1 = value1 , key2 = value2 ",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:  "Value with equals sign",
			input: "key=value=with=equals",
			expected: map[string]string{
				"key": "value=with=equals",
			},
		},
		{
			name:  "Skip invalid pairs",
			input: "valid=value,invalid,another=valid",
			expected: map[string]string{
				"valid":   "value",
				"another": "valid",
			},
		},
		{
			name:     "Only invalid pairs",
			input:    "invalid1,invalid2",
			expected: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseKVPairs(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateSystemLabels(t *testing.T) {
	testCases := []struct {
		name     string
		input    *DriverPodConfig
		expected map[string]string
	}{
		{
			name:     "Nil config",
			input:    nil,
			expected: nil,
		},
		{
			name: "Nil labels",
			input: &DriverPodConfig{
				Labels: nil,
			},
			expected: nil,
		},
		{
			name: "No reserved labels",
			input: &DriverPodConfig{
				Labels: map[string]string{
					"env":  "prod",
					"team": "data",
				},
			},
			expected: map[string]string{
				"env":  "prod",
				"team": "data",
			},
		},
		{
			name: "Filter reserved labels",
			input: &DriverPodConfig{
				Labels: map[string]string{
					"env":                           "prod",
					"pipelines.kubeflow.org/run_id": "123",
					"team":                          "data",
				},
			},
			expected: map[string]string{
				"env":  "prod",
				"team": "data",
			},
		},
		{
			name: "Filter multiple reserved labels",
			input: &DriverPodConfig{
				Labels: map[string]string{
					"env":                                "prod",
					"pipelines.kubeflow.org/run_id":      "123",
					"pipelines.kubeflow.org/pipeline_id": "456",
					"team":                               "data",
				},
			},
			expected: map[string]string{
				"env":  "prod",
				"team": "data",
			},
		},
		{
			name: "All labels are reserved",
			input: &DriverPodConfig{
				Labels: map[string]string{
					"pipelines.kubeflow.org/run_id":      "123",
					"pipelines.kubeflow.org/pipeline_id": "456",
				},
			},
			expected: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validateSystemLabels(tc.input)
			if tc.input != nil {
				assert.Equal(t, tc.expected, tc.input.Labels)
			}
		})
	}
}

func TestParseConfigValue(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:  "Valid JSON",
			input: `{"key1":"value1","key2":"value2"}`,
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:  "Valid k=v format",
			input: "key1=value1,key2=value2",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:     "Invalid JSON falls back to k=v",
			input:    `{"invalid`,
			expected: map[string]string{}, // Invalid k=v format too
		},
		{
			name:  "JSON with whitespace",
			input: ` {"key":"value"} `,
			expected: map[string]string{
				"key": "value",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseConfigValue(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}
