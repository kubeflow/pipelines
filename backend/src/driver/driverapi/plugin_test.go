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

package driverapi_test

import (
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validArgs(driverType driverapi.DriverType) map[string]any {
	args := map[string]any{
		"type": string(driverType), "pipeline_name": "pipeline", "run_id": "run", "run_name": "workflow",
		"run_display_name": "run", "parent_task_id": "", "task_name": "task", "namespace": "ns",
		"iteration_index": "-1", "ml_pipeline_server_address": "ml-pipeline", "ml_pipeline_server_port": "8887",
		"log_level": "1", "publish_logs": "true", "cache_disabled": false, "ml_pipeline_tls_enabled": false,
		"http_proxy": "", "https_proxy": "", "no_proxy": "",
	}
	switch driverType {
	case driverapi.RootDAGDriver:
		args["runtime_config"] = ""
	case driverapi.ContainerDriver:
		args["kubernetes_config"] = ""
	}
	return args
}

func TestPluginValueRoundTrip(t *testing.T) {
	value, err := driverapi.EncodePluginValue(validArgs(driverapi.ContainerDriver))
	require.NoError(t, err)
	var envelope map[string]map[string]map[string]any
	require.NoError(t, json.Unmarshal(value, &envelope))
	assert.Equal(t, "CONTAINER", envelope[driverapi.PluginName]["args"]["type"])

	config, owned, err := driverapi.DecodePluginConfig(value)
	require.NoError(t, err)
	assert.True(t, owned)
	assert.Equal(t, "task", config.Args["task_name"])

	args, owned, err := driverapi.ParseDriverPluginArgs(value)
	require.NoError(t, err)
	assert.True(t, owned)
	assert.Equal(t, driverapi.ContainerDriver, args.DriverType)
	assert.Equal(t, "ns", args.Namespace)
	assert.False(t, args.CacheDisabled)
}

func TestDecodePluginConfigIgnoresOtherPlugins(t *testing.T) {
	for _, parse := range []func([]byte) (bool, error){
		func(value []byte) (bool, error) {
			_, owned, err := driverapi.DecodePluginConfig(value)
			return owned, err
		},
		func(value []byte) (bool, error) {
			_, owned, err := driverapi.ParseDriverPluginArgs(value)
			return owned, err
		},
	} {
		owned, err := parse([]byte(`{"another-plugin":{"anything":true}}`))
		require.NoError(t, err)
		assert.False(t, owned)
	}
}

func TestDecodePluginConfigErrors(t *testing.T) {
	for _, tc := range []struct{ name, value, message string }{
		{"not an object", `[]`, "not a JSON object"},
		{"null entry", `{"driver-plugin":null}`, "must be an object with an args object"},
		{"array args", `{"driver-plugin":{"args":[]}}`, "must be an object with an args object"},
		{"missing args", `{"driver-plugin":{}}`, "has no args"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, owned, err := driverapi.DecodePluginConfig([]byte(tc.value))
			require.ErrorContains(t, err, tc.message)
			assert.Equal(t, tc.name != "not an object", owned)
		})
	}
}

func TestParseDriverPluginArgsValidation(t *testing.T) {
	for _, driverType := range []driverapi.DriverType{driverapi.RootDAGDriver, driverapi.DAGDriver, driverapi.ContainerDriver} {
		for _, field := range driverType.RequiredArgFields() {
			t.Run(string(driverType)+"/missing "+field, func(t *testing.T) {
				args := validArgs(driverType)
				delete(args, field)
				value, err := driverapi.EncodePluginValue(args)
				require.NoError(t, err)
				_, _, err = driverapi.ParseDriverPluginArgs(value)
				require.Error(t, err)
				if field == "type" {
					assert.Contains(t, err.Error(), "driver type is empty")
				} else {
					assert.Contains(t, err.Error(), field)
				}
			})
		}
	}
	for _, tc := range []struct {
		name, message string
		mutate        func(map[string]any)
	}{
		{"unknown type", "unknown driver type", func(args map[string]any) { args["type"] = "UNKNOWN" }},
		{"empty namespace", `"namespace" is empty`, func(args map[string]any) { args["namespace"] = "" }},
		{"unset proxy", `"https_proxy" is "unset"`, func(args map[string]any) { args["https_proxy"] = "unset" }},
		{"invalid log level", "invalid log_level", func(args map[string]any) { args["log_level"] = "-1" }},
		{"invalid publish logs", "invalid publish_logs", func(args map[string]any) { args["publish_logs"] = "maybe" }},
		{"wrong shape", "wrong shape", func(args map[string]any) { args["cache_disabled"] = "yes" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := validArgs(driverapi.DAGDriver)
			tc.mutate(args)
			value, err := driverapi.EncodePluginValue(args)
			require.NoError(t, err)
			_, _, err = driverapi.ParseDriverPluginArgs(value)
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestDriverType(t *testing.T) {
	assert.True(t, driverapi.RootDAGDriver.IsDAG())
	assert.True(t, driverapi.DAGDriver.IsDAG())
	assert.False(t, driverapi.ContainerDriver.IsDAG())
	assert.Contains(t, driverapi.RootDAGDriver.RequiredArgFields(), "runtime_config")
	assert.Contains(t, driverapi.ContainerDriver.RequiredArgFields(), "kubernetes_config")
	assert.NotContains(t, driverapi.DAGDriver.RequiredArgFields(), "kubernetes_config")
	require.NoError(t, driverapi.DAGDriver.Validate())
	require.ErrorContains(t, driverapi.DriverType("").Validate(), "driver type is empty")
	require.ErrorContains(t, driverapi.DriverType("OTHER").Validate(), `unknown driver type "OTHER"`)
	assert.Equal(t, "DAG", driverapi.DAGDriver.String())
}

func TestDriverPluginArgsParsedFlags(t *testing.T) {
	args := driverapi.DriverPluginArgs{LogLevel: "4", PublishLogs: "false"}
	verbosity, err := args.Verbosity()
	require.NoError(t, err)
	assert.Equal(t, 4, verbosity)
	publish, err := args.PublishDriverLogs()
	require.NoError(t, err)
	assert.False(t, publish)
	_, err = driverapi.DriverPluginArgs{LogLevel: "verbose"}.Verbosity()
	require.ErrorContains(t, err, "nonnegative integer")
	_, err = driverapi.DriverPluginArgs{PublishLogs: ""}.PublishDriverLogs()
	require.ErrorContains(t, err, "true or false")
}
