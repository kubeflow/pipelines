// Copyright 2018 The Kubeflow Authors
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
	"encoding/json"
	"strings"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
	"github.com/ghodss/yaml"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"google.golang.org/protobuf/encoding/protojson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)
type TemplateType string

const (
	V1 TemplateType = "v1"
	V2 TemplateType = "v2"
	Unknown TemplateType = "Unknown"

	argoGroup       = "argoproj.io/"
	argoVersion     = "argoproj.io/v1alpha1"
	argoK8sResource = "Workflow"
)

// Unmarshal parameters from JSON encoded string.
func UnmarshalParameters(paramsString string) ([]v1alpha1.Parameter, error) {
	if paramsString == "" {
		return nil, nil
	}
	var params []v1alpha1.Parameter
	err := json.Unmarshal([]byte(paramsString), &params)
	if err != nil {
		return nil, NewInternalServerError(err, "Parameters have wrong format")
	}
	return params, nil
}

// Marshal parameters to JSON encoded string.
// This also checks result is not longer than a limit.
func MarshalParameters(params []v1alpha1.Parameter) (string, error) {
	if params == nil {
		return "[]", nil
	}
	paramBytes, err := json.Marshal(params)
	if err != nil {
		return "", NewInvalidInputErrorWithDetails(err, "Failed to marshal the parameter.")
	}
	if len(paramBytes) > MaxParameterBytes {
		return "", NewInvalidInputError("The input parameter length exceed maximum size of %v.", MaxParameterBytes)
	}
	return string(paramBytes), nil
}

func ValidateWorkflow(template []byte) (*Workflow, error) {
	var wf v1alpha1.Workflow
	err := yaml.Unmarshal(template, &wf)
	if err != nil {
		return nil, NewInvalidInputErrorWithDetails(err, "Failed to parse the workflow template.")
	}
	if wf.APIVersion != argoVersion {
		return nil, NewInvalidInputError("Unsupported argo version. Expected: %v. Received: %v", argoVersion, wf.APIVersion)
	}
	if wf.Kind != argoK8sResource {
		return nil, NewInvalidInputError("Unexpected resource type. Expected: %v. Received: %v", argoK8sResource, wf.Kind)
	}
	_, err = validate.ValidateWorkflow(nil, nil, &wf, validate.ValidateOpts{
		Lint:                       true,
		IgnoreEntrypoint:           true,
		WorkflowTemplateValidation: false, // not used by kubeflow
	})
	if err != nil {
		return nil, err
	}
	return NewWorkflow(&wf), nil
}

// InferTemplateFormat infers format from pipeline template.
// There is no guarantee that the template is valid in inferred format, so validation
// is still needed.
func InferTemplateFormat(template []byte) TemplateType {
	switch {
	case len(template) == 0:
		return Unknown
	case isArgoWorkflow(template):
		return V1
	case isPipelineSpec(template):
		return V2
	default:
		return Unknown
	}
}

// isArgoWorkflow returns whether template is in argo workflow spec format.
func isArgoWorkflow(template []byte) bool {
	var meta metav1.TypeMeta
	err := yaml.Unmarshal(template, &meta)
	if err != nil {
		return false
	}
	return strings.HasPrefix(meta.APIVersion, argoGroup) && meta.Kind == argoK8sResource
}

// isPipelineSpec returns whether template is in KFP api/v2alpha1/PipelineSpec format.
func isPipelineSpec(template []byte) bool {
	var spec pipelinespec.PipelineSpec
	err := protojson.Unmarshal(template, &spec)
	return err == nil && spec.GetPipelineInfo().GetName() != "" && spec.GetRoot() != nil
}
