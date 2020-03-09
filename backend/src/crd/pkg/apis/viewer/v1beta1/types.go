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

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Viewer is a specification for a Viewer resource.
type Viewer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec contains specifications that pertain to how the viewer is launched and
	// managed by its controller.
	Spec ViewerSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ViewerList is a list of Viewer resources.
type ViewerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items is a list of viewers.
	Items []Viewer `json:"items"`
}

// ViewerType is the underlying type of the view. Currently, only Tensorboard is
// explicitly supported by the Viewer CRD.
type ViewerType string

const (
	// ViewerTypeTensorboard is the ViewerType constant used to indicate that the
	// underlying type is Tensorboard. An instance named `instance123` will serve
	// under /tensorboard/instance123.
	ViewerTypeTensorboard ViewerType = "tensorboard"
)

// TensorboardSpec contains fields specific to launching a tensorboard instance.
type TensorboardSpec struct {
	// LogDir is the location of the log directory to be read by tensorboard, i.e.,
	// ---log_dir.
	LogDir          string `json:"logDir"`
	TensorflowImage string `json:"tensorflowImage"`
}

// ViewerSpec is the spec for a Viewer resource.
type ViewerSpec struct {
	// Type is the type of the viewer.
	Type ViewerType `json:"type"`
	// TensorboardSpec is only checked if the Type is ViewerTypeTensorboard.
	TensorboardSpec TensorboardSpec `json:"tensorboardSpec,omitempty"`
	// PodTemplateSpec is the template spec used to launch the viewer.
	PodTemplateSpec v1.PodTemplateSpec `json:"podTemplateSpec"`
}
