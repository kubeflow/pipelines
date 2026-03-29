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
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8score "k8s.io/api/core/v1"
)

func TestNewConfig_ConcurrentRequestsDoNotChangeGlobalConfig(t *testing.T) {
	previousConfig := configInstance
	t.Cleanup(func() { configInstance = previousConfig })
	InitializeConfig("http://global.example", "https://global.example", "global.internal")
	globalConfig := GetConfig()
	globalEnv := globalConfig.GetEnvVars()

	const requestCount = 16
	configs := make([]Config, requestCount)
	var ready sync.WaitGroup
	ready.Add(requestCount)
	for i := range requestCount {
		go func() {
			defer ready.Done()
			configs[i] = NewConfig(
				fmt.Sprintf("http://request-%d.example", i),
				fmt.Sprintf("https://request-%d.example", i),
				fmt.Sprintf("request-%d.internal", i),
			)
		}()
	}
	ready.Wait()

	for i, cfg := range configs {
		assert.Equal(t, fmt.Sprintf("http://request-%d.example", i), cfg.GetHttpProxy())
		assert.Equal(t, fmt.Sprintf("https://request-%d.example", i), cfg.GetHttpsProxy())
		assert.Equal(t, fmt.Sprintf("request-%d.internal,", i)+getDefaultNoProxyValue(), cfg.GetNoProxy())
	}
	assert.Same(t, globalConfig, GetConfig())
	assert.Equal(t, globalEnv, GetConfig().GetEnvVars())
}

func TestNewConfig_MergesNoProxyWithInternalAddresses(t *testing.T) {
	cfg := NewConfig("http://proxy.example", "", " example.internal,localhost,example.internal, ,127.0.0.1 ")
	entries := strings.Split(cfg.GetNoProxy(), ",")
	expectedEntries := append([]string{"example.internal"}, strings.Split(getDefaultNoProxyValue(), ",")...)
	assert.ElementsMatch(t, expectedEntries, entries)
	assert.Equal(t, "example.internal", entries[0])
}

func TestNewConfigFromEnvVars(t *testing.T) {
	tests := []struct {
		envVars        map[string]string
		expectedConfig Config
	}{
		{
			envVars:        map[string]string{},
			expectedConfig: EmptyConfig(),
		},
		{
			envVars: map[string]string{
				HTTPProxyEnv: "http_proxy",
			},
			expectedConfig: NewConfig("http_proxy", "", getDefaultNoProxyValue()),
		},
		{
			envVars: map[string]string{
				HTTPProxyEnv:  "http_proxy",
				HTTPSProxyEnv: "https_proxy",
			},
			expectedConfig: NewConfig("http_proxy", "https_proxy", getDefaultNoProxyValue()),
		},
		{
			envVars: map[string]string{
				HTTPProxyEnv: "http_proxy",
				NoProxyEnv:   "no_proxy",
			},
			expectedConfig: NewConfig("http_proxy", "", "no_proxy"),
		},
		{
			envVars: map[string]string{
				HTTPSProxyEnv: "https_proxy",
			},
			expectedConfig: NewConfig("", "https_proxy", getDefaultNoProxyValue()),
		},
		{
			envVars: map[string]string{
				HTTPSProxyEnv: "https_proxy",
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
				HTTPProxyEnv:  "http_proxy",
				HTTPSProxyEnv: "https_proxy",
				NoProxyEnv:    "no_proxy",
			},
			expectedConfig: NewConfig("http_proxy", "https_proxy", mergeNoProxyEntries("no_proxy", getDefaultNoProxyValue())),
		},
		{
			envVars: map[string]string{
				HTTPProxyEnv:  "",
				HTTPSProxyEnv: "",
				NoProxyEnv:    "",
			},
			expectedConfig: EmptyConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%+v", tt), func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				err := os.Setenv(k, v)
				require.NoError(t, err)
			}
			actualConfig := newConfigFromEnv()
			require.Equal(t, tt.expectedConfig, actualConfig)
		})
	}
}

func TestGetEnvVars(t *testing.T) {
	tests := []struct {
		config          Config
		expectedEnvVars []k8score.EnvVar
	}{
		{
			NewConfig("http", "", ""),
			[]k8score.EnvVar{
				{Name: "http_proxy", Value: "http"},
				{Name: "HTTP_PROXY", Value: "http"},
				{Name: "no_proxy", Value: getDefaultNoProxyValue()},
				{Name: "NO_PROXY", Value: getDefaultNoProxyValue()},
			},
		},
		{
			NewConfig("", "https", ""),
			[]k8score.EnvVar{
				{Name: "https_proxy", Value: "https"},
				{Name: "HTTPS_PROXY", Value: "https"},
				{Name: "no_proxy", Value: getDefaultNoProxyValue()},
				{Name: "NO_PROXY", Value: getDefaultNoProxyValue()},
			},
		},
		{
			NewConfig("", "", "no"),
			[]k8score.EnvVar{
				{Name: "no_proxy", Value: "no"},
				{Name: "NO_PROXY", Value: "no"},
			},
		},
		{
			NewConfig("http", "https", "no"),
			[]k8score.EnvVar{
				{Name: "http_proxy", Value: "http"},
				{Name: "HTTP_PROXY", Value: "http"},
				{Name: "https_proxy", Value: "https"},
				{Name: "HTTPS_PROXY", Value: "https"},
				{Name: "no_proxy", Value: mergeNoProxyEntries("no", getDefaultNoProxyValue())},
				{Name: "NO_PROXY", Value: mergeNoProxyEntries("no", getDefaultNoProxyValue())},
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%+v", tt), func(t *testing.T) {
			os.Clearenv()
			assert.Equal(t, tt.expectedEnvVars, tt.config.GetEnvVars())
		})
	}
}
