// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriverBinaryRegistersMLflowPlugin(t *testing.T) {
	registered := plugins.RegisteredFactories()

	var found bool
	for _, factory := range registered {
		if factory.Name() == "mlflow" {
			found = true
			break
		}
	}

	assert.True(t, found, "driver binary must register MLflow so driver plugin can run task start hooks")
}

func TestParseDriverRequestArgsRuntimeArgs(t *testing.T) {
	args := validContainerDriverArgs()
	args["runtime_args"] = "{\"KFP_MLFLOW_CONFIG\":\"{\\\"endpoint\\\":\\\"http://mlflow\\\",\\\"parent_run_id\\\":\\\"parent-run\\\",\\\"experiment_id\\\":\\\"exp\\\",\\\"auth_type\\\":\\\"kubernetes\\\"}\"}"
	body := driverRequestBody(t, args)
	req := httptest.NewRequest("POST", "/driver", strings.NewReader(body))

	parsedArgs, err := parseDriverRequestArgs(req)

	require.NoError(t, err)
	require.NotNil(t, parsedArgs)
	assert.Equal(t,
		`{"endpoint":"http://mlflow","parent_run_id":"parent-run","experiment_id":"exp","auth_type":"kubernetes"}`,
		parsedArgs.RuntimeArgs[commonmlflow.EnvMLflowConfig],
	)
}

func TestParseDriverRequestArgsAllowsEmptyRequiredValues(t *testing.T) {
	req := httptest.NewRequest("POST", "/driver", strings.NewReader(driverRequestBody(t, validContainerDriverArgs())))

	args, err := parseDriverRequestArgs(req)

	require.NoError(t, err)
	assert.Equal(t, CONTAINER, args.Type)
	assert.Empty(t, args.HTTPProxy)
	assert.False(t, args.CacheDisabledFlag)
}

func TestParseDriverRequestArgsRejectsMissingRequiredValue(t *testing.T) {
	args := validContainerDriverArgs()
	delete(args, "run_id")
	req := httptest.NewRequest("POST", "/driver", strings.NewReader(driverRequestBody(t, args)))

	_, err := parseDriverRequestArgs(req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--run_id is required for CONTAINER but was not provided")
}

func validContainerDriverArgs() map[string]interface{} {
	return map[string]interface{}{
		"type":                         CONTAINER,
		"pipeline_name":                "pipeline",
		"run_id":                       "run-id",
		"run_name":                     "run-name",
		"run_display_name":             "run-display-name",
		"pipeline_job_create_time_utc": "",
		"component":                    "{}",
		"task":                         "{}",
		"task_name":                    "task-name",
		"container":                    "{}",
		"iteration_index":              "-1",
		"dag_execution_id":             "1",
		"kubernetes_config":            "",
		"cached_decision_path":         "",
		"pod_spec_patch_path":          "",
		"condition_path":               "",
		"http_proxy":                   "",
		"https_proxy":                  "",
		"no_proxy":                     "",
		"ml_pipeline_server_address":   "ml-pipeline",
		"ml_pipeline_server_port":      "8887",
		"mlmd_server_address":          "metadata-grpc-service",
		"mlmd_server_port":             "8080",
		"log_level":                    "1",
		"publish_logs":                 "true",
		"cache_disabled":               false,
		"ml_pipeline_tls_enabled":      false,
		"metadata_tls_enabled":         false,
	}
}

func driverRequestBody(t *testing.T, args map[string]interface{}) string {
	t.Helper()
	body := map[string]interface{}{
		"template": map[string]interface{}{
			"plugin": map[string]interface{}{
				"driver-plugin": map[string]interface{}{
					"args": args,
				},
			},
		},
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)
	return string(bodyBytes)
}
