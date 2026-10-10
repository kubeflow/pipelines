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

package mlflow

import (
	"encoding/json"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMlflowHandlerFactory_CreateWithRuntimeArgs_Success(t *testing.T) {
	viper.Reset()
	runtimeConfig := commonmlflow.MLflowRuntimeConfig{
		Endpoint:     "http://localhost",
		ParentRunID:  "parent-run-1",
		ExperimentID: "exp-1",
		AuthType:     "kubernetes",
		Timeout:      "10s",
	}
	runtimeConfigJSON, err := json.Marshal(runtimeConfig)
	require.NoError(t, err)

	factory := &mlflowHandlerFactory{}
	runtimeArgs := map[string]string{commonmlflow.EnvMLflowConfig: string(runtimeConfigJSON)}
	assert.True(t, factory.IsEnabledWithRuntimeArgs(runtimeArgs))

	handler, err := factory.CreateWithRuntimeArgs(runtimeArgs)

	require.NoError(t, err)
	require.NotNil(t, handler)
	assert.Equal(t, "mlflow", handler.Name())
}
