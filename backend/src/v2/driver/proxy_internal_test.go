// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/stretchr/testify/require"
)

func TestInitPodSpecPatchUsesRequestProxyConfig(t *testing.T) {
	proxy.InitializeConfig("http://global-proxy:8080", "", "global.internal")
	t.Cleanup(proxy.InitializeConfigWithEmptyForTests)

	tests := []struct {
		name        string
		config      proxy.Config
		httpProxy   string
		httpsProxy  string
		noProxyHost string
	}{
		{
			name:        "first request",
			config:      proxy.NewConfig("http://first-proxy:8080", "", "first.internal"),
			httpProxy:   "http://first-proxy:8080",
			noProxyHost: "first.internal",
		},
		{
			name:        "second request",
			config:      proxy.NewConfig("", "http://second-proxy:8080", "second.internal"),
			httpsProxy:  "http://second-proxy:8080",
			noProxyHost: "second.internal",
		},
		{name: "explicitly disabled", config: proxy.EmptyConfig()},
		{name: "unset"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			taskConfig := &TaskConfig{}
			podSpec, err := initPodSpecPatch(
				&pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "test-image"},
				&pipelinespec.ComponentSpec{},
				&pipelinespec.ExecutorInput{},
				"task-id", "parent-id", "pipeline", "run-id", "run-name",
				"1", "false", "false", taskConfig, "", nil, "task",
				false, "", "ml-pipeline", "8887", nil, test.config,
			)
			require.NoError(t, err)
			require.Len(t, podSpec.Containers, 1)

			env := make(map[string]string)
			for _, variable := range podSpec.Containers[0].Env {
				env[variable.Name] = variable.Value
			}
			require.Equal(t, test.httpProxy, env["http_proxy"])
			require.Equal(t, test.httpProxy, env["HTTP_PROXY"])
			require.Equal(t, test.httpsProxy, env["https_proxy"])
			require.Equal(t, test.httpsProxy, env["HTTPS_PROXY"])
			require.Equal(t, env["no_proxy"], env["NO_PROXY"])
			require.NotContains(t, env["no_proxy"], "global.internal")
			if test.noProxyHost == "" {
				require.NotContains(t, env, "no_proxy")
				require.NotContains(t, env, "NO_PROXY")
			} else {
				require.Contains(t, env["no_proxy"], test.noProxyHost)
				require.Contains(t, env["no_proxy"], "localhost")
			}
		})
	}
}
