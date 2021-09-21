// Copyright 2018 The Kubeflow Authors
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

package util

import (
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestFailValidation(t *testing.T) {
	wf := unmarshalWf(emptyName)
	wf.Spec.Arguments.Parameters = []v1alpha1.Parameter{{Name: "dup", Value: v1alpha1.AnyStringPtr("value1")}}
	templateBytes, _ := yaml.Marshal(wf)
	_, err := ValidateWorkflow([]byte(templateBytes))
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "name is required")
	}
}

func TestValidateWorkflow_ParametersTooLong(t *testing.T) {
	var params []v1alpha1.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, v1alpha1.Parameter{Name: "name1", Value: v1alpha1.AnyStringPtr("value1")})
	}
	template := v1alpha1.Workflow{Spec: v1alpha1.WorkflowSpec{Arguments: v1alpha1.Arguments{
		Parameters: params}}}
	templateBytes, _ := yaml.Marshal(template)
	_, err := ValidateWorkflow(templateBytes)
	assert.Equal(t, codes.InvalidArgument, err.(*UserError).ExternalStatusCode())
}

func unmarshalWf(yamlStr string) *v1alpha1.Workflow {
	var wf v1alpha1.Workflow
	err := yaml.Unmarshal([]byte(yamlStr), &wf)
	if err != nil {
		panic(err)
	}
	return &wf
}

var template = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: dup
        value: "value1"
    container:
      image: docker/whalesay:latest`

var emptyName = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: ""
        value: "value1"
    container:
      image: docker/whalesay:latest`
