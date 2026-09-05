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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConfigFile loads a config.json without the environment lookup, since an environment
// variable would take precedence over the file.
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

// t.Setenv can only set a value, so a test needing a variable absent has to
// unset it and put back whatever was there.
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

// viper.GetStringMapString swallows a non-string value as an empty map.
func TestInitDriverPodConfig_NonStringJSONValues(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	// Boolean instead of string, the most realistic ConfigMap typo.
	viper.Set(DriverPodLabels, `{"sidecar.istio.io/inject":true}`)

	err := InitDriverPodConfig()

	require.Error(t, err, "a value that is not a string should fail startup when set via ConfigMap")
	assert.Contains(t, err.Error(), DriverPodLabels)
}

// Viper turns a null into an empty string rather than dropping it, so this one
// needs a check of its own.
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

// A bare null unmarshals into a nil map without an error, so the per entry check never runs.
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

// The same check on the path a real install takes, where the value arrives as an
// environment variable rather than being set on Viper.
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

// The manifest default ships both keys as empty strings.
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

// The Deployment passes the ConfigMap keys as the uppercased forms of the Viper
// keys. Renaming one without the other silently stops the values arriving, and
// this is what catches it.
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

// Viper runs with AllowEmptyEnv, so the empty variable every default install sets
// still counts as set.
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

func TestInitDriverPodConfig_AnnotationKeyCaseFollowsKubernetes(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodAnnotations, map[string]string{"Example.com/Name": "value"})

	require.NoError(t, InitDriverPodConfig(), "Kubernetes accepts this annotation key, so it must not fail startup")
	assert.Equal(t, "value", GetDriverPodAnnotations()["Example.com/Name"])
}

// Each key and value is acceptable on its own, so only the total catches this.
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

// TestInitDriverPodConfig_ReservedAnnotationsNotCounted pins the order of the two checks:
// reserved entries never reach the pod, so they must be dropped before the size is measured.
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

// Annotation values are free form, so only the keys are checked.
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

// The example from backend/README.md verbatim, so its escaping fails here rather
// than in an install.
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

func TestInitDriverPodConfig_ConfigFileStringFormKeepsKeyCase(t *testing.T) {
	writeConfigFile(t, `{"DRIVER_POD_LABELS": "{\"example.com/BuildID\":\"123\"}"}`)

	require.NoError(t, InitDriverPodConfig())

	assert.Equal(t, map[string]string{"example.com/BuildID": "123"}, GetDriverPodLabels())
}

func TestInitDriverPodConfig_ConfigFileStringFormKeepsBothCases(t *testing.T) {
	writeConfigFile(t, `{"DRIVER_POD_LABELS": "{\"example.com/BuildID\":\"123\",\"example.com/buildid\":\"456\"}"}`)

	require.NoError(t, InitDriverPodConfig())

	assert.Equal(t, map[string]string{
		"example.com/BuildID": "123",
		"example.com/buildid": "456",
	}, GetDriverPodLabels())
}

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
				assert.Contains(t, err.Error(), "must be a JSON string",
					"the error should name the form the operator must use instead")
				assert.Nil(t, GetDriverPodConfig(), "a refused value must not reach the cache")
			}
		})
	}
}

func TestInitDriverPodConfig_ConfigFileObjectHidesValues(t *testing.T) {
	writeConfigFile(t, `{"DRIVER_POD_LABELS": {"Team": "ml", "team": null}}`)

	got, ok := viper.Get(DriverPodLabels).(map[string]any)
	require.True(t, ok, "Viper should return a map for a JSON object")
	assert.Len(t, got, 1, "case-differing keys merge, proving one value is hidden")
	_, hasTeam := got["team"]
	assert.True(t, hasTeam, "the surviving key should be lowercased")
}

func TestInitDriverPodConfig_StringFormSeesBothCases(t *testing.T) {
	writeConfigFile(t, `{"DRIVER_POD_LABELS": "{\"Team\":\"ml\",\"team\":null}"}`)

	err := InitDriverPodConfig()

	require.Error(t, err, "the null must be seen even though another key differs only in case")
	assert.Contains(t, err.Error(), "team")
}

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

func TestInitDriverPodConfig_ConfigFileNativeNullMeansUnset(t *testing.T) {
	writeConfigFile(t, `{"DRIVER_POD_LABELS": null}`)

	require.False(t, viper.IsSet(DriverPodLabels), "Viper reports a native null as not set")
	require.NoError(t, InitDriverPodConfig())
	assert.Nil(t, GetDriverPodConfig())
}

// A configuration file cannot produce this shape, so its keys arrive as written.
func TestInitDriverPodConfig_ProgrammaticMapAccepted(t *testing.T) {
	viper.Reset()
	resetDriverConfig()

	viper.Set(DriverPodLabels, map[string]string{"sidecar.istio.io/inject": "true"})

	require.NoError(t, InitDriverPodConfig())

	assert.Equal(t, "true", GetDriverPodLabels()["sidecar.istio.io/inject"])
}

// Argo's default sidecar kill command is one the Istio proxy does not answer, so
// an operator using both needs this annotation to survive key validation.
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
