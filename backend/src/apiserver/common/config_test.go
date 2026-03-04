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
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for each test
			viper.Reset()

			if tt.setEnv {
				os.Setenv(ClusterDomain, tt.envValue)
				defer os.Unsetenv(ClusterDomain)
				viper.AutomaticEnv()
			} else {
				os.Unsetenv(ClusterDomain)
			}

			result := GetClusterDomain()
			assert.Equal(t, tt.expectedDomain, result)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			if tt.setEnv {
				os.Setenv(MLPipelineServiceName, tt.envValue)
				defer os.Unsetenv(MLPipelineServiceName)
				viper.AutomaticEnv()
			} else {
				os.Unsetenv(MLPipelineServiceName)
			}

			result := GetMLPipelineServiceName()
			assert.Equal(t, tt.expectedName, result)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			if tt.setEnv {
				os.Setenv(MetadataServiceName, tt.envValue)
				defer os.Unsetenv(MetadataServiceName)
				viper.AutomaticEnv()
			} else {
				os.Unsetenv(MetadataServiceName)
			}

			result := GetMetadataServiceName()
			assert.Equal(t, tt.expectedName, result)
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
