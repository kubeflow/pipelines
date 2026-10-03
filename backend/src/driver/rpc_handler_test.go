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
	"os"
	"strings"
	"testing"

	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
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

func TestParseDriverRequestArgsNativeTaskIdentity(t *testing.T) {
	requestArgs := validContainerDriverArgs()
	requestArgs["parent_task_id"] = "963f53d7-85e0-4af1-99b8-45b28c9d1e32"
	req := httptest.NewRequest("POST", "/driver", strings.NewReader(driverRequestBody(t, requestArgs)))

	args, err := parseDriverRequestArgs(req)

	require.NoError(t, err)
	assert.Equal(t, "test-namespace", args.Namespace)
	assert.Equal(t, requestArgs["parent_task_id"], args.ParentTaskID)
}

func TestParseDriverRequestArgsRequiredFields(t *testing.T) {
	commonFields := []string{
		"type", "pipeline_name", "run_id", "run_name", "run_display_name",
		"namespace", "parent_task_id", "task_name", "iteration_index",
		"ml_pipeline_server_address", "ml_pipeline_server_port",
		"log_level", "publish_logs", "cache_disabled", "ml_pipeline_tls_enabled",
		"http_proxy", "https_proxy", "no_proxy",
	}
	for _, driverType := range []string{RootDag, DAG, CONTAINER} {
		t.Run(driverType, func(t *testing.T) {
			args := validDriverArgs(driverType)
			req := httptest.NewRequest("POST", "/driver", strings.NewReader(driverRequestBody(t, args)))
			_, err := parseDriverRequestArgs(req)
			require.NoError(t, err)

			required := append([]string{}, commonFields...)
			switch driverType {
			case CONTAINER:
				required = append(required, "kubernetes_config")
			case RootDag:
				required = append(required, "runtime_config")
			}
			for _, field := range required {
				t.Run("missing_"+field, func(t *testing.T) {
					args := validDriverArgs(driverType)
					delete(args, field)
					req := httptest.NewRequest("POST", "/driver", strings.NewReader(driverRequestBody(t, args)))

					_, err := parseDriverRequestArgs(req)

					require.Error(t, err)
					assert.Contains(t, err.Error(), field)
				})
			}
		})
	}
}

func TestParseDriverRequestArgsRejectsUnknownDriverType(t *testing.T) {
	args := validContainerDriverArgs()
	args["type"] = "UNKNOWN"
	req := httptest.NewRequest("POST", "/driver", strings.NewReader(driverRequestBody(t, args)))

	_, err := parseDriverRequestArgs(req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown driver type")
}

func TestParseDriverRequestArgsPreservesOptionalSecurityDefaults(t *testing.T) {
	t.Run("absent defaults remain unset", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/driver", strings.NewReader(driverRequestBody(t, validContainerDriverArgs())))
		args, err := parseDriverRequestArgs(req)
		require.NoError(t, err)
		assert.Nil(t, args.DefaultRunAsUser)
		assert.Nil(t, args.DefaultRunAsGroup)
		assert.Empty(t, args.DefaultRunAsNonRoot)
		assert.Empty(t, args.DefaultHostUsers)
	})

	t.Run("explicit zero and false survive decoding", func(t *testing.T) {
		requestArgs := validContainerDriverArgs()
		requestArgs["default_run_as_user"] = 0
		requestArgs["default_run_as_group"] = 0
		requestArgs["default_run_as_non_root"] = "false"
		requestArgs["default_host_users"] = "false"
		req := httptest.NewRequest("POST", "/driver", strings.NewReader(driverRequestBody(t, requestArgs)))
		args, err := parseDriverRequestArgs(req)
		require.NoError(t, err)
		require.NotNil(t, args.DefaultRunAsUser)
		require.NotNil(t, args.DefaultRunAsGroup)
		assert.Zero(t, *args.DefaultRunAsUser)
		assert.Zero(t, *args.DefaultRunAsGroup)
		assert.Equal(t, "false", args.DefaultRunAsNonRoot)
		assert.Equal(t, "false", args.DefaultHostUsers)
	})
}

func TestParseDriverRequestArgsGRPCBackoff(t *testing.T) {
	for _, configured := range []bool{false, true} {
		name := "optional settings omitted"
		if configured {
			name = "explicit settings preserved"
		}
		t.Run(name, func(t *testing.T) {
			requestArgs := validContainerDriverArgs()
			want := []string{"", "", "", "", ""}
			if configured {
				requestArgs["ml_pipeline_grpc_backoff_base_delay"] = "2s"
				requestArgs["ml_pipeline_grpc_backoff_multiplier"] = "1.5"
				requestArgs["ml_pipeline_grpc_backoff_jitter"] = "0"
				requestArgs["ml_pipeline_grpc_backoff_max_delay"] = "30s"
				requestArgs["ml_pipeline_grpc_min_connect_timeout"] = "10s"
				want = []string{"2s", "1.5", "0", "30s", "10s"}
			}
			req := httptest.NewRequest("POST", "/driver", strings.NewReader(driverRequestBody(t, requestArgs)))

			args, err := parseDriverRequestArgs(req)

			require.NoError(t, err)
			assert.Equal(t, want, []string{
				args.MlPipelineGRPCBackoffBaseDelay,
				args.MlPipelineGRPCBackoffMultiplier,
				args.MlPipelineGRPCBackoffJitter,
				args.MlPipelineGRPCBackoffMaxDelay,
				args.MlPipelineGRPCMinConnectTimeout,
			})
		})
	}
}

func validContainerDriverArgs() map[string]interface{} {
	return validDriverArgs(CONTAINER)
}

func validDriverArgs(driverType string) map[string]interface{} {
	args := map[string]interface{}{
		"type":                       driverType,
		"pipeline_name":              "pipeline",
		"run_id":                     "run-id",
		"run_name":                   "run-name",
		"run_display_name":           "run-display-name",
		"namespace":                  "test-namespace",
		"parent_task_id":             "parent-task-id",
		"task_name":                  "task-name",
		"iteration_index":            "-1",
		"http_proxy":                 "",
		"https_proxy":                "",
		"no_proxy":                   "",
		"ml_pipeline_server_address": "ml-pipeline",
		"ml_pipeline_server_port":    "8887",
		"log_level":                  "1",
		"publish_logs":               "true",
		"cache_disabled":             false,
		"ml_pipeline_tls_enabled":    false,
	}
	if driverType == RootDag {
		args["parent_task_id"] = ""
		args["task_name"] = ""
	}
	switch driverType {
	case CONTAINER:
		args["kubernetes_config"] = ""
	case RootDag:
		args["runtime_config"] = ""
	}
	return args
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

func TestAPIClientConfigRequestOverridesDoNotLeak(t *testing.T) {
	environment := map[string]string{
		apiclient.KFPAPIAddressEnvVar:               "environment-api",
		apiclient.KFPAPIPortEnvVar:                  "9000",
		apiclient.KFPAPIGRPCBackoffBaseDelayEnvVar:  "1s",
		apiclient.KFPAPIGRPCBackoffMultiplierEnvVar: "1.6",
		apiclient.KFPAPIGRPCBackoffJitterEnvVar:     "0.2",
		apiclient.KFPAPIGRPCBackoffMaxDelayEnvVar:   "120s",
		apiclient.KFPAPIGRPCMinConnectTimeoutEnvVar: "20s",
	}
	for key, value := range environment {
		t.Setenv(key, value)
	}

	requestConfig := apiClientConfig(driverapi.DriverPluginArgs{
		MlPipelineServerAddress:         "request-api",
		MlPipelineServerPort:            "8887",
		MlPipelineGRPCBackoffBaseDelay:  "2s",
		MlPipelineGRPCBackoffMultiplier: "1.5",
		MlPipelineGRPCBackoffJitter:     "0",
		MlPipelineGRPCBackoffMaxDelay:   "30s",
		MlPipelineGRPCMinConnectTimeout: "10s",
	})
	assert.Equal(t, &apiclient.Config{
		Endpoint:          "request-api:8887",
		BackoffBaseDelay:  "2s",
		BackoffMultiplier: "1.5",
		BackoffJitter:     "0",
		BackoffMaxDelay:   "30s",
		MinConnectTimeout: "10s",
	}, requestConfig)

	fallbackConfig := apiClientConfig(driverapi.DriverPluginArgs{})
	assert.Equal(t, &apiclient.Config{
		Endpoint:          "environment-api:9000",
		BackoffBaseDelay:  "1s",
		BackoffMultiplier: "1.6",
		BackoffJitter:     "0.2",
		BackoffMaxDelay:   "120s",
		MinConnectTimeout: "20s",
	}, fallbackConfig)
	for key, value := range environment {
		assert.Equal(t, value, os.Getenv(key), "request changed process setting %s", key)
	}
	assert.Equal(t, "0", requestConfig.BackoffJitter)
}
