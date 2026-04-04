// Copyright 2025 The Kubeflow Authors
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

package main

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func strPtr(s string) *string {
	return &s
}

func TestSpecParsing(t *testing.T) {
	tt := []struct {
		name     string
		input    *string
		expected *kubernetesplatform.KubernetesExecutorConfig
		wantErr  bool
	}{
		{
			"Valid - test kubecfg value parse.",
			strPtr("{\"imagePullSecret\":[{\"secret_name\":\"value1\"}]}"),
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullSecret: []*kubernetesplatform.ImagePullSecret{
					{SecretName: "value1"},
				},
			},
			false,
		},
		{
			"Valid - test kubecfg value ignores unknown field.",
			strPtr("{\"imagePullSecret\":[{\"secret_name\":\"value1\"}], \"unknown_field\": \"something\"}"),
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullSecret: []*kubernetesplatform.ImagePullSecret{
					{SecretName: "value1"},
				},
			},
			false,
		},
	}

	for _, tc := range tt {
		t.Logf("Running test case: %s", tc.name)
		cfg, err := parseExecConfigJSON(tc.input)
		assert.Equal(t, tc.wantErr, err != nil)
		assert.True(t, proto.Equal(tc.expected, cfg))
	}
}

func Test_extractOutputParametersContainer(t *testing.T) {
	execution := &driver.Execution{}

	outputs := extractOutputParameters(execution, CONTAINER)

	require.NotEmpty(t, outputs)
	verifyOutputParameter(t, outputs, "condition", "nil")
}

func Test_extractOutputParametersRootDAG(t *testing.T) {
	execution := &driver.Execution{}

	outputs := extractOutputParameters(execution, RootDag)
	require.NotEmpty(t, outputs)
	verifyOutputParameter(t, outputs, "iteration-count", "0")
	verifyOutputParameter(t, outputs, "condition", "nil")
}

func Test_extractOutputParametersDAG(t *testing.T) {
	execution := &driver.Execution{}

	outputs := extractOutputParameters(execution, DAG)

	require.NotEmpty(t, outputs)
	verifyOutputParameter(t, outputs, "iteration-count", "0")
	verifyOutputParameter(t, outputs, "condition", "nil")
}

func verifyOutputParameter(t *testing.T, parameters []driverapi.Parameter, key, expectedValue string) {
	filtered := make([]driverapi.Parameter, 0, 1)
	for _, p := range parameters {
		if p.Name == key {
			filtered = append(filtered, p)
		}
	}
	require.Len(t, filtered, 1)
	require.Equal(t, expectedValue, filtered[0].Value)
}
