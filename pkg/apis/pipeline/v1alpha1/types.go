// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Pipeline is a specification for a Pipeline resource
type Pipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineSpec   `json:"spec"`
	Status PipelineStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineList is a list of Pipeline resources
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pipeline `json:"items"`
}

// PipelineSpec is the spec for a Pipeline resource
type PipelineSpec struct {

	// A human readable description of the pipeline.
	Description string `json:"description"`

	// Schedule describes when to trigger a workflow (based on a schedule) for this pipeline.
	// If no schedule is specified, a single instance of the workflow runs immediately.
	// +optional
	Schedule *Schedule `json:schedule`

	// If the pipeline is disabled, it does not start any new workflow despite its schedule.
	Enabled bool `json:enabled`

	// TODO: add concurrency policy to specify the maximum number of concurrent workflows.

	// Workflow history limit.
	// +optional
	WorkflowHistoryLimit *int32

	// Name of the workflow to substitue in the workflow template.
	// The name may include special strings that the pipeline system will
	// substitute (by for instance the current time).
	// +optional
	Name *string `json:name`

	// List of parameters to substitute in the workflow template.
	// The parameter values may include special strings that the pipeline system
	// will substitute (by for instance the current time).
	Parameters []Parameter `json:parameters`

	// Specification of the workflow to start.
	Workflow v1alpha1.Workflow `json:workflow`
}

type Parameter struct {
	// Name of the parameter.
	Name string `json:name`

	// Value of the parameter.
	Value string `json:value`
}

type Schedule struct {
	// TODO: add start date, end date, repeat interval.
	// Cron string describing when a pipeline should run.
	// +optional
	Cron *string `json:cron`
}

// PipelineStatus is the status for a Pipeline resource.
type PipelineStatus struct {
	// Time at which this pipeline started.
	StartedAt metav1.Time `json:"startedAt,omitempty"`

	// Time at which this pipeline completed.
	FinishedAt metav1.Time `json:"finishedAt,omitempty"`

	// Time at which this pipeline was last updated.
	UpdatedAt metav1.Time `json:"startedAt,omitempty"`

	// Time at which this pipeline was last enabled/disabled.
	EnabledAt metav1.Time `json:"startedAt,omitempty"`

	// Status is a simple, high-level summary of where the pipeline is in its lifecycle.
	Status string `json:"status,omitempty"`

	// A human readable message indicating why the pipeline is in the current status.
	Message string `json:"message,omitempty"`

	// The list of workflows started by this pipeline.
	Workflows []WorkflowStatus `json:"workflows,omitempty"`
}

type WorkflowStatus struct {
	// The name of the workflow.
	Name string `json:"name,omitempty"`

	// List of parameters substituted in the workflow.
	Parameters []Parameter `json:parameters`

	// The namespace of the workflow.
	Namespace string `json:"namespace,omitempty"`

	// URL representing this object.
	SelfLink string `json:"selfLink,omitempty"`

	// UID is the unique identifier in time and space for the workflow.
	UID types.UID `json:"uid,omitempty"`

	// Status is a simple, high-level summary of where the workflow is in its lifecycle.
	Status string `json:"status,omitempty"`

	// A human readable message indicating details about why the workflow is in this condition.
	Message string `json:"message,omitempty"`

	// Time at which this workflow was created.
	CreatedAt metav1.Time `json:"createdAt,omitempty"`

	// Time at which this workflow started.
	StartedAt metav1.Time `json:"startedAt,omitempty"`

	// Time at which this workflow completed
	FinishedAt metav1.Time `json:"finishedAt,omitempty"`

	// Time at which the workflow was scheduled to start.
	ScheduledAt metav1.Time `json:"ScheduledAt,omitempty"`
}