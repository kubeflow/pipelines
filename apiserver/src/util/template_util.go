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
	"ml/apiserver/src/message/argo"
	"ml/apiserver/src/message/pipelinemanager"

	"github.com/ghodss/yaml"
)

func GetParameters(template []byte) ([]pipelinemanager.Parameter, error) {
	var wf argo.Workflow
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		return nil, NewInvalidInputError("Failed to parse the parameter.", err.Error())
	}
	return pipelinemanager.ToParameters(wf.Spec.Arguments.Parameters), nil
}

// Inject the parameter to the workflow template.
// If the value of a parameter exists in both template and the parameters to be injected,
// the latter one will take the precedence and override the template one.
func InjectParameters(template []byte, parameters []pipelinemanager.Parameter) ([]byte, error) {
	var wf argo.Workflow
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		return nil, NewInvalidInputError("The template isn't a valid argo template.", err.Error())
	}

	newParams := make([]argo.Parameter, 0)
	passedParams := make(map[string]bool)

	// Create argo.Parameter object for the parameters values passed in.
	for _, param := range parameters {
		param := argo.Parameter{
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

	wfByte, err := yaml.Marshal(wf)
	if err != nil {
		return nil, NewInternalError(
			"Failed to apply parameter to the template.",
			"There is an issue marshalling the yaml file. Error: %s", err.Error())
	}

	return wfByte, nil
}
