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

package model

import (
	"strings"
)

type (
	RuntimeState string
	StorageState string
)

const (
	RuntimeStateUnspecified    RuntimeState = "RUNTIME_STATE_UNSPECIFIED"
	RuntimeStatePending        RuntimeState = "PENDING"
	RuntimeStateRunning        RuntimeState = "RUNNING"
	RuntimeStateSucceeded      RuntimeState = "SUCCEEDED"
	RuntimeStateSkipped        RuntimeState = "SKIPPED"
	RuntimeStateFailed         RuntimeState = "FAILED"
	RuntimeStateCancelling     RuntimeState = "CANCELING"
	RuntimeStateCanceled       RuntimeState = "CANCELED"
	RuntimeStatePaused         RuntimeState = "PAUSED"
	RuntimeStatePendingV1      RuntimeState = "Pending"
	RuntimeStateRunningV1      RuntimeState = "Running"
	RuntimeStateSucceededV1    RuntimeState = "Succeeded"
	RuntimeStateSkippedV1      RuntimeState = "Skipped"
	RuntimeStateTerminatingV1  RuntimeState = "Terminating"
	RuntimeStateFailedV1       RuntimeState = "Failed"
	RuntimeStateErrorV1        RuntimeState = "Error"
	StorageStateUnspecified    StorageState = "STORAGE_STATE_UNSPECIFIED"
	StorageStateAvailable      StorageState = "AVAILABLE"
	StorageStateArchived       StorageState = "ARCHIVED"
	StorageStateUnspecifiedV1  StorageState = "STORAGESTATE_UNSPECIFIED"
	StorageStateAvailableV1    StorageState = "STORAGESTATE_AVAILABLE"
	StorageStateArchivedV1     StorageState = "STORAGESTATE_ARCHIVED"
	RunTerminatingConditionsV1 string       = "Terminating"
)

func (s RuntimeState) ToV2() RuntimeState {
	return RuntimeState(s.ToString())
}

func (s RuntimeState) ToV1() RuntimeState {
	switch s {
	case RuntimeStateUnspecified, "":
		return RuntimeStateErrorV1
	case RuntimeStatePending:
		return RuntimeStatePendingV1
	case RuntimeStateRunning:
		return RuntimeStateRunningV1
	case RuntimeStateSucceeded:
		return RuntimeStateSucceededV1
	case RuntimeStateSkipped:
		return RuntimeStateSkippedV1
	case RuntimeStateFailed:
		return RuntimeStateFailedV1
	case RuntimeStateCancelling:
		return RuntimeStateTerminatingV1
	case RuntimeStateCanceled:
		return RuntimeStateFailedV1
	case RuntimeStatePaused:
		return RuntimeStatePendingV1
	default:
		return ""
	}
}

func (s RuntimeState) ToUpper() RuntimeState {
	return RuntimeState(strings.ToUpper(string(s)))
}

func (s RuntimeState) ToString() string {
	switch s.ToUpper() {
	case RuntimeStateErrorV1, RuntimeStateUnspecified, "NO_STATUS", "":
		return string(RuntimeStateUnspecified)
	case RuntimeStatePending:
		return string(RuntimeStatePending)
	case RuntimeStateRunning, "ENABLED", "READY":
		return string(RuntimeStateRunning)
	case RuntimeStateSucceeded, "DONE":
		return string(RuntimeStateSucceeded)
	case RuntimeStateSkipped:
		return string(RuntimeStateSkipped)
	case RuntimeStateFailed:
		return string(RuntimeStateFailed)
	case RuntimeStateCancelling:
		return string(RuntimeStateCancelling)
	case RuntimeStateCanceled, "DISABLED":
		return string(RuntimeStateCanceled)
	case RuntimeStatePaused:
		return string(RuntimeStatePaused)
	default:
		return ""
	}
}

func (s RuntimeState) IsValid() bool {
	switch s {
	case RuntimeStateUnspecified, RuntimeStatePending, RuntimeStateRunning, RuntimeStateSucceeded, RuntimeStateSkipped, RuntimeStateFailed, RuntimeStateCancelling, RuntimeStateCanceled, RuntimeStatePaused, RuntimeStateErrorV1:
		return true
	default:
		return false
	}
}

func (s StorageState) ToV2() StorageState {
	return StorageState(s.ToString())
}

func (s StorageState) ToV1() StorageState {
	switch s {
	case StorageStateAvailable:
		return StorageStateAvailableV1
	case StorageStateArchived:
		return StorageStateArchivedV1
	case StorageStateUnspecified:
		return StorageStateUnspecifiedV1
	case StorageStateAvailableV1:
		return StorageStateAvailableV1
	case StorageStateArchivedV1:
		return StorageStateArchivedV1
	case StorageStateUnspecifiedV1, "":
		return StorageStateUnspecifiedV1
	default:
		return ""
	}
}

func (s StorageState) ToUpper() StorageState {
	return StorageState(strings.ToUpper(string(s)))
}

func (s StorageState) ToString() string {
	switch s.ToUpper() {
	case StorageStateUnspecified, StorageStateUnspecifiedV1, "NO_STATUS", "ERROR", "":
		return string(StorageStateUnspecified)
	case StorageStateAvailable, StorageStateAvailableV1, "ENABLED", "READY":
		return string(StorageStateAvailable)
	case StorageStateArchived, StorageStateArchivedV1, "DISABLED":
		return string(StorageStateArchived)
	default:
		return ""
	}
}

func (s StorageState) IsValid() bool {
	switch s {
	case StorageStateAvailable, StorageStateArchived, StorageStateUnspecified, StorageStateAvailableV1, StorageStateArchivedV1, StorageStateUnspecifiedV1:
		return true
	default:
		return false
	}
}

type Tabler interface {
	TableName() string
}

// TableName overrides the table name used by Run.
func (Run) TableName() string {
	return "run_details"
}

type Run struct {
	UUID        string `gorm:"column:UUID; not null; primary_key"`
	DisplayName string `gorm:"column:DisplayName; not null;"` /* The name that user provides. Can contain special characters*/
	K8SName     string `gorm:"column:Name; not null;"`        /* The name of the K8s resource. Follow regex '[a-z0-9]([-a-z0-9]*[a-z0-9])?'*/
	Description string `gorm:"column:Description; not null;"`

	Namespace      string `gorm:"column:Namespace; not null;"`
	ExperimentId   string `gorm:"column:ExperimentUUID; not null;"`
	RecurringRunId string `gorm:"column:JobUUID; default:null;"`

	StorageState   StorageState `gorm:"column:StorageState; not null;"`
	ServiceAccount string       `gorm:"column:ServiceAccount; not null;"`
	Metrics        []*RunMetric

	// ResourceReferences are deprecated. Use Namespace, ExperimentId,
	// RecurringRunId, PipelineSpec.PipelineId, PipelineSpec.PipelineVersionId
	ResourceReferences []*ResourceReference

	PipelineSpec

	RunDetails
}

func (r *Run) ToV1() *Run {
	if r.ResourceReferences == nil {
		r.ResourceReferences = make([]*ResourceReference, 0)
	}
	r.ResourceReferences = append(
		r.ResourceReferences,
		&ResourceReference{
			ResourceUUID:  r.UUID,
			ResourceType:  RunResourceType,
			ReferenceUUID: r.ExperimentId,
			ReferenceType: ExperimentResourceType,
			Relationship:  OwnerRelationship,
		},
	)
	r.ResourceReferences = append(
		r.ResourceReferences,
		&ResourceReference{
			ResourceUUID:  r.UUID,
			ResourceType:  RunResourceType,
			ReferenceUUID: r.Namespace,
			ReferenceType: NamespaceResourceType,
			Relationship:  OwnerRelationship,
		},
	)
	if r.RecurringRunId != "" {
		r.ResourceReferences = append(
			r.ResourceReferences,
			&ResourceReference{
				ResourceUUID:  r.UUID,
				ResourceType:  RunResourceType,
				ReferenceUUID: r.RecurringRunId,
				ReferenceType: JobResourceType,
				Relationship:  CreatorRelationship,
			},
		)
	}
	if r.State == "" && r.Conditions != "" {
		state := RuntimeState(r.Conditions).ToV2()
		r.Conditions = string(state.ToV1())
	} else if r.State != "" {
		r.Conditions = string(r.State.ToV1())
	}
	return r
}

func (r *Run) ToV2() *Run {
	if r.ResourceReferences != nil {
		for _, ref := range r.ResourceReferences {
			switch ref.ReferenceType {
			case ExperimentResourceType:
				r.ExperimentId = ref.ReferenceUUID
			case NamespaceResourceType:
				r.Namespace = ref.ReferenceUUID
			case JobResourceType:
				r.RecurringRunId = ref.ReferenceUUID
			}
		}
	}
	if r.Conditions != "" && r.State == "" {
		r.State = RuntimeState(r.Conditions).ToV2()
	}
	return r
}

// Stores runtime information about a pipeline run.
type RunDetails struct {
	CreatedAtInSec   int64 `gorm:"column:CreatedAtInSec; not null;"`
	ScheduledAtInSec int64 `gorm:"column:ScheduledAtInSec; default:0;"`
	FinishedAtInSec  int64 `gorm:"column:FinishedAtInSec; default:0;"`
	// Conditions were deprecated. Use State instead.
	Conditions         string       `gorm:"column:Conditions; not null;"`
	State              RuntimeState `gorm:"column:State; default:null;"`
	StateHistoryString string       `gorm:"column:StateHistory; default:'';"`
	StateHistory       []*RuntimeStatus
	// Serialized runtime details of a run in v2beta1
	PipelineRuntimeManifest string `gorm:"column:PipelineRuntimeManifest; not null; size:33554432;"`
	// Serialized Argo CRD in v1beta1
	WorkflowRuntimeManifest string `gorm:"column:WorkflowRuntimeManifest; not null; size:33554432;"`
	// Deserialized runtime details of a run includes PipelineContextId, PipelineRunContextId, and TaskDetails
	PipelineContextId    int64
	PipelineRunContextId int64
	TaskDetails          []*Task
}

type RunMetric struct {
	RunUUID     string  `gorm:"column:RunUUID; not null; primary_key;"`
	NodeID      string  `gorm:"column:NodeID; not null; primary_key;"`
	Name        string  `gorm:"column:Name; not null; primary_key;"`
	NumberValue float64 `gorm:"column:NumberValue;"`
	Format      string  `gorm:"column:Format;"`
	Payload     string  `gorm:"column:Payload; not null; size:65535;"`
}

type RuntimeStatus struct {
	UpdateTimeInSec int64        `json:"UpdateTimeInSec,omitempty"`
	State           RuntimeState `json:"State,omitempty"`
	Error           error        `json:"Error,omitempty"`
}

func (r Run) GetValueOfPrimaryKey() string {
	return r.UUID
}

func GetRunTablePrimaryKeyColumn() string {
	return "UUID"
}

// PrimaryKeyColumnName returns the primary key for model Run.
func (r *Run) PrimaryKeyColumnName() string {
	return "UUID"
}

// DefaultSortField returns the default sorting field for model Run.
func (r *Run) DefaultSortField() string {
	return "CreatedAtInSec"
}

var runAPIToModelFieldMap = map[string]string{
	"run_id":           "UUID", // added in API v2
	"id":               "UUID",
	"display_name":     "DisplayName", // added in API v2
	"name":             "DisplayName",
	"created_at":       "CreatedAtInSec",
	"finished_at":      "FinishedAtInSec",
	"description":      "Description",
	"scheduled_at":     "ScheduledAtInSec",
	"storage_state":    "StorageState",
	"status":           "Conditions",
	"namespace":        "Namespace",               // added in API v2
	"experiment_id":    "ExperimentId",            // added in API v2
	"state":            "State",                   // added in API v2
	"state_history":    "StateHistory",            // added in API v2
	"runtime_details":  "PipelineRuntimeManifest", // added in API v2
	"recurring_run_id": "RecurringRunId",          // added in API v2
}

// APIToModelFieldMap returns a map from API names to field names for model Run.
func (r *Run) APIToModelFieldMap() map[string]string {
	return runAPIToModelFieldMap
}

// GetModelName returns table name used as sort field prefix.
func (r *Run) GetModelName() string {
	// TODO(jingzhang36): return run_details here, and use model name as alias
	// and thus as prefix in sorting fields.
	return ""
}

func (r *Run) GetField(name string) (string, bool) {
	if field, ok := runAPIToModelFieldMap[name]; ok {
		return field, true
	}
	if strings.HasPrefix(name, "metric:") {
		return name[7:], true
	}
	return "", false
}

func (r *Run) GetFieldValue(name string) interface{} {
	// "name" could be a field in Run type or a name inside an array typed field
	// in Run type
	// First, try to find the value if "name" is a field in Run type
	switch name {
	case "UUID":
		return r.UUID
	case "DisplayName":
		return r.DisplayName
	case "CreatedAtInSec":
		return r.RunDetails.CreatedAtInSec
	case "Description":
		return r.Description
	case "ScheduledAtInSec":
		return r.RunDetails.ScheduledAtInSec
	case "StorageState":
		return r.StorageState
	case "Conditions":
		return r.RunDetails.Conditions
	case "Namespace":
		return r.Namespace
	case "ExperimentId":
		return r.ExperimentId
	case "State":
		return r.RunDetails.State
	case "StateHistory":
		return r.RunDetails.StateHistory
	case "PipelineRuntimeManifest":
		return r.RunDetails.PipelineRuntimeManifest
	case "RecurringRunId":
		return r.RecurringRunId
	}
	// Second, try to find the match of "name" inside an array typed field
	for _, metric := range r.Metrics {
		if metric.Name == name {
			return metric.NumberValue
		}
	}
	return nil
}

// Regular fields are the fields that are mapped to columns in Run table.
// Non-regular fields are the run metrics for now. Could have other non-regular
// sorting fields later.
func (r *Run) IsRegularField(name string) bool {
	for _, field := range runAPIToModelFieldMap {
		if field == name {
			return true
		}
	}
	return false
}

func (r *Run) GetSortByFieldPrefix(name string) string {
	if r.IsRegularField(name) {
		return r.GetModelName()
	} else {
		return ""
	}
}

func (r *Run) GetKeyFieldPrefix() string {
	return r.GetModelName()
}
