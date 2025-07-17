/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2beta1

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"google.golang.org/protobuf/encoding/protojson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

// PipelineVersionSpec defines the desired state of PipelineVersion.
type PipelineVersionSpec struct {
	Description   string `json:"description,omitempty"`
	CodeSourceURL string `json:"codeSourceURL,omitempty"`
	PipelineName  string `json:"pipelineName,omitempty"`
	DisplayName   string `json:"displayName,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	PipelineSpec IRSpec `json:"pipelineSpec"`

	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	PlatformSpec *IRSpec `json:"platformSpec,omitempty"`

	PipelineSpecURI string `json:"pipelineSpecURI,omitempty"`
}

// SimplifiedCondition is a metav1.Condition without lastTransitionTime since the database model doesn't have such
// a concept and it allows a default status in the CRD without a controller setting it.
type SimplifiedCondition struct {
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/)?(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])$`
	// +kubebuilder:validation:MaxLength=316
	Type string `json:"type" protobuf:"bytes,1,opt,name=type"`
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=True;False;Unknown
	Status metav1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status"`
	Reason string                 `json:"reason" protobuf:"bytes,5,opt,name=reason"`
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=32768
	Message string `json:"message" protobuf:"bytes,6,opt,name=message"`
}

// PipelineVersionStatus defines the observed state of PipelineVersion.
type PipelineVersionStatus struct {
	Conditions []SimplifiedCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PipelineVersion is the Schema for the pipelineversions API.
type PipelineVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineVersionSpec   `json:"spec,omitempty"`
	Status PipelineVersionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineVersionList contains a list of PipelineVersion.
type PipelineVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineVersion `json:"items"`
}

func FromPipelineVersionModel(pipeline model.Pipeline, pipelineVersion model.PipelineVersion) (*PipelineVersion, error) {
	v2Spec, err := template.NewV2SpecTemplate([]byte(pipelineVersion.PipelineSpec), false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the pipeline spec: %w", err)
	}

	pipelineSpec := IRSpec{}
	var platformSpec *IRSpec

	if v2Spec.PipelineSpec() != nil {
		pipelineSpecBytes, err := protojson.Marshal(v2Spec.PipelineSpec())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal the pipeline spec: %w", err)
		}

		var pipelineSpecValue interface{}

		err = json.Unmarshal(pipelineSpecBytes, &pipelineSpecValue)
		if err != nil {
			return nil, fmt.Errorf("failed to decode the pipeline spec: %w", err)
		}

		pipelineSpec = IRSpec{Value: pipelineSpecValue}
	}

	if v2Spec.PlatformSpec() != nil {
		pipelineSpecBytes, err := protojson.Marshal(v2Spec.PlatformSpec())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal the platform spec: %w", err)
		}

		var platformSpecValue interface{}

		err = json.Unmarshal(pipelineSpecBytes, &platformSpecValue)
		if err != nil {
			return nil, fmt.Errorf("failed to decode the platform spec: %w", err)
		}

		platformSpec = &IRSpec{Value: platformSpecValue}
	}

	return &PipelineVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineVersion.Name,
			Namespace: pipeline.Namespace,
			UID:       types.UID(pipelineVersion.UUID),
			Labels: map[string]string{
				"pipelines.kubeflow.org/pipeline-id": pipeline.UUID,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: GroupVersion.String(),
					Kind:       "Pipeline",
					UID:        types.UID(pipeline.UUID),
					Name:       pipeline.Name,
				},
			},
		},
		Spec: PipelineVersionSpec{
			DisplayName:     pipelineVersion.DisplayName,
			Description:     pipelineVersion.Description,
			PipelineSpec:    pipelineSpec,
			PlatformSpec:    platformSpec,
			PipelineName:    pipeline.Name,
			CodeSourceURL:   pipelineVersion.CodeSourceUrl,
			PipelineSpecURI: pipelineVersion.PipelineSpecURI,
		},
	}, nil
}

func (p *PipelineVersion) ToModel() (*model.PipelineVersion, error) {
	piplineSpecAndPlatformSpec, err := yaml.Marshal(p.Spec.PipelineSpec.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pipeline spec to YAML: %w", err)
	}

	if p.Spec.PlatformSpec != nil && p.Spec.PlatformSpec.Value != nil {
		platformSpecBytes, err := yaml.Marshal(p.Spec.PlatformSpec.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal platform spec to YAML: %w", err)
		}

		piplineSpecAndPlatformSpec = append(piplineSpecAndPlatformSpec, []byte("\n---\n")...)
		piplineSpecAndPlatformSpec = append(piplineSpecAndPlatformSpec, platformSpecBytes...)
	}

	pipelineVersionStatus := model.PipelineVersionCreating

	for _, condition := range p.Status.Conditions {
		if condition.Type == "PipelineVersionStatus" {
			pipelineVersionStatus = model.PipelineVersionStatus(condition.Reason)

			break
		}
	}

	var pipelineID types.UID

	for _, ref := range p.OwnerReferences {
		if ref.Kind == "Pipeline" && ref.APIVersion == GroupVersion.String() {
			pipelineID = ref.UID

			break
		}
	}

	displayName := p.Spec.DisplayName
	if displayName == "" {
		displayName = p.Name
	}

	return &model.PipelineVersion{
		UUID:            string(p.UID),
		CreatedAtInSec:  p.CreationTimestamp.Unix(),
		Name:            p.Name,
		DisplayName:     displayName,
		Parameters:      "",
		PipelineId:      string(pipelineID),
		Status:          pipelineVersionStatus,
		CodeSourceUrl:   p.Spec.CodeSourceURL,
		Description:     p.Spec.Description,
		PipelineSpec:    string(piplineSpecAndPlatformSpec),
		PipelineSpecURI: p.Spec.PipelineSpecURI,
	}, nil
}

func (p *PipelineVersion) IsOwnedByPipeline(pipelineId string) bool {
	for _, ownerRef := range p.OwnerReferences {
		if string(ownerRef.UID) == pipelineId {
			return true
		}
	}

	return false
}

// IRSpec is just a generalization of an interface{} type
// +kubebuilder:object:generate=false
// +kubebuilder:validation:Type=""
type IRSpec struct {
	Value interface{} `json:"-"`
}

func (in *IRSpec) GetValue() interface{} {
	return runtime.DeepCopyJSONValue(in.Value)
}

func (in *IRSpec) UnmarshalJSON(val []byte) error {
	if bytes.Equal(val, []byte("null")) {
		return nil
	}
	return json.Unmarshal(val, &in.Value)
}

// MarshalJSON should be implemented against a value
// per http://stackoverflow.com/questions/21390979/custom-marshaljson-never-gets-called-in-go
// credit to K8s api machinery's RawExtension for finding this.
func (in *IRSpec) MarshalJSON() ([]byte, error) {
	if in.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(in.Value)
}

func (in *IRSpec) DeepCopy() *IRSpec {
	if in == nil {
		return nil
	}

	return &IRSpec{Value: runtime.DeepCopyJSONValue(in.Value)}
}

func (in *IRSpec) DeepCopyInto(out *IRSpec) {
	*out = *in

	if in.Value != nil {
		out.Value = runtime.DeepCopyJSONValue(in.Value)
	}
}

func (p *PipelineVersion) GetField(name string) interface{} {
	switch name {
	case "pipeline_versions.UUID":
		return p.UID
	case "pipeline_versions.pipeline_version_id":
		return p.OwnerReferences[0].UID
	case "pipeline_versions.Name":
		return p.Name
	case "pipeline_versions.CreatedAtInSec":
		return p.CreationTimestamp
	case "pipeline_versions.DisplayName":
		return p.Spec.DisplayName
	case "pipeline_versions.Description":
		return p.Spec.Description
	default:
		return nil
	}
}

func init() {
	SchemeBuilder.Register(&PipelineVersion{}, &PipelineVersionList{})
}
