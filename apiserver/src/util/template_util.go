package util

import (
	"ml/apiserver/src/message/argo"
	"encoding/json"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"ml/apiserver/src/message/pipelinemanager"
)

// TODO(yangpa): Error handling. Implement
func GetParameter(template []byte) []pipelinemanager.Parameter {
	var wf argo.Workflow
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		// throw
	}
	return pipelinemanager.ToParameters(wf.Spec.Arguments.Parameters)
}

// TODO(yangpa): Error handling. Implement
func InjectParameter(template []byte, parameters []pipelinemanager.Parameter) []byte {
	var wf argo.Workflow
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		// throw
	}

	newParams := make([]argo.Parameter, 0)
	passedParams := make(map[string]bool)

	for _, param := range parameters {
		param := argo.Parameter{
			Name:  param.Name,
			Value: &param.Value,
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
	b, _ := json.Marshal(newParams)
	glog.Info("new param map is " + string(b))

	wf.Spec.Arguments.Parameters = newParams

	wfByte, err := yaml.Marshal(wf)
	if err != nil {
		// throw
	}

	return wfByte
}
