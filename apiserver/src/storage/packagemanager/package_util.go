package packagemanager

import (
	"ml/apiserver/src/message/argo"
	"github.com/ghodss/yaml"
	"ml/apiserver/src/util"
)

// Assume the package is only single YAML file for now
func checkValidPackage(body []byte) error {
	var wf argo.Workflow
	err := yaml.Unmarshal(body, &wf)
	if err != nil {
		return util.NewInvalidInputError("The file has an invalid YAML format.")
	}
	if wf.Kind == "" || wf.Kind != argo.Kind {
		return util.NewInvalidInputError("No workflow exist in the package")
	}
	return nil
}
