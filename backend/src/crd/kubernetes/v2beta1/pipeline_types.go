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
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// PipelineSpec defines the desired state of Pipeline.
type PipelineSpec struct {
	Description string `json:"description,omitempty"`
}

// +kubebuilder:object:root=true

// Pipeline is the Schema for the pipelines API.
type Pipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PipelineSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineList contains a list of Pipeline.
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}

func FromPipelineModel(pipeline model.Pipeline) Pipeline {
	return Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipeline.Name,
			Namespace: pipeline.Namespace,
			UID:       types.UID(pipeline.UUID),
		},
		Spec: PipelineSpec{
			Description: pipeline.Description,
		},
	}
}

func (p *Pipeline) ToModel() *model.Pipeline {
	pipelineStatus := model.PipelineCreating

	return &model.Pipeline{
		Name:           p.Name,
		Description:    p.Spec.Description,
		Namespace:      p.Namespace,
		UUID:           string(p.UID),
		CreatedAtInSec: p.CreationTimestamp.Unix(),
		Status:         pipelineStatus,
	}
}

func (p *Pipeline) GetField(name string) interface{} {
	switch name {
	case "pipelines.id":
		return p.UID
	case "pipelines.pipeline_id":
		return p.UID
	case "pipelines.Name":
		return p.Name
	case "pipelines.display_name":
		return p.Name
	case "pipelines.created_at":
		return p.CreationTimestamp
	case "description":
		return p.Spec.Description
	case "pipelines.namespace":
		return p.Namespace
	default:
		return ""
	}
}

func init() {
	SchemeBuilder.Register(&Pipeline{}, &PipelineList{})
}
