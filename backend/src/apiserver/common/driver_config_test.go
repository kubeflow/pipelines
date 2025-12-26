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

package common

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// resetDriverConfig resets the driver config state for testing
func resetDriverConfig() {
	driverConfigMutex.Lock()
	defer driverConfigMutex.Unlock()
	cachedDriverPodLabels = nil
	cachedDriverPodAnnotations = nil
	driverConfigInitialized = false
}

func TestInitDriverPodConfig(t *testing.T) {
	tests := []struct {
		name                string
		labels              map[string]string
		annotations         map[string]string
		expectedLabels      map[string]string
		expectedAnnotations map[string]string
	}{
		{
			name:                "empty config",
			labels:              nil,
			annotations:         nil,
			expectedLabels:      nil,
			expectedAnnotations: nil,
		},
		{
			name: "valid config with filtering",
			labels: map[string]string{
				"sidecar.istio.io/inject":       "true",
				"pipelines.kubeflow.org/system": "reserved",
				"app":                           "test",
			},
			annotations: map[string]string{
				"proxy.istio.io/config": "{\"holdApplicationUntilProxyStarts\":true}",
			},
			expectedLabels: map[string]string{
				"sidecar.istio.io/inject": "true",
				"app":                     "test",
			},
			expectedAnnotations: map[string]string{
				"proxy.istio.io/config": "{\"holdApplicationUntilProxyStarts\":true}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper and driver config state
			viper.Reset()
			resetDriverConfig()

			if tt.labels != nil {
				viper.Set(DriverPodLabels, tt.labels)
			}
			if tt.annotations != nil {
				viper.Set(DriverPodAnnotations, tt.annotations)
			}

			// Initialize driver config
			InitDriverPodConfig()

			// Verify the config was initialized
			assert.True(t, driverConfigInitialized)

			// Verify labels
			labels := GetDriverPodLabels()
			assert.Equal(t, tt.expectedLabels, labels)

			// Verify annotations
			annotations := GetDriverPodAnnotations()
			assert.Equal(t, tt.expectedAnnotations, annotations)
		})
	}
}

func TestGetDriverPodLabels(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]string
		expected map[string]string
	}{
		{
			name:     "empty config",
			config:   nil,
			expected: nil,
		},
		{
			name: "valid labels",
			config: map[string]string{
				"sidecar.istio.io/inject": "true",
				"app":                     "test",
			},
			expected: map[string]string{
				"sidecar.istio.io/inject": "true",
				"app":                     "test",
			},
		},
		{
			name: "filters reserved labels",
			config: map[string]string{
				"sidecar.istio.io/inject":       "true",
				"pipelines.kubeflow.org/system": "reserved",
				"app":                           "test",
			},
			expected: map[string]string{
				"sidecar.istio.io/inject": "true",
				"app":                     "test",
			},
		},
		{
			name: "all reserved labels",
			config: map[string]string{
				"pipelines.kubeflow.org/system": "reserved",
				"pipelines.kubeflow.org/task":   "reserved",
			},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper and driver config state
			viper.Reset()
			resetDriverConfig()

			if tt.config != nil {
				viper.Set(DriverPodLabels, tt.config)
			}

			// Initialize driver config to load from Viper
			InitDriverPodConfig()

			result := GetDriverPodLabels()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDriverPodAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]string
		expected map[string]string
	}{
		{
			name:     "empty config",
			config:   nil,
			expected: nil,
		},
		{
			name: "valid annotations",
			config: map[string]string{
				"proxy.istio.io/config": "{\"holdApplicationUntilProxyStarts\":true}",
				"custom":                "annotation",
			},
			expected: map[string]string{
				"proxy.istio.io/config": "{\"holdApplicationUntilProxyStarts\":true}",
				"custom":                "annotation",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper and driver config state
			viper.Reset()
			resetDriverConfig()

			if tt.config != nil {
				viper.Set(DriverPodAnnotations, tt.config)
			}

			// Initialize driver config to load from Viper
			InitDriverPodConfig()

			result := GetDriverPodAnnotations()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterReservedLabels(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "no reserved labels",
			input: map[string]string{
				"app": "test",
				"env": "prod",
			},
			expected: map[string]string{
				"app": "test",
				"env": "prod",
			},
		},
		{
			name: "mixed labels",
			input: map[string]string{
				"app":                           "test",
				"pipelines.kubeflow.org/system": "reserved",
				"env":                           "prod",
			},
			expected: map[string]string{
				"app": "test",
				"env": "prod",
			},
		},
		{
			name: "all reserved",
			input: map[string]string{
				"pipelines.kubeflow.org/system": "reserved",
				"pipelines.kubeflow.org/task":   "reserved",
			},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterReservedLabels(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
