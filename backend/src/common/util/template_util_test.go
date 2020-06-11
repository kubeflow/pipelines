// Copyright 2018 Google LLC
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

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetParameters(t *testing.T) {
	template := v1alpha1.Workflow{
		TypeMeta: v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
		Spec: v1alpha1.WorkflowSpec{Arguments: v1alpha1.Arguments{
			Parameters: []v1alpha1.Parameter{{Name: "name1", Value: StringPointer("value1")}}}}}
	templateBytes, _ := yaml.Marshal(template)
	paramString, err := GetParameters(templateBytes)
	assert.Nil(t, err)
	assert.Equal(t, `[{"name":"name1","value":"value1"}]`, paramString)
}

func TestGetParameters_ParametersTooLong(t *testing.T) {
	var params []v1alpha1.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, v1alpha1.Parameter{Name: "name1", Value: StringPointer("value1")})
	}
	template := v1alpha1.Workflow{Spec: v1alpha1.WorkflowSpec{Arguments: v1alpha1.Arguments{
		Parameters: params}}}
	templateBytes, _ := yaml.Marshal(template)
	_, err := GetParameters(templateBytes)
	assert.Equal(t, codes.InvalidArgument, err.(*UserError).ExternalStatusCode())
}
