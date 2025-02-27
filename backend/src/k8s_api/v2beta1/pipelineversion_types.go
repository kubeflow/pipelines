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
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// PipelineVersionSpec defines the desired state of PipelineVersion.
type PipelineVersionSpec struct {
	Description   string `json:"description,omitempty"`
	CodeSourceURL string `json:"codeSourceURL,omitempty"`
	PipelineName  string `json:"pipelineName,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// Todo
	PipelineSpec PipelineIRSpec `json:"pipelineSpec"`
}

func ToPipelineSpec(p *PipelineIRSpec) (*pipelinespec.PipelineSpec, error) {
	pDefBytes, err := json.Marshal(p.Value)
	if err != nil {
		return nil, err
	}

	rv := &pipelinespec.PipelineSpec{}
	err = json.Unmarshal(pDefBytes, rv)
	if err != nil {
		return nil, err
	}

	return rv, nil
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
	pipelineSpec := PipelineIRSpec{}

	err := yaml.Unmarshal([]byte(pipelineVersion.PipelineSpec), &pipelineSpec.Value)
	if err != nil {
		return nil, fmt.Errorf("the pipeline spec is invalid YAML: %w", err)
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
			Description:   pipelineVersion.Description,
			PipelineSpec:  pipelineSpec,
			PipelineName:  pipeline.Name,
			CodeSourceURL: pipelineVersion.CodeSourceUrl,
		},
	}, nil
}

func (p *PipelineVersion) ToModel() (*model.PipelineVersion, error) {
	pipelineSpec, err := json.Marshal(p.Spec.PipelineSpec)
	if err != nil {
		return nil, fmt.Errorf("the pipeline spec is invalid JSON: %w", err)
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

	return &model.PipelineVersion{
		UUID:           string(p.UID),
		CreatedAtInSec: p.CreationTimestamp.Unix(),
		Name:           p.Name,
		PipelineId:     string(pipelineID),
		Description:    p.Spec.Description,
		PipelineSpec:   string(pipelineSpec),
		Status:         pipelineVersionStatus,
		CodeSourceUrl:  p.Spec.CodeSourceURL,
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

// PipelineIRSpec is just a generalization of an interface{} type
// +kubebuilder:object:generate=false
// +kubebuilder:validation:Type=""
type PipelineIRSpec struct {
	Value interface{} `json:"-"`
}

func (in *PipelineIRSpec) GetValue() interface{} {
	return runtime.DeepCopyJSONValue(in.Value)
}

func (in *PipelineIRSpec) UnmarshalJSON(val []byte) error {
	if bytes.Equal(val, []byte("null")) {
		return nil
	}
	return json.Unmarshal(val, &in.Value)
}

// MarshalJSON should be implemented against a value
// per http://stackoverflow.com/questions/21390979/custom-marshaljson-never-gets-called-in-go
// credit to K8s api machinery's RawExtension for finding this.
func (in *PipelineIRSpec) MarshalJSON() ([]byte, error) {
	if in.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(in.Value)
}

func (in *PipelineIRSpec) DeepCopy() *PipelineIRSpec {
	if in == nil {
		return nil
	}

	return &PipelineIRSpec{Value: runtime.DeepCopyJSONValue(in.Value)}
}

func (in *PipelineIRSpec) DeepCopyInto(out *PipelineIRSpec) {
	*out = *in

	if in.Value != nil {
		out.Value = runtime.DeepCopyJSONValue(in.Value)
	}
}

func init() {
	SchemeBuilder.Register(&PipelineVersion{}, &PipelineVersionList{})
}
