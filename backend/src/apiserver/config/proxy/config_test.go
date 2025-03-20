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
package proxy

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestNewConfigFromEnvVars(t *testing.T) {
	tests := []struct {
		envVars        map[string]string
		expectedConfig Config
	}{
		{
			envVars:        map[string]string{},
			expectedConfig: NewConfig("", "", ""),
		},
		{
			envVars: map[string]string{
				HttpProxyEnv: "http_proxy",
			},
			expectedConfig: NewConfig("http_proxy", "", defaultNoProxyValue),
		},
		{
			envVars: map[string]string{
				HttpProxyEnv:  "http_proxy",
				HttpsProxyEnv: "https_proxy",
			},
			expectedConfig: NewConfig("http_proxy", "https_proxy", defaultNoProxyValue),
		},
		{
			envVars: map[string]string{
				HttpProxyEnv: "http_proxy",
				NoProxyEnv:   "no_proxy",
			},
			expectedConfig: NewConfig("http_proxy", "", "no_proxy"),
		},
		{
			envVars: map[string]string{
				HttpsProxyEnv: "https_proxy",
			},
			expectedConfig: NewConfig("", "https_proxy", defaultNoProxyValue),
		},
		{
			envVars: map[string]string{
				HttpsProxyEnv: "https_proxy",
				NoProxyEnv:    "no_proxy",
			},
			expectedConfig: NewConfig("", "https_proxy", "no_proxy"),
		},
		{
			envVars: map[string]string{
				NoProxyEnv: "no_proxy",
			},
			expectedConfig: NewConfig("", "", "no_proxy"),
		},
		{
			envVars: map[string]string{
				HttpProxyEnv:  "http_proxy",
				HttpsProxyEnv: "https_proxy",
				NoProxyEnv:    "no_proxy",
			},
			expectedConfig: NewConfig("http_proxy", "https_proxy", "no_proxy"),
		},
		{
			envVars: map[string]string{
				HttpProxyEnv:  "",
				HttpsProxyEnv: "",
				NoProxyEnv:    "",
			},
			expectedConfig: NewConfig("", "", ""),
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%+v", tt), func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				err := os.Setenv(k, v)
				require.NoError(t, err)
			}
			actualConfig := NewConfigFromEnv()
			require.Equal(t, tt.expectedConfig, actualConfig)
		})
	}
}
