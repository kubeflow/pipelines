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

package v1beta1

import (
	"github.com/kubeflow/pipelines/backend/src/common"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScheduledWorkflow is a specification for a ScheduledWorkflow resource
type ScheduledWorkflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScheduledWorkflowSpec   `json:"spec"`
	Status ScheduledWorkflowStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScheduledWorkflowList is a list of ScheduledWorkflow resources
type ScheduledWorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ScheduledWorkflow `json:"items"`
}

// ScheduledWorkflowSpec is the spec for a ScheduledWorkflow resource
type ScheduledWorkflowSpec struct {
	// If the schedule is disabled, it does not create any new workflow.
	Enabled bool `json:"enabled,omitempty"`

	// Max number of created workflows that can coexist.
	// If MaxConcurrency is not specified, maxConcurrency is 1.
	// MaxConcurrency cannot be smaller than 1.
	// MaxConcurrency cannot be larger than 10.
	// +optional
	MaxConcurrency *int64 `json:"maxConcurrency,omitempty"`

	// If NoCatchup is true, controller only schedules the latest period when
	// cannot catch up.
	// NoCatchup defaults to false if not specified.
	// +optional
	NoCatchup *bool `json:"noCatchup,omitempty"`

	// Max number of completed workflows to keep track of.
	// If MaxHistory is not specified, MaxHistory is 10.
	// MaxHistory cannot be smaller than 0.
	// MaxHistory cannot be larger than 100.
	// +optional
	MaxHistory *int64 `json:"maxHistory,omitempty"`

	// Trigger describes when to create a new workflow.
	Trigger `json:"trigger,omitempty"`

	// Specification of the workflow to schedule.
	// +optional
	Workflow *WorkflowResource `json:"workflow,omitempty"`

	// ExperimentId
	ExperimentId string `json:"experimentId,omitempty"`

	// PipelineId
	PipelineId string `json:"pipelineId,omitempty"`

	// PipelineVersionId
	PipelineVersionId string `json:"pipelineVersionId,omitempty"`

	// TODO(gkcalat): consider adding PipelineVersionName to avoid confusion.
	// Pipeline versions's Name will be required if ID is not empty.
	// This carries the name of the pipeline version in v2beta1.
	PipelineName string `json:"pipelineName,omitempty"`

	// ServiceAccount
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// TODO: support additional resource types: K8 jobs, etc.

}

type WorkflowResource struct {
	// List of parameters to substitute in the workflow template.
	// The parameter values may include special strings that the controller will substitute:
	// [[ScheduledTime]] is substituted by the scheduled time of the workflow (default format)
	// [[CurrentTime]] is substituted by the current time (default format)
	// [[Index]] is substituted by the index of the workflow (e.g. 3 means that it was the 3rd workflow created)
	// [[ScheduledTime.15-04-05]] is substituted by the sheduled time (custom format specified as a Go time format: https://golang.org/pkg/time/#Parse)
	// [[CurrentTime.15-04-05]] is substituted by the current time (custom format specified as a Go time format: https://golang.org/pkg/time/#Parse)

	Parameters []Parameter `json:"parameters,omitempty"`

	// Specification of the workflow to start.
	// Use interface{} for backward compatibility
	// TODO: change it to string and avoid type casting
	//       after several releases
	Spec interface{} `json:"spec,omitempty"`
}

type Parameter struct {
	// Name of the parameter.
	Name string `json:"name,omitempty"`

	// Value of the parameter.
	Value string `json:"value,omitempty"`
}

// Trigger specifies when to create a new workflow.
type Trigger struct {
	// If all the following fields are nil, the schedule create a single workflow
	// immediately.

	// Create workflows according to a cron schedule.
	CronSchedule *CronSchedule `json:"cronSchedule,omitempty"`

	// Create workflows periodically.
	PeriodicSchedule *PeriodicSchedule `json:"periodicSchedule,omitempty"`
}

type CronSchedule struct {
	// Time at which scheduling starts.
	// If no start time is specified, the StartTime is the creation time of the schedule.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Time at which scheduling ends.
	// If no end time is specified, the EndTime is the end of time.
	// +optional
	EndTime *metav1.Time `json:"endTime,omitempty"`

	// Cron string describing when a workflow should be created within the
	// time interval defined by StartTime and EndTime.
	// +optional
	Cron string `json:"cron,omitempty"`
}

type PeriodicSchedule struct {
	// Time at which scheduling starts.
	// If no start time is specified, the StartTime is the creation time of the schedule.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Time at which scheduling ends.
	// If no end time is specified, the EndTime is the end of time.
	// +optional
	EndTime *metav1.Time `json:"endTime,omitempty"`

	// Cron string describing when a workflow should be created within the
	// time interval defined by StartTime and EndTime.
	// +optional
	IntervalSecond int64 `json:"intervalSecond,omitempty"`
}

// ScheduledWorkflowStatus is the status for a ScheduledWorkflow resource.
type ScheduledWorkflowStatus struct {

	// The latest available observations of an object's current state.
	// +optional
	Conditions []ScheduledWorkflowCondition `json:"conditions,omitempty"`

	// TriggerStatus provides status info depending on the type of triggering.
	Trigger TriggerStatus `json:"trigger,omitempty"`

	// Status of workflow resources.
	WorkflowHistory *WorkflowHistory `json:"workflowHistory,omitempty"`
}

type ScheduledWorkflowConditionType string

// These are valid conditions of a ScheduledWorkflow.
const (
	ScheduledWorkflowEnabled   ScheduledWorkflowConditionType = "Enabled"
	ScheduledWorkflowDisabled  ScheduledWorkflowConditionType = "Disabled"
	ScheduledWorkflowRunning   ScheduledWorkflowConditionType = "Running"
	ScheduledWorkflowSucceeded ScheduledWorkflowConditionType = "Succeeded"
	ScheduledWorkflowError     ScheduledWorkflowConditionType = "Error"
)

type ScheduledWorkflowCondition struct {
	// Type of job condition.
	Type ScheduledWorkflowConditionType `json:"type,omitempty"`
	// Status of the condition, one of True, False, Unknown.
	Status core.ConditionStatus `json:"status,omitempty"`
	// Last time the condition was checked.
	// +optional
	LastProbeTime metav1.Time `json:"lastHeartbeatTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

type TriggerStatus struct {
	// Time of the last creation of a workflow.
	LastTriggeredTime *metav1.Time `json:"lastTriggeredTime,omitempty"`

	// Time of the next creation of a workflow (assuming that the schedule is enabled).
	NextTriggeredTime *metav1.Time `json:"nextTriggeredTime,omitempty"`

	// Index of the last workflow created.
	LastIndex *int64 `json:"lastWorkflowIndex,omitempty"`
}

type WorkflowHistory struct {
	// The list of active workflows started by this schedule.
	Active []WorkflowStatus `json:"active,omitempty"`

	// The list of completed workflows started by this schedule.
	Completed []WorkflowStatus `json:"completed,omitempty"`
}

type WorkflowStatus struct {
	// The name of the workflow.
	Name string `json:"name,omitempty"`

	// The namespace of the workflow.
	Namespace string `json:"namespace,omitempty"`

	// URL representing this object.
	SelfLink string `json:"selfLink,omitempty"`

	// UID is the unique identifier in time and space for the workflow.
	UID types.UID `json:"uid,omitempty"`

	// Phase is a high level summary of the status of the workflow.
	Phase common.ExecutionPhase `json:"phase,omitempty"`

	// A human readable message indicating details about why the workflow is in
	// this condition.
	Message string `json:"message,omitempty"`

	// Time at which this workflow was created.
	CreatedAt metav1.Time `json:"createdAt,omitempty"`

	// Time at which this workflow started.
	StartedAt metav1.Time `json:"startedAt,omitempty"`

	// Time at which this workflow completed
	FinishedAt metav1.Time `json:"finishedAt,omitempty"`

	// Time at which the workflow was triggered.
	ScheduledAt metav1.Time `json:"scheduledAt,omitempty"`

	// The index of the workflow. For instance, if this workflow is the second one
	// to execute as part of this schedule, the index is 1.
	Index int64 `json:"index,omitempty"`
}
