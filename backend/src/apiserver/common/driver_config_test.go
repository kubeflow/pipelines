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

// Do not add t.Parallel() to these tests -- they share global Viper and module-level state.
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
		{
			name: "filters reserved annotations too",
			labels: map[string]string{
				"app": "test",
			},
			annotations: map[string]string{
				"custom":                              "value",
				"pipelines.kubeflow.org/v2_component": "true",
			},
			expectedLabels: map[string]string{
				"app": "test",
			},
			expectedAnnotations: map[string]string{
				"custom": "value",
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
			name: "all reserved labels returns nil",
			config: map[string]string{
				"pipelines.kubeflow.org/system": "reserved",
				"pipelines.kubeflow.org/task":   "reserved",
			},
			expected: nil,
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
		{
			name: "filters reserved annotation prefix",
			config: map[string]string{
				"proxy.istio.io/config":               "value",
				"pipelines.kubeflow.org/v2_component": "true",
			},
			expected: map[string]string{
				"proxy.istio.io/config": "value",
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

func TestInitDriverPodConfigFromJSONString(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	// ConfigMap values arrive as JSON strings, not Go maps.
	// Viper's GetStringMapString must parse them correctly.
	viper.Set(DriverPodLabels, `{"sidecar.istio.io/inject":"true","app":"driver"}`)
	viper.Set(DriverPodAnnotations, `{"proxy.istio.io/config":"hold"}`)

	InitDriverPodConfig()

	labels := GetDriverPodLabels()
	assert.Equal(t, map[string]string{
		"sidecar.istio.io/inject": "true",
		"app":                     "driver",
	}, labels)

	annotations := GetDriverPodAnnotations()
	assert.Equal(t, map[string]string{
		"proxy.istio.io/config": "hold",
	}, annotations)
}

func TestGetDriverPodLabelsNotInitialized(t *testing.T) {
	resetDriverConfig()

	labels := GetDriverPodLabels()
	assert.Nil(t, labels, "should return nil when not initialized")

	annotations := GetDriverPodAnnotations()
	assert.Nil(t, annotations, "should return nil when not initialized")
}

func TestCopyMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name:     "nil returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty returns empty",
			input:    map[string]string{},
			expected: map[string]string{},
		},
		{
			name:     "copies entries",
			input:    map[string]string{"a": "1", "b": "2"},
			expected: map[string]string{"a": "1", "b": "2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := copyMap(tt.input)
			assert.Equal(t, tt.expected, result)
			// Verify it's a true copy (not same reference) for non-nil input
			if tt.input != nil {
				tt.input["mutated"] = "yes"
				assert.NotContains(t, result, "mutated")
			}
		})
	}
}

func TestGetDriverPodLabelsReturnsCopy(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodLabels, map[string]string{
		"app": "test",
	})
	InitDriverPodConfig()

	copy1 := GetDriverPodLabels()
	copy2 := GetDriverPodLabels()

	// Mutating one copy should not affect the other
	copy1["app"] = "mutated"
	assert.Equal(t, "test", copy2["app"], "returned maps should be independent copies")
}

func TestFilterReservedEntries(t *testing.T) {
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
			name:     "empty input returns nil",
			input:    map[string]string{},
			expected: nil,
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
			name: "all reserved returns nil",
			input: map[string]string{
				"pipelines.kubeflow.org/system": "reserved",
				"pipelines.kubeflow.org/task":   "reserved",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterReservedEntries(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
