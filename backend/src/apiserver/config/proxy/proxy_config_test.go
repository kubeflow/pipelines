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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewProxyConfigFromEnvVars(t *testing.T) {
	tests := []struct {
		envVars             map[string]interface{}
		expectedProxyConfig ProxyConfig
	}{
		{
			envVars:             map[string]interface{}{},
			expectedProxyConfig: NewProxyConfig("", "", ""),
		},
		{
			envVars: map[string]interface{}{
				HttpProxyEnv: "http_proxy",
			},
			expectedProxyConfig: NewProxyConfig("http_proxy", "", defaultNoProxyValue),
		},
		{
			envVars: map[string]interface{}{
				HttpProxyEnv:  "http_proxy",
				HttpsProxyEnv: "https_proxy",
			},
			expectedProxyConfig: NewProxyConfig("http_proxy", "https_proxy", defaultNoProxyValue),
		},
		{
			envVars: map[string]interface{}{
				HttpProxyEnv: "http_proxy",
				NoProxyEnv:   "no_proxy",
			},
			expectedProxyConfig: NewProxyConfig("http_proxy", "", "no_proxy"),
		},
		{
			envVars: map[string]interface{}{
				HttpsProxyEnv: "https_proxy",
			},
			expectedProxyConfig: NewProxyConfig("", "https_proxy", defaultNoProxyValue),
		},
		{
			envVars: map[string]interface{}{
				HttpsProxyEnv: "https_proxy",
				NoProxyEnv:    "no_proxy",
			},
			expectedProxyConfig: NewProxyConfig("", "https_proxy", "no_proxy"),
		},
		{
			envVars: map[string]interface{}{
				NoProxyEnv: "no_proxy",
			},
			expectedProxyConfig: NewProxyConfig("", "", "no_proxy"),
		},
		{
			envVars: map[string]interface{}{
				HttpProxyEnv:  "http_proxy",
				HttpsProxyEnv: "https_proxy",
				NoProxyEnv:    "no_proxy",
			},
			expectedProxyConfig: NewProxyConfig("http_proxy", "https_proxy", "no_proxy"),
		},
		{
			envVars: map[string]interface{}{
				HttpProxyEnv:  "",
				HttpsProxyEnv: "",
				NoProxyEnv:    "",
			},
			expectedProxyConfig: NewProxyConfig("", "", ""),
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%+v", tt), func(t *testing.T) {
			actualProxyConfig := NewProxyConfigFromSettings(tt.envVars)
			assert.Equal(t, tt.expectedProxyConfig, actualProxyConfig)
		})
	}
}
