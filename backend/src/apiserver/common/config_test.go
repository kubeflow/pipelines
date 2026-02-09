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
