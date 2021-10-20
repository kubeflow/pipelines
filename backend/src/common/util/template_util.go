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
	"fmt"
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
	V1      TemplateType = "v1"
	V2      TemplateType = "v2"
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

var ErrorInvalidPipelineSpec = fmt.Errorf("pipeline spec is invalid")

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

// Pipeline template
type Template interface {
	IsV2() bool
	// Gets v2 pipeline name.
	V2PipelineName() string
	// Overrides v2 pipeline name to distinguish shared/namespaced pipelines.
	// The name is used as ML Metadata pipeline context name.
	OverrideV2PipelineName(name, namespace string)
	// Gets parameters in JSON format.
	ParametersJSON() (string, error)
	// Get bytes content.
	Bytes() []byte
}

func NewTemplate(bytes []byte) (Template, error) {
	format := InferTemplateFormat(bytes)
	switch format {
	case V1:
		return NewArgoTemplate(bytes)
	case V2:
		return NewV2SpecTemplate(bytes)
	default:
		return nil, NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, "unknown template format")
	}
}

type ArgoTemplate struct {
	wf *Workflow
}

func NewArgoTemplate(bytes []byte) (*ArgoTemplate, error) {
	wf, err := ValidateWorkflow(bytes)
	if err != nil {
		return nil, err
	}
	return &ArgoTemplate{wf}, nil
}

func (t *ArgoTemplate) Bytes() []byte {
	if t == nil {
		return nil
	}
	return []byte(t.wf.ToStringForStore())
}

func (t *ArgoTemplate) IsV2() bool {
	if t == nil {
		return false
	}
	return t.wf.IsV2Compatible()
}

const (
	paramV2compatPipelineName = "pipeline-name"
)

func (t *ArgoTemplate) V2PipelineName() string {
	if t == nil {
		return ""
	}
	return t.wf.GetWorkflowParametersAsMap()[paramV2compatPipelineName]
}

func (t *ArgoTemplate) OverrideV2PipelineName(name, namespace string) {
	if t == nil || !t.wf.IsV2Compatible() {
		return
	}
	var pipelineRef string
	if namespace != "" {
		pipelineRef = fmt.Sprintf("namespace/%s/pipeline/%s", namespace, name)
	} else {
		pipelineRef = fmt.Sprintf("pipeline/%s", name)
	}
	overrides := make(map[string]string)
	overrides[paramV2compatPipelineName] = pipelineRef
	t.wf.OverrideParameters(overrides)
}

func (t *ArgoTemplate) ParametersJSON() (string, error) {
	if t == nil {
		return "", nil
	}
	return MarshalParameters(t.wf.Spec.Arguments.Parameters)
}

type V2SpecTemplate struct {
	spec *pipelinespec.PipelineSpec
}

func NewV2SpecTemplate(template []byte) (*V2SpecTemplate, error) {
	var spec pipelinespec.PipelineSpec
	err := protojson.Unmarshal(template, &spec)
	if err != nil {
		return nil, NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, fmt.Sprintf("invalid v2 pipeline spec: %s", err.Error()))
	}
	if spec.GetPipelineInfo().GetName() == "" {
		return nil, NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, "invalid v2 pipeline spec: name is empty")
	}
	if spec.GetRoot() == nil {
		return nil, NewInvalidInputErrorWithDetails(ErrorInvalidPipelineSpec, "invalid v2 pipeline spec: root component is empty")
	}
	return &V2SpecTemplate{spec: &spec}, nil
}

func (t *V2SpecTemplate) Bytes() []byte {
	if t == nil {
		return nil
	}
	bytes, err := protojson.Marshal(t.spec)
	if err != nil {
		// this is unexpected
		return nil
	}
	return bytes
}

func (t *V2SpecTemplate) IsV2() bool {
	return true
}

func (t *V2SpecTemplate) V2PipelineName() string {
	if t == nil {
		return ""
	}
	return t.spec.GetPipelineInfo().GetName()
}

func (t *V2SpecTemplate) OverrideV2PipelineName(name, namespace string) {
	if t == nil {
		return
	}
	var pipelineRef string
	if namespace != "" {
		pipelineRef = fmt.Sprintf("namespace/%s/pipeline/%s", namespace, name)
	} else {
		pipelineRef = fmt.Sprintf("pipeline/%s", name)
	}
	t.spec.PipelineInfo.Name = pipelineRef
}

func (t *V2SpecTemplate) ParametersJSON() (string, error) {
	// TODO(v2): implement this after pipeline spec can contain parameter defaults
	return "[]", nil
}
