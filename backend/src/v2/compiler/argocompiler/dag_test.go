// Copyright 2021-2024 The Kubeflow Authors
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
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDagDriverTask_WithTaskName(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	c := &workflowCompiler{
		templates: make(map[string]*wfapi.Template),
		wf: &wfapi.Workflow{
			Spec: wfapi.WorkflowSpec{
				Templates: []wfapi.Template{},
			},
		},
		spec: &pipelinespec.PipelineSpec{
			PipelineInfo: &pipelinespec.PipelineInfo{
				Name: "test-pipeline",
			},
		},
		job: &pipelinespec.PipelineJob{
			DisplayName: "test-pipeline-run",
		},
	}

	tests := []struct {
		name         string
		inputs       dagDriverInputs
		expectedTask string
		checkParam   bool
	}{
		{
			name: "With custom taskName",
			inputs: dagDriverInputs{
				component:   "component-placeholder",
				parentDagID: "parent-1",
				task:        `{"taskInfo": {"name": "task-1"}}`,
				taskName:    "simple-loop",
			},
			expectedTask: "simple-loop",
			checkParam:   true,
		},
		{
			name: "Without taskName",
			inputs: dagDriverInputs{
				component:   "component-placeholder",
				parentDagID: "parent-1",
				task:        `{"taskInfo": {"name": "test-task"}}`,
				taskName:    "",
			},
			expectedTask: "",
			checkParam:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driverTask, outputs, err := c.dagDriverTask("test-driver", tt.inputs)
			require.NoError(t, err)
			require.NotNil(t, driverTask)
			require.NotNil(t, outputs)

			if tt.checkParam {
				foundTaskName := false
				for _, param := range driverTask.Arguments.Parameters {
					if param.Name == paramTaskName {
						foundTaskName = true
						actualValue := ""
						if param.Value != nil {
							actualValue = param.Value.String()
						}

						assert.Equal(t, tt.expectedTask, actualValue, "task-name parameter should match provided taskName")
						break
					}
				}
				assert.True(t, foundTaskName, "task-name parameter should be present when taskName is provided")
			} else {
				for _, param := range driverTask.Arguments.Parameters {
					if param.Name == paramTaskName {
						t.Errorf("task-name parameter should not be present when taskName is empty")
					}
				}
			}
		})
	}
}

func TestDagDriverTask_TaskNameIncludedInArguments(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	c := &workflowCompiler{
		templates: make(map[string]*wfapi.Template),
		wf: &wfapi.Workflow{
			Spec: wfapi.WorkflowSpec{
				Templates: []wfapi.Template{},
			},
		},
		spec: &pipelinespec.PipelineSpec{
			PipelineInfo: &pipelinespec.PipelineInfo{
				Name: "test-pipeline",
			},
		},
		job: &pipelinespec.PipelineJob{
			DisplayName: "test-pipeline-run",
		},
	}

	inputs := dagDriverInputs{
		component:   "component-spec-json",
		parentDagID: "parent-dag-123",
		task:        `{"taskInfo": {"name": "for-loop-task"}}`,
		taskName:    "simple-loop",
	}

	driverTask, _, err := c.dagDriverTask("iteration-driver", inputs)
	require.NoError(t, err)
	require.NotNil(t, driverTask)

	paramMap := make(map[string]string)
	for _, param := range driverTask.Arguments.Parameters {
		if param.Value != nil {
			paramMap[param.Name] = param.Value.String()
		}
	}

	require.Contains(t, paramMap, paramTaskName, "Driver task must include task-name parameter")
	assert.Equal(t, "simple-loop", paramMap[paramTaskName], "task-name parameter should match provided taskName")
	assert.Equal(t, "component-spec-json", paramMap[paramComponent])
	assert.Equal(t, "parent-dag-123", paramMap[paramParentDagID])
	assert.Equal(t, `{"taskInfo": {"name": "for-loop-task"}}`, paramMap[paramTask])
}
