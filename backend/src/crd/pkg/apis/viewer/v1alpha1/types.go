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

package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Viewer is a specification for a Viewer resource.
type Viewer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ViewerSpec `json:"spec"`
	// Status ViewerStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ViewerList is a list of Viewer resources.
type ViewerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Viewer `json:"items"`
}

type ViewerType string

const (
	TensorboardViewer ViewerType = "TensorboardViewer"
)

// TensorboardSpec is...
type TensorboardSpec struct {
	// LogDir is ...
	LogDir string
}

// ViewerSpec is the spec for a Viewer resource.
type ViewerSpec struct {
	// Name is the unique name for this viewer instance.
	Name string `json:"name"`
	// Type is...
	Type ViewerType `json:"type"`
	//TensorboardSpec is...
	TensorboardSpec TensorboardSpec `json:"tensorboardSpec,omitempty"`
	// PodSpec is the
	PodSpec v1.PodTemplateSpec `json:"podSpec"`
}
