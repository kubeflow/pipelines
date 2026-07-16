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

package resource

import (
	"encoding/json"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateIntermediateParams(t *testing.T) {
	template := &wfapi.Template{
		Inputs: wfapi.Inputs{Parameters: []wfapi.Parameter{{
			Name: "decision",
			Enum: []wfapi.AnyString{*wfapi.AnyStringPtr("YES"), *wfapi.AnyStringPtr("NO")},
		}}},
		Outputs: wfapi.Outputs{Parameters: []wfapi.Parameter{{
			Name: "decision",
			ValueFrom: &wfapi.ValueFrom{
				Supplied: &wfapi.SuppliedValueFrom{},
			},
		}}},
	}

	assert.NoError(t, validateIntermediateParams(template, map[string]string{"decision": "YES"}))
	assert.Error(t, validateIntermediateParams(template, map[string]string{"decision": "MAYBE"}))
}

func TestBuildIntermediateInputsPatch(t *testing.T) {
	template := &wfapi.Template{
		Outputs: wfapi.Outputs{Parameters: []wfapi.Parameter{{
			Name: "decision",
			ValueFrom: &wfapi.ValueFrom{
				Supplied: &wfapi.SuppliedValueFrom{},
			},
		}}},
	}

	patch, err := buildIntermediateInputsPatch(
		"workflow-approval",
		wfapi.NodeStatus{Phase: wfapi.NodeRunning},
		template,
		map[string]string{"decision": "YES"},
	)
	require.NoError(t, err)

	var result struct {
		Status struct {
			Nodes map[string]struct {
				Phase   wfapi.NodePhase `json:"phase"`
				Message string          `json:"message"`
				Outputs wfapi.Outputs   `json:"outputs"`
			} `json:"nodes"`
		} `json:"status"`
	}
	require.NoError(t, json.Unmarshal(patch, &result))
	node := result.Status.Nodes["workflow-approval"]
	assert.Equal(t, wfapi.NodeSucceeded, node.Phase)
	assert.Equal(t, "human input provided", node.Message)
	require.Len(t, node.Outputs.Parameters, 1)
	assert.Equal(t, "YES", node.Outputs.Parameters[0].Value.String())
}
