// Copyright 2026 The Kubeflow Authors
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
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
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
