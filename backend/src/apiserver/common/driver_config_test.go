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

// Do not add t.Parallel() to these tests, because they share global Viper and package level state.
package common

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetDriverConfig resets the driver config state for testing
func resetDriverConfig() {
	driverConfigMutex.Lock()
	defer driverConfigMutex.Unlock()
	cachedDriverPodConfig = nil
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
			require.NoError(t, InitDriverPodConfig())

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
			require.NoError(t, InitDriverPodConfig())

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
			require.NoError(t, InitDriverPodConfig())

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

	require.NoError(t, InitDriverPodConfig())

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

// TestInitDriverPodConfig_NonStringJSONValues verifies that a ConfigMap value whose
// JSON contains a value that is not a string (for example a boolean) fails startup
// instead of being silently swallowed as an empty map by viper.GetStringMapString.
func TestInitDriverPodConfig_NonStringJSONValues(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	// Boolean instead of string, the most realistic ConfigMap typo.
	viper.Set(DriverPodLabels, `{"sidecar.istio.io/inject":true}`)

	err := InitDriverPodConfig()

	require.Error(t, err, "a value that is not a string should fail startup when set via ConfigMap")
	assert.Contains(t, err.Error(), DriverPodLabels)
}

// TestInitDriverPodConfig_NullJSONValue guards a case that is easy to miss. Viper does not
// drop a JSON null the way it drops a boolean or a number. It quietly turns null into an
// empty string, so without an explicit check the operator would get a label with an empty
// value and no warning at all. Someone writing null for the Istio injection flag would
// believe injection was enabled while the driver pod carried an empty value instead.
func TestInitDriverPodConfig_NullJSONValue(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodLabels, `{"sidecar.istio.io/inject":null}`)

	err := InitDriverPodConfig()

	require.Error(t, err, "a JSON null should fail startup rather than become an empty label value")
	assert.Contains(t, err.Error(), DriverPodLabels)
	assert.Contains(t, err.Error(), "sidecar.istio.io/inject")
}

// TestInitDriverPodConfig_NullMixedWithValidValue covers the same problem when the null
// sits alongside a perfectly good entry, which is where it is easiest to overlook.
func TestInitDriverPodConfig_NullMixedWithValidValue(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodLabels, `{"team":"ml","sidecar.istio.io/inject":null}`)

	err := InitDriverPodConfig()

	require.Error(t, err, "a JSON null mixed with a valid value should still fail startup")
	assert.Contains(t, err.Error(), "sidecar.istio.io/inject")
}

// TestInitDriverPodConfig_MalformedJSON verifies that a syntactically invalid JSON
// string in the ConfigMap fails startup with a clear error.
func TestInitDriverPodConfig_MalformedJSON(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodAnnotations, "{bad json}")

	err := InitDriverPodConfig()

	require.Error(t, err, "malformed JSON should fail at startup")
	assert.Contains(t, err.Error(), DriverPodAnnotations)
}

// TestInitDriverPodConfig_EmptyJSONObjectIsValid guards the edge case that a valid but
// empty JSON object ("{}") is a legitimate configuration that does nothing, not a parse failure.
func TestInitDriverPodConfig_EmptyJSONObjectIsValid(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodLabels, "{}")
	viper.Set(DriverPodAnnotations, "{}")

	require.NoError(t, InitDriverPodConfig())
	assert.Nil(t, GetDriverPodLabels())
	assert.Nil(t, GetDriverPodAnnotations())
}

// TestInitDriverPodConfig_BlankValuesAreValid guards the manifest default, where the
// ConfigMap ships DriverPodLabels and DriverPodAnnotations as empty strings. A blank or
// whitespace value must initialize cleanly with no configuration and never fail startup.
func TestInitDriverPodConfig_BlankValuesAreValid(t *testing.T) {
	for _, raw := range []string{"", "   "} {
		viper.Reset()
		resetDriverConfig()
		viper.Set(DriverPodLabels, raw)
		viper.Set(DriverPodAnnotations, raw)

		require.NoError(t, InitDriverPodConfig(), "blank value %q should not fail startup", raw)
		assert.Nil(t, GetDriverPodConfig(), "blank value %q should yield no configuration", raw)
	}
}

// TestInitDriverPodConfig_FromEnvVarWiring locks the contract between the API server
// Deployment and Viper. The Deployment wires the pipeline-install-config keys into the
// container as the environment variables DRIVERPODLABELS and DRIVERPODANNOTATIONS,
// which are the uppercased forms of the Viper keys below. Without that wiring the
// ConfigMap values never reach the API server. If the Viper keys are ever renamed, the
// Deployment must be updated to match, and this test is what will catch it.
func TestInitDriverPodConfig_FromEnvVarWiring(t *testing.T) {
	viper.Reset()
	resetDriverConfig()
	t.Cleanup(func() {
		viper.Reset()
		resetDriverConfig()
	})

	// Same Viper setup as initConfig() in main.go.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)

	t.Setenv(strings.ToUpper(DriverPodLabels), `{"sidecar.istio.io/inject":"true"}`)
	t.Setenv(strings.ToUpper(DriverPodAnnotations), `{"proxy.istio.io/config":"hold"}`)

	require.NoError(t, InitDriverPodConfig())

	assert.Equal(t, map[string]string{"sidecar.istio.io/inject": "true"}, GetDriverPodLabels())
	assert.Equal(t, map[string]string{"proxy.istio.io/config": "hold"}, GetDriverPodAnnotations())
}

// TestInitDriverPodConfig_EmptyEnvVarIsValid guards the default install. Every manifest
// ships DriverPodLabels and DriverPodAnnotations as empty strings, and the Deployment
// passes them to the container as empty environment variables. Viper runs with
// AllowEmptyEnv, so an empty variable still counts as set. Startup must succeed with no
// configuration, otherwise every default install would fail to start.
func TestInitDriverPodConfig_EmptyEnvVarIsValid(t *testing.T) {
	viper.Reset()
	resetDriverConfig()
	t.Cleanup(func() {
		viper.Reset()
		resetDriverConfig()
	})

	// Same Viper setup as initConfig() in main.go.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)

	t.Setenv(strings.ToUpper(DriverPodLabels), "")
	t.Setenv(strings.ToUpper(DriverPodAnnotations), "")

	require.NoError(t, InitDriverPodConfig(), "the shipped empty default must not fail startup")
	assert.Nil(t, GetDriverPodConfig(), "empty defaults should yield no configuration")
}

// TestInitDriverPodConfig_InvalidLabelKey verifies that a label key Kubernetes would
// reject fails at startup instead of when the driver pod is created.
func TestInitDriverPodConfig_InvalidLabelKey(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodLabels, map[string]string{"not a valid key!": "x"})

	err := InitDriverPodConfig()

	require.Error(t, err, "an invalid label key should fail startup")
	assert.Contains(t, err.Error(), "invalid label key")
}

// TestInitDriverPodConfig_InvalidLabelValue verifies that a label value Kubernetes
// would reject fails at startup.
func TestInitDriverPodConfig_InvalidLabelValue(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodLabels, map[string]string{"app": "spaces are not allowed"})

	err := InitDriverPodConfig()

	require.Error(t, err, "an invalid label value should fail startup")
	assert.Contains(t, err.Error(), "invalid label value")
}

// TestInitDriverPodConfig_InvalidAnnotationKey verifies that an annotation key
// Kubernetes would reject fails at startup.
func TestInitDriverPodConfig_InvalidAnnotationKey(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodAnnotations, map[string]string{"bad key!": "value"})

	err := InitDriverPodConfig()

	require.Error(t, err, "an invalid annotation key should fail startup")
	assert.Contains(t, err.Error(), "invalid annotation key")
}

// TestInitDriverPodConfig_IstioMetadataAccepted guards the primary use case. Annotation
// values are free form and often carry JSON, so validation must check annotation keys
// only and must never reject a value like the Istio proxy config below.
func TestInitDriverPodConfig_IstioMetadataAccepted(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodLabels, map[string]string{"sidecar.istio.io/inject": "true"})
	viper.Set(DriverPodAnnotations, map[string]string{
		"proxy.istio.io/config": `{"holdApplicationUntilProxyStarts":true}`,
	})

	require.NoError(t, InitDriverPodConfig())

	assert.Equal(t, "true", GetDriverPodLabels()["sidecar.istio.io/inject"])
	assert.Equal(t, `{"holdApplicationUntilProxyStarts":true}`, GetDriverPodAnnotations()["proxy.istio.io/config"])
}

func TestGetDriverPodLabelsNotInitialized(t *testing.T) {
	resetDriverConfig()

	labels := GetDriverPodLabels()
	assert.Nil(t, labels, "should return nil when not initialized")

	annotations := GetDriverPodAnnotations()
	assert.Nil(t, annotations, "should return nil when not initialized")

	config := GetDriverPodConfig()
	assert.Nil(t, config, "should return nil when not initialized")
}

func TestGetDriverPodConfig(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodLabels, map[string]string{"app": "driver"})
	viper.Set(DriverPodAnnotations, map[string]string{"note": "value"})
	require.NoError(t, InitDriverPodConfig())

	config := GetDriverPodConfig()
	require.NotNil(t, config)
	assert.Equal(t, map[string]string{"app": "driver"}, config.Labels)
	assert.Equal(t, map[string]string{"note": "value"}, config.Annotations)
}

func TestGetDriverPodConfigNilWhenAllEmpty(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	// Only reserved keys are configured, so everything is filtered out.
	viper.Set(DriverPodLabels, map[string]string{"pipelines.kubeflow.org/system": "reserved"})
	require.NoError(t, InitDriverPodConfig())

	assert.Nil(t, GetDriverPodConfig(), "should return nil when no labels or annotations remain after filtering")
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
			// Verify it is a true copy (not the same reference) for input that is not nil
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
	require.NoError(t, InitDriverPodConfig())

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
