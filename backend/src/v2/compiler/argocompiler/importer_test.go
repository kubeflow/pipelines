// Copyright 2025 The Kubeflow Authors
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

package argocompiler

import (
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddImporterTemplate_PropagatesIterationIndex(t *testing.T) {
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
	}

	name := c.addImporterTemplate(false)
	require.Equal(t, "system-importer", name)

	tmpl, exists := c.templates[name]
	require.True(t, exists)
	require.NotNil(t, tmpl)
	require.NotNil(t, tmpl.Container)

	assert.Contains(t, tmpl.Container.Args, "--iteration_index")
	assert.Contains(t, tmpl.Container.Args, inputValue(paramIterationIndex))

	var foundIterationInput bool
	for _, param := range tmpl.Inputs.Parameters {
		if param.Name == paramIterationIndex {
			foundIterationInput = true
			require.NotNil(t, param.Default)
			assert.EqualValues(t, "-1", *param.Default)
		}
	}
	assert.True(t, foundIterationInput, "importer template should declare the iteration index input")
}
