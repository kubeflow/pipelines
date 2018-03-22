// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"ml/src/message"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
)

func GetParameters(template []byte) ([]message.Parameter, error) {
	var wf v1alpha1.Workflow
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		return nil, NewInvalidInputError("Failed to parse the parameter.", err.Error())
	}
	return message.ToParameters(wf.Spec.Arguments.Parameters), nil
}

// Inject the parameter to the workflow template.
// If the value of a parameter exists in both template and the parameters to be injected,
// the latter one will take the precedence and override the template one.
func InjectParameters(template []byte, parameters []message.Parameter) (v1alpha1.Workflow, error) {
	var wf v1alpha1.Workflow
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		return wf, NewBadRequestError("The template isn't a valid argo template.", err.Error())
	}

	newParams := make([]v1alpha1.Parameter, 0)
	passedParams := make(map[string]bool)

	// Create argo.Parameter object for the parameters values passed in.
	for _, param := range parameters {
		param := v1alpha1.Parameter{
			Name:  param.Name,
			Value: param.Value,
		}
		newParams = append(newParams, param)
		passedParams[param.Name] = true
	}

	// Merge the parameters in template with the parameters passed in.
	for _, param := range wf.Spec.Arguments.Parameters {
		if _, ok := passedParams[param.Name]; ok {
			// this parameter was overridden via command line
			continue
		}
		newParams = append(newParams, param)
	}
	wf.Spec.Arguments.Parameters = newParams

	return wf, nil
}
