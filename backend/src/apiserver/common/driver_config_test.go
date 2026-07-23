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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConfigFile writes a config.json into a temporary directory and loads it through Viper
// so tests can exercise the configuration file on its own. It deliberately does not set up the
// environment lookup the API server also enables, because an environment variable takes
// precedence over the file; TestInitDriverPodConfig_EmptyEnvVarShadowsConfigFile covers that.
func writeConfigFile(t *testing.T, contents string) {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(contents), 0o600))

	viper.Reset()
	resetDriverConfig()
	t.Cleanup(func() {
		viper.Reset()
		resetDriverConfig()
	})

	viper.SetConfigName("config")
	viper.AddConfigPath(dir)
	require.NoError(t, viper.ReadInConfig())
}

// unsetEnvForTest removes the named variables for the duration of the test and puts back
// whatever was there before. t.Setenv can only set a value, so a test that needs a variable to
// be absent has to do this itself or inherit whatever started it.
func unsetEnvForTest(t *testing.T, names ...string) {
	t.Helper()

	for _, name := range names {
		if previous, ok := os.LookupEnv(name); ok {
			t.Cleanup(func() { _ = os.Setenv(name, previous) })
		} else {
			t.Cleanup(func() { _ = os.Unsetenv(name) })
		}
		require.NoError(t, os.Unsetenv(name))
	}
}

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

// TestInitDriverPodConfig_TopLevelNull covers the document itself being null rather than one
// of its values. A bare null unmarshals into a nil map without error, so the per entry check
// has nothing to walk and the value used to be accepted while the feature stayed off. That is
// the same silent outcome the README promises will never happen, so it must fail startup for
// both keys.
func TestInitDriverPodConfig_TopLevelNull(t *testing.T) {
	for _, name := range []string{DriverPodLabels, DriverPodAnnotations} {
		t.Run(name, func(t *testing.T) {
			viper.Reset()
			resetDriverConfig()

			viper.Set(name, "null")

			err := InitDriverPodConfig()

			require.Error(t, err, "a null document should fail startup rather than be accepted")
			assert.Contains(t, err.Error(), name)
			assert.Nil(t, GetDriverPodConfig(), "a rejected value must not reach the cache")
		})
	}
}

// TestInitDriverPodConfig_TopLevelNullFromEnvVar repeats the check on the path a real install
// actually uses. The ConfigMap value reaches the API server as an environment variable, so the
// rejection has to hold there and not only for a value set directly on Viper.
func TestInitDriverPodConfig_TopLevelNullFromEnvVar(t *testing.T) {
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

	t.Setenv(strings.ToUpper(DriverPodLabels), "null")

	err := InitDriverPodConfig()

	require.Error(t, err, "a null delivered as an environment variable should fail startup")
	assert.Contains(t, err.Error(), DriverPodLabels)
}

// TestInitDriverPodConfig_NotAJSONObject checks the whole family of values that parse as JSON
// but are not an object, so the rejection is not limited to the null that prompted it.
func TestInitDriverPodConfig_NotAJSONObject(t *testing.T) {
	for _, tt := range []struct {
		name string
		raw  string
	}{
		{"null", "null"},
		{"boolean", "true"},
		{"number", "42"},
		{"string", `"a string"`},
		{"array", `["a","b"]`},
		{"array of objects", `[{"app":"driver"}]`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			resetDriverConfig()

			viper.Set(DriverPodLabels, tt.raw)

			err := InitDriverPodConfig()

			require.Error(t, err, "%s is not a JSON object and should fail startup", tt.raw)
			assert.Contains(t, err.Error(), DriverPodLabels)
		})
	}
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
// ConfigMap ships DRIVER_POD_LABELS and DRIVER_POD_ANNOTATIONS as empty strings. A blank or
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
// container as the environment variables DRIVER_POD_LABELS and DRIVER_POD_ANNOTATIONS,
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
// ships DRIVER_POD_LABELS and DRIVER_POD_ANNOTATIONS as empty strings, and the Deployment
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
	assert.Contains(t, err.Error(), DriverPodAnnotations)
	assert.Contains(t, err.Error(), "bad key!", "the error should name the key that was refused")
}

// TestInitDriverPodConfig_AnnotationKeyCaseFollowsKubernetes pins the rule that annotation keys
// have their syntax validated case insensitively. Kubernetes lowers the key before checking it,
// which labels do
// not do, so a prefix such as Example.com is a valid annotation key and an invalid label key.
// Reimplementing the check by hand quietly refused these, which would have failed startup for a
// configuration Kubernetes itself accepts.
func TestInitDriverPodConfig_AnnotationKeyCaseFollowsKubernetes(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodAnnotations, map[string]string{"Example.com/Name": "value"})

	require.NoError(t, InitDriverPodConfig(), "Kubernetes accepts this annotation key, so it must not fail startup")
	assert.Equal(t, "value", GetDriverPodAnnotations()["Example.com/Name"])
}

// TestInitDriverPodConfig_AnnotationsTooLarge covers the limit Kubernetes puts on the
// combined annotations of one object. Keys and values are each individually acceptable
// here, so only the total catches it. Without this check the API server would start, the
// workflow would compile, and the failure would surface only when Argo asked Kubernetes to
// create the driver pod.
func TestInitDriverPodConfig_AnnotationsTooLarge(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodAnnotations, map[string]string{
		"example.com/payload": strings.Repeat("x", 300*1024),
	})

	err := InitDriverPodConfig()

	require.Error(t, err, "annotations larger than the Kubernetes limit should fail startup")
	assert.Contains(t, err.Error(), DriverPodAnnotations)
	assert.Nil(t, GetDriverPodConfig(), "a rejected value must not reach the cache")
}

// TestInitDriverPodConfig_AnnotationsAtLimitAccepted guards the other side of the bound, so the
// startup check cannot quietly become stricter than Kubernetes. It says the configured
// annotations alone are within the limit, which is the only thing this layer can know. The
// driver pod also carries annotations added by the compiler, so a configuration that fills the
// budget exactly here would be refused when the pod is created, and a value anywhere near this
// size cannot be delivered in the first place, since each key arrives as a single environment
// variable and Linux caps one of those at roughly 128 KiB. Reserving a margin for the compiler
// annotations was considered and rejected, because their size varies and guessing it would
// refuse configurations that would in fact have fitted. What this case pins is only that the
// check does not become stricter than Kubernetes.
func TestInitDriverPodConfig_AnnotationsAtLimitAccepted(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	const key = "example.com/payload"
	value := strings.Repeat("x", 256*1024-len(key))
	viper.Set(DriverPodAnnotations, map[string]string{key: value})

	require.NoError(t, InitDriverPodConfig(),
		"the configured annotations alone are within the limit, so startup validation must accept them")
	assert.Len(t, GetDriverPodAnnotations()[key], len(value))
}

// TestInitDriverPodConfig_ReservedAnnotationsNotCounted pins the order the two checks run in.
// Reserved entries are dropped before the driver pod is built, so they never occupy any of the
// budget Kubernetes measures, and counting them would refuse a configuration that would in fact
// have fitted. Reversing the order would be easy to do by accident and hard to notice.
func TestInitDriverPodConfig_ReservedAnnotationsNotCounted(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodAnnotations, map[string]string{
		ReservedLabelPrefix + "oversized": strings.Repeat("x", 300*1024),
		"example.com/kept":                "value",
	})

	require.NoError(t, InitDriverPodConfig(), "a reserved entry must not count toward the limit")
	assert.Equal(t, map[string]string{"example.com/kept": "value"}, GetDriverPodAnnotations())
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

// TestInitDriverPodConfig_ConfigFileExample loads the configuration file example printed in
// backend/README.md exactly as it appears there. The escaping is three layers deep, which is
// easy to get wrong in documentation and impossible to notice until an install silently has
// no driver metadata, so the example is kept under test rather than only proofread.
func TestInitDriverPodConfig_ConfigFileExample(t *testing.T) {
	writeConfigFile(t, `{
  "DRIVER_POD_LABELS": "{\"sidecar.istio.io/inject\":\"true\",\"app\":\"ml-pipeline-driver\"}",
  "DRIVER_POD_ANNOTATIONS": "{\"proxy.istio.io/config\":\"{\\\"holdApplicationUntilProxyStarts\\\":true}\"}"
}`)

	require.NoError(t, InitDriverPodConfig())

	assert.Equal(t, "true", GetDriverPodLabels()["sidecar.istio.io/inject"])
	assert.Equal(t, "ml-pipeline-driver", GetDriverPodLabels()["app"])
	assert.Equal(t, `{"holdApplicationUntilProxyStarts":true}`,
		GetDriverPodAnnotations()["proxy.istio.io/config"])
}

// TestInitDriverPodConfig_ConfigFileStringFormKeepsKeyCase is the reason the configuration
// file example is written as a JSON string. Viper folds the keys of a nested JSON object to
// lower case, which would rewrite an operator's metadata on the way through and silently drop
// one of two keys that differ only in case. Parsing the string ourselves avoids both.
func TestInitDriverPodConfig_ConfigFileStringFormKeepsKeyCase(t *testing.T) {
	writeConfigFile(t, `{"DRIVER_POD_LABELS": "{\"example.com/BuildID\":\"123\"}"}`)

	require.NoError(t, InitDriverPodConfig())

	assert.Equal(t, map[string]string{"example.com/BuildID": "123"}, GetDriverPodLabels())
}

// TestInitDriverPodConfig_ConfigFileStringFormKeepsBothCases is the case the nested object cannot
// express at all. Two keys differing only in case survive as two keys through the string form,
// where folding them to lower case would merge them and leave whichever value happened to be
// written last.
func TestInitDriverPodConfig_ConfigFileStringFormKeepsBothCases(t *testing.T) {
	writeConfigFile(t, `{"DRIVER_POD_LABELS": "{\"example.com/BuildID\":\"123\",\"example.com/buildid\":\"456\"}"}`)

	require.NoError(t, InitDriverPodConfig())

	assert.Equal(t, map[string]string{
		"example.com/BuildID": "123",
		"example.com/buildid": "456",
	}, GetDriverPodLabels())
}

// TestInitDriverPodConfig_ConfigFileObjectRefused covers the shape this configuration cannot
// check and therefore no longer accepts. The configuration loader lowers the keys of a JSON
// object written directly in the file, before any of this code runs, so a value that would be
// refused can be merged away and never seen. TestInitDriverPodConfig_ConfigFileObjectHidesValues
// below demonstrates that. Refusing the shape and naming the string form is the only answer that
// keeps the promise that every value is checked.
func TestInitDriverPodConfig_ConfigFileObjectRefused(t *testing.T) {
	for _, tt := range []struct{ name, body string }{
		{"all values are strings", `{"sidecar.istio.io/inject": "true"}`},
		{"null", `{"sidecar.istio.io/inject": null}`},
		{"boolean", `{"enabled": true}`},
		{"number", `{"replicas": 3}`},
		{"list", `{"items": ["a","b"]}`},
		{"nested object", `{"outer": {"inner": "v"}}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{DriverPodLabels, DriverPodAnnotations} {
				writeConfigFile(t, `{"`+key+`": `+tt.body+`}`)

				err := InitDriverPodConfig()

				require.Error(t, err, "%s written as an object in %s should fail startup", tt.name, key)
				assert.Contains(t, err.Error(), key)
				assert.Contains(t, err.Error(), "written as a string",
					"the error should name the form the operator must use instead")
				assert.Nil(t, GetDriverPodConfig(), "a refused value must not reach the cache")
			}
		})
	}
}

// TestInitDriverPodConfig_ConfigFileObjectHidesValues is the evidence behind refusing the
// object form rather than checking it entry by entry. The loader lowers the keys while reading
// the file, so two keys differing only in case become one and one of the values is dropped.
// Where a single key needs lowering the outcome is fixed and the refused value simply vanishes;
// where both need it the survivor follows map iteration order, so the same file would start the
// API server on one restart and stop it on the next. Neither is visible to anything downstream,
// which is why the shape is refused rather than validated.
// A second shape, two keys both needing to be lowered, was measured while deciding this and
// produced either survivor across repeated reads, since map iteration order picks the winner.
// It is left out of the assertions here because a test that depends on that order can only be
// flaky, and the case below already shows a refused value being merged away.
func TestInitDriverPodConfig_ConfigFileObjectHidesValues(t *testing.T) {
	const runs = 40

	seen := map[string]int{}
	for i := 0; i < runs; i++ {
		writeConfigFile(t, `{"DRIVER_POD_LABELS": {"Team": "ml", "team": null}}`)
		seen[fmt.Sprintf("%v", viper.Get(DriverPodLabels))]++
	}

	require.Len(t, seen, 1, "one key needing to be lowered collides deterministically")
	assert.Contains(t, fmt.Sprintf("%v", seen), "ml",
		"the null is merged away before anything downstream can refuse it")
}

// TestInitDriverPodConfig_StringFormSeesBothCases is the other half of the argument. The string
// form is parsed here rather than by the configuration loader, so the keys arrive as written and
// a refused value cannot hide behind one that is accepted.
func TestInitDriverPodConfig_StringFormSeesBothCases(t *testing.T) {
	writeConfigFile(t, `{"DRIVER_POD_LABELS": "{\"Team\":\"ml\",\"team\":null}"}`)

	err := InitDriverPodConfig()

	require.Error(t, err, "the null must be seen even though another key differs only in case")
	assert.Contains(t, err.Error(), "team")
}

// TestInitDriverPodConfig_EmptyEnvVarShadowsConfigFile records how the two sources actually
// combine on a standard install, which is not what the wording "configure through the
// configuration file or the ConfigMap" suggests on its own. Viper puts an environment variable
// ahead of the configuration file, AllowEmptyEnv makes an empty one count as a value, and the
// Deployment always passes both keys in from the ConfigMap, which ships them empty. So the
// configuration file supplies the built in default and the ConfigMap is what an operator
// actually controls. backend/README.md says the same thing in prose.
func TestInitDriverPodConfig_EmptyEnvVarShadowsConfigFile(t *testing.T) {
	// The first half asserts what happens with no variable set, so a variable inherited from
	// whoever started the test would quietly invalidate it.
	unsetEnvForTest(t, strings.ToUpper(DriverPodLabels), strings.ToUpper(DriverPodAnnotations))

	writeConfigFile(t, `{"DRIVER_POD_LABELS": "{\"sidecar.istio.io/inject\":\"true\"}"}`)

	// Same Viper setup as initConfig() in main.go, which the configuration file tests above
	// leave out because they exercise the file on its own.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)

	require.NoError(t, InitDriverPodConfig())
	require.Equal(t, "true", GetDriverPodLabels()["sidecar.istio.io/inject"],
		"without the Deployment variables the configuration file applies")

	t.Setenv(strings.ToUpper(DriverPodLabels), "")
	resetDriverConfig()

	require.NoError(t, InitDriverPodConfig())
	assert.Nil(t, GetDriverPodLabels(), "the empty Deployment variable takes precedence over the configuration file")
}

// TestInitDriverPodConfig_ConfigFileNativeNullMeansUnset records a deliberate boundary. Viper
// reports a key whose configuration file value is a native JSON null as not set at all, exactly
// as if the key were absent, and it offers no way to tell the two apart: InConfig cannot be used
// for this either, because the shipped default sets the variable to an empty string without any
// configuration file entry. Reading the file a second time to recover the distinction would put
// a second source of truth beside Viper for one input. Since a null there asks for no metadata
// and produces no metadata, the outcome matches the request, and the value that is rejected is
// the quoted string "null", which reads as a document rather than as nothing.
func TestInitDriverPodConfig_ConfigFileNativeNullMeansUnset(t *testing.T) {
	writeConfigFile(t, `{"DRIVER_POD_LABELS": null}`)

	require.False(t, viper.IsSet(DriverPodLabels), "Viper reports a native null as not set")
	require.NoError(t, InitDriverPodConfig())
	assert.Nil(t, GetDriverPodConfig())
}

// TestInitDriverPodConfig_ProgrammaticMapAccepted keeps the shape a caller can set directly.
// A configuration file cannot produce it, so its keys are as written and its values are already
// strings, which is what separates it from the object form refused above.
func TestInitDriverPodConfig_ProgrammaticMapAccepted(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodLabels, map[string]string{"sidecar.istio.io/inject": "true"})

	require.NoError(t, InitDriverPodConfig())

	assert.Equal(t, "true", GetDriverPodLabels()["sidecar.istio.io/inject"])
}

// TestInitDriverPodConfig_ArgoKillCommandAccepted covers the annotation an operator needs
// alongside sidecar injection. Argo kills an injected sidecar with /bin/sh -c 'kill 1' by
// default, which the Istio proxy does not answer, and a driver pod whose proxy keeps running
// never completes. The remedy is the per container kill command annotation documented by
// Argo, so this configuration has to survive key validation and reach the driver pod intact.
// backend/README.md points operators at exactly this value.
func TestInitDriverPodConfig_ArgoKillCommandAccepted(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	// Copied verbatim from the ConfigMap example in backend/README.md, so the escaping an
	// operator is told to write stays under test.
	viper.Set(DriverPodLabels, `{"sidecar.istio.io/inject":"true"}`)
	viper.Set(DriverPodAnnotations, `{"proxy.istio.io/config":"{\"holdApplicationUntilProxyStarts\":true}","workflows.argoproj.io/kill-cmd-istio-proxy":"[\"pilot-agent\", \"request\", \"POST\", \"quitquitquit\"]"}`)

	require.NoError(t, InitDriverPodConfig())

	annotations := GetDriverPodAnnotations()
	assert.Equal(t, "true", GetDriverPodLabels()["sidecar.istio.io/inject"])
	assert.Equal(t, `{"holdApplicationUntilProxyStarts":true}`, annotations["proxy.istio.io/config"])
	assert.Equal(t, `["pilot-agent", "request", "POST", "quitquitquit"]`, annotations["workflows.argoproj.io/kill-cmd-istio-proxy"])
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
