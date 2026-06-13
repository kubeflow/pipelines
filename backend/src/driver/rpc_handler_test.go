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
		if factory.Name() == "MLflow" {
			found = true
			break
		}
	}

	assert.True(t, found, "driver binary must register MLflow so driver plugin can run task start hooks")
}

func TestParseDriverRequestArgsRuntimeArgs(t *testing.T) {
	body := `{
		"template": {
			"plugin": {
				"driver-plugin": {
					"args": {
						"type": "CONTAINER",
						"http_proxy": "",
						"https_proxy": "",
						"no_proxy": "",
						"runtime_args": "{\"KFP_MLFLOW_CONFIG\":\"{\\\"endpoint\\\":\\\"http://mlflow\\\",\\\"parent_run_id\\\":\\\"parent-run\\\",\\\"experiment_id\\\":\\\"exp\\\",\\\"auth_type\\\":\\\"kubernetes\\\"}\"}"
					}
				}
			}
		}
	}`
	req := httptest.NewRequest("POST", "/driver", strings.NewReader(body))

	args, err := parseDriverRequestArgs(req)

	require.NoError(t, err)
	require.NotNil(t, args)
	assert.Equal(t,
		`{"endpoint":"http://mlflow","parent_run_id":"parent-run","experiment_id":"exp","auth_type":"kubernetes"}`,
		args.RuntimeArgs[commonmlflow.EnvMLflowConfig],
	)
}
