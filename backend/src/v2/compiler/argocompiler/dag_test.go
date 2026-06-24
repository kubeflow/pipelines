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
	backendcommon "github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/spf13/viper"
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
				parentTaskID: "parent-1",
				taskName:     "simple-loop",
			},
			expectedTask: "simple-loop",
			checkParam:   true,
		},
		{
			name: "Without taskName",
			inputs: dagDriverInputs{
				parentTaskID: "parent-1",
				taskName:     "",
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
		parentTaskID: "parent-dag-123",
		taskName:     "simple-loop",
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
	assert.Equal(t, "parent-dag-123", paramMap[paramParentDagTaskID])
	assert.NotContains(t, paramMap, paramTask)
}

func TestAddDAGDriverTemplate_IncludesDebugMetadata(t *testing.T) {
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

	name := c.addDAGDriverTemplate()
	require.Equal(t, "system-dag-driver", name)

	tmpl, exists := c.templates[name]
	require.True(t, exists, "system-dag-driver template should exist")
	assert.Equal(t, "dag-driver", tmpl.Metadata.Labels[systemPodRoleLabelKey])
	assert.Equal(t, "system-dag-driver", tmpl.Metadata.Annotations[systemTemplateNameAnnotationKey])
	require.NotNil(t, tmpl.SecurityContext, "system-dag-driver template should preserve pod security context hardening")
	require.NotNil(t, tmpl.Container)
	require.NotNil(t, tmpl.Container.SecurityContext, "system-dag-driver container should preserve container security context hardening")
}

func TestAddDAGDriverTemplate_PropagatesGRPCBackoffEnv(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set(backendcommon.MLPipelineGRPCBackoffBaseDelay, "2s")

	c := &workflowCompiler{
		templates: make(map[string]*wfapi.Template),
		wf: &wfapi.Workflow{
			Spec: wfapi.WorkflowSpec{
				Templates: []wfapi.Template{},
			},
		},
		spec: &pipelinespec.PipelineSpec{
			PipelineInfo: &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		},
		job: &pipelinespec.PipelineJob{DisplayName: "test-pipeline-run"},
	}

	name := c.addDAGDriverTemplate()
	tmpl := c.templates[name]
	require.NotNil(t, tmpl)
	require.NotNil(t, tmpl.Container)

	found := false
	for _, env := range tmpl.Container.Env {
		if env.Name == apiclient.KFPAPIGRPCBackoffBaseDelayEnvVar && env.Value == "2s" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected DAG driver to include configured gRPC backoff env")
}

func TestPropagateIterationIndexToNestedDAGTemplates_DefaultsTemplateInput(t *testing.T) {
	c := &workflowCompiler{
		templates: map[string]*wfapi.Template{
			"nested-dag": {
				Name: "nested-dag",
				DAG:  &wfapi.DAGTemplate{},
			},
		},
		wf: &wfapi.Workflow{
			Spec: wfapi.WorkflowSpec{
				Templates: []wfapi.Template{},
			},
		},
	}

	require.NoError(t, c.propagateIterationIndexToNestedDAGTemplates("nested-dag", map[string]bool{}))
	template := c.templates["nested-dag"]
	require.Len(t, template.Inputs.Parameters, 1)
	require.Equal(t, paramIterationIndex, template.Inputs.Parameters[0].Name)
	require.NotNil(t, template.Inputs.Parameters[0].Default)
	assert.Equal(t, "-1", template.Inputs.Parameters[0].Default.String())
}

func TestPropagateIterationIndexToNestedDAGTemplates(t *testing.T) {
	c := &workflowCompiler{
		templates: make(map[string]*wfapi.Template),
		wf: &wfapi.Workflow{
			Spec: wfapi.WorkflowSpec{
				Templates: []wfapi.Template{},
			},
		},
		spec: &pipelinespec.PipelineSpec{
			PipelineInfo: &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		},
		job: &pipelinespec.PipelineJob{DisplayName: "test-pipeline-run"},
	}

	driverTemplate := c.addDAGDriverTemplate()
	containerDriverTemplate := c.addContainerDriverTemplate()

	inner := &wfapi.Template{
		Name: "comp-inner",
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{{Name: paramParentDagTaskID}},
		},
		DAG: &wfapi.DAGTemplate{
			Tasks: []wfapi.DAGTask{
				{
					Name:     "inner-driver",
					Template: driverTemplate,
					Arguments: wfapi.Arguments{Parameters: []wfapi.Parameter{
						{Name: paramParentDagTaskID, Value: wfapi.AnyStringPtr(inputParameter(paramParentDagTaskID))},
					}},
				},
				{
					Name:     "inner-container-driver",
					Template: containerDriverTemplate,
					Arguments: wfapi.Arguments{Parameters: []wfapi.Parameter{
						{Name: paramParentDagTaskID, Value: wfapi.AnyStringPtr(inputParameter(paramParentDagTaskID))},
					}},
				},
			},
		},
	}
	outer := &wfapi.Template{
		Name: "comp-outer",
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{{Name: paramParentDagTaskID}},
		},
		DAG: &wfapi.DAGTemplate{
			Tasks: []wfapi.DAGTask{
				{
					Name:     "nested",
					Template: "comp-inner",
					Arguments: wfapi.Arguments{Parameters: []wfapi.Parameter{
						{Name: paramParentDagTaskID, Value: wfapi.AnyStringPtr(inputParameter(paramParentDagTaskID))},
					}},
				},
			},
		},
	}
	c.templates[inner.Name] = inner
	c.templates[outer.Name] = outer
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *inner, *outer)

	require.NoError(t, c.propagateIterationIndexToNestedDAGTemplates("comp-outer", map[string]bool{}))

	require.Contains(t, parameterNames(c.templates["comp-outer"].Inputs.Parameters), paramIterationIndex)
	require.Contains(t, parameterNames(c.templates["comp-inner"].Inputs.Parameters), paramIterationIndex)
	require.Contains(t, parameterNames(c.templates["comp-outer"].DAG.Tasks[0].Arguments.Parameters), paramIterationIndex)
	require.Contains(t, parameterNames(c.templates["comp-inner"].DAG.Tasks[0].Arguments.Parameters), paramIterationIndex)
	require.Contains(t, parameterNames(c.templates["comp-inner"].DAG.Tasks[1].Arguments.Parameters), paramIterationIndex)
}

func parameterNames(parameters []wfapi.Parameter) []string {
	names := make([]string, 0, len(parameters))
	for _, parameter := range parameters {
		names = append(names, parameter.Name)
	}
	return names
}
