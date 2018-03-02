package util

import (
	"ml/apiserver/src/message/argo"
	"ml/apiserver/src/message/pipelinemanager"

	"github.com/ghodss/yaml"
)

func GetParameter(template []byte) ([]pipelinemanager.Parameter, error) {
	var wf argo.Workflow
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		return nil, NewInvalidInputError("Failed to parse the parameter.")
	}
	return pipelinemanager.ToParameters(wf.Spec.Arguments.Parameters), nil
}

func InjectParameter(template []byte, parameters []pipelinemanager.Parameter) ([]byte, error) {
	var wf argo.Workflow
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		return nil, NewInvalidInputError("The template isn't a valid argo template.")
	}

	newParams := make([]argo.Parameter, 0)
	passedParams := make(map[string]bool)

	for _, param := range parameters {
		param := argo.Parameter{
			Name:  param.Name,
			Value: param.Value,
		}
		newParams = append(newParams, param)
		passedParams[param.Name] = true
	}

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
