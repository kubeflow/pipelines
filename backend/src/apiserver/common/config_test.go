// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package common

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: These tests use viper.Reset() which mutates the global viper singleton.
// Do not add t.Parallel() to these subtests — the shared viper state would race.

func TestGetStringConfigWithDefault(t *testing.T) {
	tests := []struct {
		name     string
		setEnv   bool
		envValue string
		expected string
	}{
		{
			name:     "returns default when config not set",
			setEnv:   false,
			expected: "my-default",
		},
		{
			name:     "returns config value when set via env var",
			setEnv:   true,
			envValue: "custom-value",
			expected: "custom-value",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			viper.Reset()

			if testCase.setEnv {
				t.Setenv("TEST_STRING_CONFIG", testCase.envValue)
				viper.AutomaticEnv()
			}

			result := GetStringConfigWithDefault("TEST_STRING_CONFIG", "my-default")
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestGetBoolConfigWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		setEnv       bool
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{
			name:         "returns default true when config not set",
			setEnv:       false,
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "returns default false when config not set",
			setEnv:       false,
			defaultValue: false,
			expected:     false,
		},
		{
			name:         "returns parsed true from env var",
			setEnv:       true,
			envValue:     "true",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "returns parsed false from env var",
			setEnv:       true,
			envValue:     "false",
			defaultValue: true,
			expected:     false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			viper.Reset()

			if testCase.setEnv {
				t.Setenv("TEST_BOOL_CONFIG", testCase.envValue)
				viper.AutomaticEnv()
			}

			result := GetBoolConfigWithDefault("TEST_BOOL_CONFIG", testCase.defaultValue)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestGetFloat64ConfigWithDefault(t *testing.T) {
	tests := []struct {
		name     string
		setEnv   bool
		envValue string
		expected float64
	}{
		{
			name:     "returns default when not set",
			setEnv:   false,
			expected: 3.14,
		},
		{
			name:     "returns float value when set",
			setEnv:   true,
			envValue: "2.718",
			expected: 2.718,
		},
		{
			name:     "returns zero when env var is not a valid float",
			setEnv:   true,
			envValue: "not-a-number",
			expected: 0,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			viper.Reset()

			if testCase.setEnv {
				t.Setenv("TEST_FLOAT_CONFIG", testCase.envValue)
				viper.AutomaticEnv()
			}

			result := GetFloat64ConfigWithDefault("TEST_FLOAT_CONFIG", 3.14)
			assert.InDelta(t, testCase.expected, result, 0.001)
		})
	}
}

func TestGetIntConfigWithDefault(t *testing.T) {
	tests := []struct {
		name     string
		setEnv   bool
		envValue string
		expected int
	}{
		{
			name:     "returns default when not set",
			setEnv:   false,
			expected: 42,
		},
		{
			name:     "returns int value when set",
			setEnv:   true,
			envValue: "100",
			expected: 100,
		},
		{
			name:     "returns zero when env var is not a valid int",
			setEnv:   true,
			envValue: "not-a-number",
			expected: 0,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			viper.Reset()

			if testCase.setEnv {
				t.Setenv("TEST_INT_CONFIG", testCase.envValue)
				viper.AutomaticEnv()
			}

			result := GetIntConfigWithDefault("TEST_INT_CONFIG", 42)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestGetMapConfig(t *testing.T) {
	tests := []struct {
		name     string
		setValue bool
		expected map[string]string
	}{
		{
			name:     "returns nil when config not set",
			setValue: false,
			expected: nil,
		},
		{
			name:     "returns map when config is set",
			setValue: true,
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			viper.Reset()

			if testCase.setValue {
				viper.Set("TEST_MAP_CONFIG", map[string]string{
					"key1": "value1",
					"key2": "value2",
				})
			}

			result := GetMapConfig("TEST_MAP_CONFIG")
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestGetBoolFromStringWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue bool
		expected     bool
	}{
		{
			name:         "true string returns true",
			value:        "true",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "false string returns false",
			value:        "false",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "1 returns true",
			value:        "1",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "empty string returns default value",
			value:        "",
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "invalid string returns default value",
			value:        "invalid",
			defaultValue: false,
			expected:     false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := GetBoolFromStringWithDefault(testCase.value, testCase.defaultValue)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestGetStringConfig_WhenSet(t *testing.T) {
	viper.Reset()
	t.Setenv("TEST_REQUIRED_STRING", "hello")
	viper.AutomaticEnv()

	result := GetStringConfig("TEST_REQUIRED_STRING")
	assert.Equal(t, "hello", result)
}

func TestGetDurationConfig_WhenSet(t *testing.T) {
	viper.Reset()
	t.Setenv("TEST_DURATION", "5s")
	viper.AutomaticEnv()

	result := GetDurationConfig("TEST_DURATION")
	assert.Equal(t, 5*time.Second, result)
}

func TestConfigWrapperDefaults(t *testing.T) {
	tests := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{
			name:     "IsPipelineVersionUpdatedByDefault defaults to true",
			getter:   func() interface{} { return IsPipelineVersionUpdatedByDefault() },
			expected: true,
		},
		{
			name:     "IsNamespaceRequiredForPipelines defaults to false",
			getter:   func() interface{} { return IsNamespaceRequiredForPipelines() },
			expected: false,
		},
		{
			name:     "IsMultiUserMode defaults to false",
			getter:   func() interface{} { return IsMultiUserMode() },
			expected: false,
		},
		{
			name:     "IsMultiUserSharedReadMode defaults to false",
			getter:   func() interface{} { return IsMultiUserSharedReadMode() },
			expected: false,
		},
		{
			name:     "GetPodNamespace defaults to kubeflow",
			getter:   func() interface{} { return GetPodNamespace() },
			expected: DefaultPodNamespace,
		},
		{
			name:     "IsCacheEnabled defaults to true string",
			getter:   func() interface{} { return IsCacheEnabled() },
			expected: "true",
		},
		{
			name:     "GetKubeflowUserIDHeader defaults to GoogleIAPUserIdentityHeader",
			getter:   func() interface{} { return GetKubeflowUserIDHeader() },
			expected: GoogleIAPUserIdentityHeader,
		},
		{
			name:     "GetKubeflowUserIDPrefix defaults to GoogleIAPUserIdentityPrefix",
			getter:   func() interface{} { return GetKubeflowUserIDPrefix() },
			expected: GoogleIAPUserIdentityPrefix,
		},
		{
			name:     "GetTokenReviewAudience defaults to DefaultTokenReviewAudience",
			getter:   func() interface{} { return GetTokenReviewAudience() },
			expected: DefaultTokenReviewAudience,
		},
		{
			name:     "GetMetadataTLSEnabled defaults to false",
			getter:   func() interface{} { return GetMetadataTLSEnabled() },
			expected: false,
		},
		{
			name:     "GetCaBundleSecretName defaults to empty string",
			getter:   func() interface{} { return GetCaBundleSecretName() },
			expected: "",
		},
		{
			name:     "GetCABundleKey defaults to empty string",
			getter:   func() interface{} { return GetCABundleKey() },
			expected: "",
		},
		{
			name:     "GetCaBundleConfigMapName defaults to empty string",
			getter:   func() interface{} { return GetCaBundleConfigMapName() },
			expected: "",
		},
		{
			name:     "GetCompiledPipelineSpecPatch defaults to empty JSON object",
			getter:   func() interface{} { return GetCompiledPipelineSpecPatch() },
			expected: "{}",
		},
		{
			name:     "GetDefaultSecurityContextRunAsUser defaults to empty string",
			getter:   func() interface{} { return GetDefaultSecurityContextRunAsUser() },
			expected: "",
		},
		{
			name:     "GetDefaultSecurityContextRunAsGroup defaults to empty string",
			getter:   func() interface{} { return GetDefaultSecurityContextRunAsGroup() },
			expected: "",
		},
		{
			name:     "GetDefaultSecurityContextRunAsNonRoot defaults to empty string",
			getter:   func() interface{} { return GetDefaultSecurityContextRunAsNonRoot() },
			expected: "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			viper.Reset()
			result := testCase.getter()
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestConfigWrapperCustomValues(t *testing.T) {
	tests := []struct {
		name        string
		envKey      string
		envValue    string
		useViperSet bool
		getter      func() interface{}
		expected    interface{}
	}{
		{
			name:     "IsPipelineVersionUpdatedByDefault with custom false",
			envKey:   UpdatePipelineVersionByDefault,
			envValue: "false",
			getter:   func() interface{} { return IsPipelineVersionUpdatedByDefault() },
			expected: false,
		},
		{
			name:     "IsNamespaceRequiredForPipelines with custom true",
			envKey:   RequireNamespaceForPipelines,
			envValue: "true",
			getter:   func() interface{} { return IsNamespaceRequiredForPipelines() },
			expected: true,
		},
		{
			name:     "IsMultiUserMode with custom true",
			envKey:   MultiUserMode,
			envValue: "true",
			getter:   func() interface{} { return IsMultiUserMode() },
			expected: true,
		},
		{
			name:     "IsMultiUserSharedReadMode with custom true",
			envKey:   MultiUserModeSharedReadAccess,
			envValue: "true",
			getter:   func() interface{} { return IsMultiUserSharedReadMode() },
			expected: true,
		},
		{
			name:     "GetPodNamespace with custom namespace",
			envKey:   PodNamespace,
			envValue: "custom-ns",
			getter:   func() interface{} { return GetPodNamespace() },
			expected: "custom-ns",
		},
		{
			name:        "IsCacheEnabled with custom false",
			envKey:      CacheEnabled,
			envValue:    "false",
			useViperSet: true, // CacheEnabled is mixed-case, env var lookup via AutomaticEnv uppercases the key
			getter:      func() interface{} { return IsCacheEnabled() },
			expected:    "false",
		},
		{
			name:     "GetKubeflowUserIDHeader with custom header",
			envKey:   KubeflowUserIDHeader,
			envValue: "x-custom-user-header",
			getter:   func() interface{} { return GetKubeflowUserIDHeader() },
			expected: "x-custom-user-header",
		},
		{
			name:     "GetKubeflowUserIDPrefix with custom prefix",
			envKey:   KubeflowUserIDPrefix,
			envValue: "custom-prefix:",
			getter:   func() interface{} { return GetKubeflowUserIDPrefix() },
			expected: "custom-prefix:",
		},
		{
			name:     "GetTokenReviewAudience with custom audience",
			envKey:   TokenReviewAudience,
			envValue: "custom.audience.org",
			getter:   func() interface{} { return GetTokenReviewAudience() },
			expected: "custom.audience.org",
		},
		{
			name:     "GetMetadataTLSEnabled with custom true",
			envKey:   MetadataTLSEnabled,
			envValue: "true",
			getter:   func() interface{} { return GetMetadataTLSEnabled() },
			expected: true,
		},
		{
			name:     "GetCaBundleSecretName with custom value",
			envKey:   CaBundleSecretName,
			envValue: "my-ca-secret",
			getter:   func() interface{} { return GetCaBundleSecretName() },
			expected: "my-ca-secret",
		},
		{
			name:     "GetCABundleKey with custom value",
			envKey:   CaBundleKeyName,
			envValue: "ca.crt",
			getter:   func() interface{} { return GetCABundleKey() },
			expected: "ca.crt",
		},
		{
			name:     "GetCaBundleConfigMapName with custom value",
			envKey:   CaBundleConfigMapName,
			envValue: "my-ca-configmap",
			getter:   func() interface{} { return GetCaBundleConfigMapName() },
			expected: "my-ca-configmap",
		},
		{
			name:     "GetCompiledPipelineSpecPatch with custom JSON",
			envKey:   CompiledPipelineSpecPatch,
			envValue: `{"nodeSelector":{"gpu":"true"}}`,
			getter:   func() interface{} { return GetCompiledPipelineSpecPatch() },
			expected: `{"nodeSelector":{"gpu":"true"}}`,
		},
		{
			name:     "GetDefaultSecurityContextRunAsUser with custom value",
			envKey:   DefaultSecurityContextRunAsUser,
			envValue: "1000",
			getter:   func() interface{} { return GetDefaultSecurityContextRunAsUser() },
			expected: "1000",
		},
		{
			name:     "GetDefaultSecurityContextRunAsGroup with custom value",
			envKey:   DefaultSecurityContextRunAsGroup,
			envValue: "2000",
			getter:   func() interface{} { return GetDefaultSecurityContextRunAsGroup() },
			expected: "2000",
		},
		{
			name:     "GetDefaultSecurityContextRunAsNonRoot with custom value",
			envKey:   DefaultSecurityContextRunAsNonRoot,
			envValue: "true",
			getter:   func() interface{} { return GetDefaultSecurityContextRunAsNonRoot() },
			expected: "true",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			viper.Reset()
			if testCase.useViperSet {
				viper.Set(testCase.envKey, testCase.envValue)
			} else {
				t.Setenv(testCase.envKey, testCase.envValue)
				viper.AutomaticEnv()
			}

			result := testCase.getter()
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestGetClusterDomain(t *testing.T) {
	tests := []struct {
		name           string
		envValue       string
		setEnv         bool
		expectedDomain string
	}{
		{
			name:           "returns default when env var not set",
			envValue:       "",
			setEnv:         false,
			expectedDomain: DefaultClusterDomain,
		},
		{
			name:           "returns default when env var is empty string",
			envValue:       "",
			setEnv:         true,
			expectedDomain: DefaultClusterDomain,
		},
		{
			name:           "returns custom domain when env var is set",
			envValue:       "cluster.test",
			setEnv:         true,
			expectedDomain: "cluster.test",
		},
		{
			name:           "returns custom domain with different TLD",
			envValue:       "my.custom.domain",
			setEnv:         true,
			expectedDomain: "my.custom.domain",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// Reset viper for each test
			viper.Reset()

			if testCase.setEnv {
				t.Setenv(ClusterDomain, testCase.envValue)
				viper.AutomaticEnv()
			}

			result := GetClusterDomain()
			assert.Equal(t, testCase.expectedDomain, result)
		})
	}
}

func TestGetMLPipelineServiceName(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		setEnv       bool
		expectedName string
	}{
		{
			name:         "returns default when env var not set",
			envValue:     "",
			setEnv:       false,
			expectedName: DefaultMLPipelineServiceName,
		},
		{
			name:         "returns default when env var is empty string",
			envValue:     "",
			setEnv:       true,
			expectedName: DefaultMLPipelineServiceName,
		},
		{
			name:         "returns custom name when env var is set",
			envValue:     "custom-ml-pipeline",
			setEnv:       true,
			expectedName: "custom-ml-pipeline",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			viper.Reset()

			if testCase.setEnv {
				t.Setenv(MLPipelineServiceName, testCase.envValue)
				viper.AutomaticEnv()
			}

			result := GetMLPipelineServiceName()
			assert.Equal(t, testCase.expectedName, result)
		})
	}
}

func TestGetMetadataServiceName(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		setEnv       bool
		expectedName string
	}{
		{
			name:         "returns default when env var not set",
			envValue:     "",
			setEnv:       false,
			expectedName: DefaultMetadataServiceName,
		},
		{
			name:         "returns default when env var is empty string",
			envValue:     "",
			setEnv:       true,
			expectedName: DefaultMetadataServiceName,
		},
		{
			name:         "returns custom name when env var is set",
			envValue:     "custom-metadata-service",
			setEnv:       true,
			expectedName: "custom-metadata-service",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			viper.Reset()

			if testCase.setEnv {
				t.Setenv(MetadataServiceName, testCase.envValue)
				viper.AutomaticEnv()
			}

			result := GetMetadataServiceName()
			assert.Equal(t, testCase.expectedName, result)
		})
	}
}

func TestGetPluginLimitsConfigDefaults(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	limits, err := GetPluginLimitsConfig()
	require.NoError(t, err)

	assert.Equal(t, DefaultPluginMaxKeys, limits.MaxKeys)
	assert.Equal(t, DefaultPluginMaxPayloadBytes, limits.MaxPayloadBytes)
	assert.Equal(t, DefaultPluginMaxTotalPayloadBytes, limits.MaxTotalPayloadBytes)
	assert.Equal(t, DefaultPluginMaxNestingDepth, limits.MaxNestingDepth)
}

func TestGetPluginLimitsConfigOverrides(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	viper.Set(PluginMaxKeys, "8")
	viper.Set(PluginMaxPayloadBytes, "32768")
	viper.Set(PluginMaxTotalPayloadBytes, "131072")
	viper.Set(PluginMaxNestingDepth, "6")

	limits, err := GetPluginLimitsConfig()
	require.NoError(t, err)

	assert.Equal(t, 8, limits.MaxKeys)
	assert.Equal(t, 32768, limits.MaxPayloadBytes)
	assert.Equal(t, 131072, limits.MaxTotalPayloadBytes)
	assert.Equal(t, 6, limits.MaxNestingDepth)
}

func TestGetPluginLimitsConfigRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		wantError string
	}{
		{
			name:      "reject zero max keys",
			key:       PluginMaxKeys,
			value:     "0",
			wantError: PluginMaxKeys,
		},
		{
			name:      "reject negative max payload bytes",
			key:       PluginMaxPayloadBytes,
			value:     "-1",
			wantError: PluginMaxPayloadBytes,
		},
		{
			name:      "reject malformed max total payload bytes",
			key:       PluginMaxTotalPayloadBytes,
			value:     "not-a-number",
			wantError: PluginMaxTotalPayloadBytes,
		},
		{
			name:      "reject empty max nesting depth",
			key:       PluginMaxNestingDepth,
			value:     "",
			wantError: PluginMaxNestingDepth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(viper.Reset)
			viper.Reset()

			viper.Set(tt.key, tt.value)

			_, err := GetPluginLimitsConfig()
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantError)
		})
	}
}

func TestGetPluginLimitsConfigRejectsOverflow(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	viper.Set(PluginMaxPayloadBytes, "999999999999999999999999999999")

	_, err := GetPluginLimitsConfig()
	require.Error(t, err)
	assert.ErrorContains(t, err, PluginMaxPayloadBytes)
}

func TestGetPluginLimitsConfigRejectsCrossFieldInvariant(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	viper.Set(PluginMaxPayloadBytes, "1024")
	viper.Set(PluginMaxTotalPayloadBytes, "512")

	_, err := GetPluginLimitsConfig()
	require.Error(t, err)
	assert.ErrorContains(t, err, PluginMaxTotalPayloadBytes)
}

func TestGetPluginLimitsConfigConflictingSourceUsesGetterContract(t *testing.T) {
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv(PluginMaxKeys))
		viper.Reset()
	})
	viper.Reset()
	viper.AutomaticEnv()

	require.NoError(t, os.Setenv(PluginMaxKeys, "7"))
	viper.Set(PluginMaxKeys, "9")

	limits, err := GetPluginLimitsConfig()
	require.NoError(t, err)
	assert.Equal(t, 9, limits.MaxKeys)
}
