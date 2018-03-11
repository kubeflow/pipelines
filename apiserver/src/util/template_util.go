package util

import (
	"ml/apiserver/src/message/pipelinemanager"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
)

func GetParameters(template []byte) ([]pipelinemanager.Parameter, error) {
	var wf v1alpha1.Workflow
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
	var wf v1alpha1.Workflow
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		return nil, NewInvalidInputError("The template isn't a valid argo template.", err.Error())
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

	wfByte, err := yaml.Marshal(wf)
	if err != nil {
		return nil, NewInternalError(
			"Failed to apply parameter to the template.",
			"There is an issue marshalling the yaml file. Error: %s", err.Error())
	}

	return wfByte, nil
}
