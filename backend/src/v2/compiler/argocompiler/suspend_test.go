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

package argocompiler

import (
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddHumanInputTemplate(t *testing.T) {
	compiler := &workflowCompiler{
		templates: map[string]*wfapi.Template{},
		wf:        &wfapi.Workflow{},
	}

	templateName, err := compiler.addHumanInputTemplate("approval", map[string]humanInputParamMeta{
		"decision": {
			Description: "Approve deployment",
			Default:     "NO",
			Enum:        []string{"YES", "NO"},
		},
	})

	require.NoError(t, err)
	require.Len(t, compiler.wf.Spec.Templates, 1)
	template := compiler.wf.Spec.Templates[0]
	assert.Equal(t, templateName, template.Name)
	require.Len(t, template.Inputs.Parameters, 1)
	assert.Equal(t, []wfapi.AnyString{*wfapi.AnyStringPtr("YES"), *wfapi.AnyStringPtr("NO")}, template.Inputs.Parameters[0].Enum)
	require.Len(t, template.Outputs.Parameters, 1)
	assert.NotNil(t, template.Outputs.Parameters[0].ValueFrom.Supplied)
}
