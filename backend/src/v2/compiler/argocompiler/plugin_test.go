// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argocompiler

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	backendcommon "github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireDriverPluginArgs(t *testing.T, tmpl *wfapi.Template) map[string]interface{} {
	t.Helper()
	require.NotNil(t, tmpl)
	require.NotNil(t, tmpl.Plugin, "driver template should use an executor plugin")
	assert.Nil(t, tmpl.Container)
	assert.Nil(t, tmpl.SecurityContext)
	assert.Empty(t, tmpl.InitContainers)
	assert.Empty(t, tmpl.Sidecars)
	assert.Empty(t, tmpl.Volumes)
	assert.Empty(t, tmpl.Metadata.Labels)
	assert.Empty(t, tmpl.Metadata.Annotations)

	var pluginConfig map[string]map[string]map[string]interface{}
	require.NoError(t, json.Unmarshal(tmpl.Plugin.Value, &pluginConfig))
	driverPlugin, ok := pluginConfig["driver-plugin"]
	require.True(t, ok, "driver-plugin config should exist")
	args, ok := driverPlugin["args"]
	require.True(t, ok, "driver-plugin args should exist")
	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)
	decoder := json.NewDecoder(bytes.NewReader(argsJSON))
	decoder.DisallowUnknownFields()
	var requestArgs driverapi.DriverPluginArgs
	require.NoError(t, decoder.Decode(&requestArgs), "compiler args must match the driver request schema")
	for key := range args {
		assert.False(t, strings.HasSuffix(key, "_path"), "plugin outputs must not use file paths: %s", key)
	}
	for _, key := range []string{"component", "task", "container", "dag_execution_id", "mlmd_server_address", "mlmd_server_port", "metadata_tls_enabled", "pipeline_job_create_time_utc", "pipeline_job_schedule_time_epoch_seconds"} {
		assert.NotContains(t, args, key)
	}
	return args
}

func TestDriverPluginArgs_PropagatesGRPCSettings(t *testing.T) {
	settings := map[string]string{
		backendcommon.MLPipelineGRPCBackoffBaseDelay:  "2s",
		backendcommon.MLPipelineGRPCBackoffMultiplier: "1.8",
		backendcommon.MLPipelineGRPCBackoffJitter:     "0.4",
		backendcommon.MLPipelineGRPCBackoffMaxDelay:   "45s",
		backendcommon.MLPipelineGRPCMinConnectTimeout: "25s",
	}
	for _, configured := range []bool{false, true} {
		name := "defaults"
		if configured {
			name = "configured"
		}
		t.Run(name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			proxy.InitializeConfigWithEmptyForTests()
			if configured {
				for key, value := range settings {
					viper.Set(key, value)
				}
			}
			c := newCompilerWithCustomTokenAudience("")
			for _, addTemplate := range []func() (string, error){c.addContainerDriverTemplate, c.addDAGDriverTemplate} {
				templateName, err := addTemplate()
				require.NoError(t, err)
				args := requireDriverPluginArgs(t, c.templates[templateName])
				for key, value := range settings {
					arg := strings.ToLower(key)
					if configured {
						assert.Equal(t, value, args[arg])
					} else {
						assert.NotContains(t, args, arg)
					}
				}
			}
		})
	}
}

func TestDriverPluginArgs_UsesRunTokenAudience(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	for _, test := range []struct {
		name     string
		audience string
		want     string
	}{
		{name: "default", want: "pipelines.kubeflow.org/runs/{{workflow.uid}}"},
		{name: "custom", audience: "custom.kfp.example", want: "custom.kfp.example/runs/{{workflow.uid}}"},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := newCompilerWithCustomTokenAudience(test.audience)
			for _, addTemplate := range []func() (string, error){c.addContainerDriverTemplate, c.addDAGDriverTemplate} {
				templateName, err := addTemplate()
				require.NoError(t, err)
				args := requireDriverPluginArgs(t, c.templates[templateName])
				assert.Equal(t, test.want, args["kfp_token_audience"])
			}
		})
	}
}

func TestContainerDriverPluginArgs_PreservesOptionalSecurityDefaults(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	for _, configured := range []bool{false, true} {
		c := newCompilerWithCustomTokenAudience("")
		if configured {
			user, group := int64(1000), int64(100)
			nonRoot, hostUsers := false, false
			c.defaultRunAsUser = &user
			c.defaultRunAsGroup = &group
			c.defaultRunAsNonRoot = &nonRoot
			c.defaultHostUsers = &hostUsers
		}
		name, err := c.addContainerDriverTemplate()
		require.NoError(t, err)
		args := requireDriverPluginArgs(t, c.templates[name])
		if configured {
			assert.Equal(t, float64(1000), args["default_run_as_user"])
			assert.Equal(t, float64(100), args["default_run_as_group"])
			assert.Equal(t, "false", args["default_run_as_non_root"])
			assert.Equal(t, "false", args["default_host_users"])
		} else {
			for _, key := range []string{"default_run_as_user", "default_run_as_group", "default_run_as_non_root", "default_host_users"} {
				assert.NotContains(t, args, key)
			}
		}
	}
}
