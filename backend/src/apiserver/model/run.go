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
	// V2 runtime states
	RuntimeStateUnspecified RuntimeState = "RUNTIME_STATE_UNSPECIFIED"
	RuntimeStatePending     RuntimeState = "PENDING"
	RuntimeStateRunning     RuntimeState = "RUNNING"
	RuntimeStateSucceeded   RuntimeState = "SUCCEEDED"
	RuntimeStateSkipped     RuntimeState = "SKIPPED"
	RuntimeStateFailed      RuntimeState = "FAILED"
	RuntimeStateCancelling  RuntimeState = "CANCELING"
	RuntimeStateCanceled    RuntimeState = "CANCELED"
	RuntimeStatePaused      RuntimeState = "PAUSED"

	// V1 runtime states
	RuntimeStatePendingV1     RuntimeState = "Pending"
	RuntimeStateRunningV1     RuntimeState = "Running"
	RuntimeStateSucceededV1   RuntimeState = "Succeeded"
	RuntimeStateSkippedV1     RuntimeState = "Skipped"
	RuntimeStateTerminatingV1 RuntimeState = "Terminating"
	RuntimeStateFailedV1      RuntimeState = "Failed"
	RuntimeStateErrorV1       RuntimeState = "Error"
	RuntimeStateUnknownV1     RuntimeState = "Unknown"

	// V2 storage states
	StorageStateUnspecified StorageState = "STORAGE_STATE_UNSPECIFIED"
	StorageStateAvailable   StorageState = "AVAILABLE"
	StorageStateArchived    StorageState = "ARCHIVED"

	// V1 storage states
	StorageStateUnspecifiedV1 StorageState = "STORAGESTATE_UNSPECIFIED"
	StorageStateAvailableV1   StorageState = "STORAGESTATE_AVAILABLE"
	StorageStateArchivedV1    StorageState = "STORAGESTATE_ARCHIVED"

	// TODO(gkcalat): verify if some of these v1 states can be safely removed
	RunTerminatingConditionsV1 string = "Terminating"
	LegacyStateNoStatus        string = "NO_STATUS"
	LegacyStateEnabled         string = "ENABLED"
	LegacyStateDisabled        string = "DISABLED"
	LegacyStateError           string = "Error"
	LegacyStateReady           string = "Ready"
	LegacyStateRunning         string = "Running"
	LegacyStateSucceeded       string = "Succeeded"
	LegacyStateDone            string = "Done"
	LegacyStateEmpty           string = ""
)

func (s RuntimeState) toUpper() RuntimeState {
	return RuntimeState(strings.ToUpper(string(s)))
}

// Converts the runtime state into a string.
// This should be called before saving data to the database.
// The returned string is one of [RUNTIME_STATE_UNSPECIFIED,
// PENDING, RUNNING, SUCCEEDED, SKIPPED, FAILED, CANCELING,
// CANCELED, PAUSED]
func (s RuntimeState) ToString() string {
	return string(s.ToV2())
}

// Checks is the runtime state contains a v2-compatible value that can be written to a store.
// This should be called before converting the data.
func (s RuntimeState) IsValid() bool {
	switch s {
	case RuntimeStateUnspecified, RuntimeStatePending, RuntimeStateRunning, RuntimeStateSucceeded, RuntimeStateSkipped, RuntimeStateFailed, RuntimeStateCancelling, RuntimeStateCanceled, RuntimeStatePaused:
		return true
	default:
		return false
	}
}

// Converts to v2beta1-compatible internal representation of runtime state.
// This should be called before converting to v2beta1 API type or writing to a store.
func (s RuntimeState) ToV2() RuntimeState {
	switch s.toUpper() {
	case RuntimeStateUnspecified, RuntimeStateUnknownV1.toUpper(), RuntimeState(LegacyStateNoStatus).toUpper(), RuntimeState(LegacyStateEmpty).toUpper():
		return RuntimeStateUnspecified
	case RuntimeStatePending:
		return RuntimeStatePending
	case RuntimeStateRunning, RuntimeState(LegacyStateEnabled).toUpper(), RuntimeState(LegacyStateReady).toUpper():
		return RuntimeStateRunning
	case RuntimeStateSucceeded, RuntimeState(LegacyStateDone).toUpper():
		return RuntimeStateSucceeded
	case RuntimeStateSkipped:
		return RuntimeStateSkipped
	case RuntimeStateFailed, RuntimeStateErrorV1.toUpper():
		return RuntimeStateFailed
	case RuntimeStateCancelling, RuntimeState(RunTerminatingConditionsV1).toUpper():
		return RuntimeStateCancelling
	case RuntimeStateCanceled, RuntimeState(LegacyStateDisabled).toUpper():
		return RuntimeStateCanceled
	case RuntimeStatePaused:
		return RuntimeStatePaused
	default:
		return RuntimeStateUnspecified
	}
}

// Converts to v1beta1-compatible internal representation of runtime state.
// This should be called before converting to v1beta1 API type.
func (s RuntimeState) ToV1() RuntimeState {
	switch s.toUpper() {
	case RuntimeStateUnspecified, RuntimeStateUnknownV1.toUpper(), RuntimeState(LegacyStateNoStatus).toUpper(), RuntimeState(LegacyStateEmpty).toUpper():
		return RuntimeStateUnknownV1
	case RuntimeStatePending, RuntimeStatePendingV1.toUpper(), RuntimeStatePaused:
		return RuntimeStatePendingV1
	case RuntimeStateRunning, RuntimeStateRunningV1.toUpper(), RuntimeState(LegacyStateEnabled).toUpper(), RuntimeState(LegacyStateReady).toUpper():
		return RuntimeStateRunningV1
	case RuntimeStateSucceeded, RuntimeStateSucceededV1.toUpper(), RuntimeState(LegacyStateDone).toUpper():
		return RuntimeStateSucceededV1
	case RuntimeStateSkipped, RuntimeStateSkippedV1.toUpper():
		return RuntimeStateSkippedV1
	case RuntimeStateFailed, RuntimeStateFailedV1.toUpper(), RuntimeStateCanceled, RuntimeStateErrorV1.toUpper(), RuntimeState(LegacyStateDisabled).toUpper():
		return RuntimeStateFailedV1
	case RuntimeStateCancelling, RuntimeStateTerminatingV1.toUpper():
		return RuntimeStateTerminatingV1
	default:
		return RuntimeStateUnknownV1
	}
}

func (s StorageState) toUpper() StorageState {
	return StorageState(strings.ToUpper(string(s)))
}

// Converts the storage state into a string.
// This should be called before saving data to the database.
// The returned string is one of [STORAGE_STATE_UNSPECIFIED,
// AVAILABLE, ARCHIVED]
func (s StorageState) ToString() string {
	return string(s.ToV2())
}

// Checks is the storage state contains a v2-compatible value that can be written to a store.
// This should be called before converting the data.
func (s StorageState) IsValid() bool {
	switch s {
	case StorageStateAvailable, StorageStateArchived, StorageStateUnspecified:
		return true
	default:
		return false
	}
}

// Converts to v2beta1-compatible internal representation of storage state.
// This should be called before converting to v2beta1 API type or writing to a store.
func (s StorageState) ToV2() StorageState {
	switch s.toUpper() {
	case StorageStateUnspecified, StorageStateUnspecifiedV1, StorageState(LegacyStateNoStatus).toUpper(), StorageState(LegacyStateError).toUpper(), StorageState(LegacyStateEmpty).toUpper():
		return StorageStateUnspecified
	case StorageStateAvailable, StorageStateAvailableV1, StorageState(LegacyStateEnabled).toUpper(), StorageState(LegacyStateReady).toUpper():
		return StorageStateAvailable
	case StorageStateArchived, StorageStateArchivedV1, StorageState(LegacyStateDisabled).toUpper():
		return StorageStateArchived
	default:
		return StorageStateUnspecified
	}
}

// Converts to v1beta1-compatible internal representation of storage state.
// This should be called before converting to v1beta1 API type.
func (s StorageState) ToV1() StorageState {
	switch s.toUpper() {
	case StorageStateAvailable, StorageStateAvailableV1:
		return StorageStateAvailableV1
	case StorageStateArchived, StorageStateArchivedV1:
		return StorageStateArchivedV1
	case StorageStateUnspecified, StorageStateUnspecifiedV1, StorageState(LegacyStateDisabled).toUpper():
		return StorageStateUnspecifiedV1
	default:
		return StorageStateUnspecifiedV1
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

// Converts to v1beta1-compatible internal representation of run.
// This should be called before converting to v1beta1 API type.
func (r *Run) ToV1() *Run {
	r.ResourceReferences = make([]*ResourceReference, 0)
	if r.ExperimentId != "" {
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
	}
	if r.Namespace != "" {
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
	}
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
		r.State = RuntimeState(r.Conditions).ToV2()
		r.Conditions = string(r.State.ToV1())
	} else if r.State != "" {
		r.Conditions = string(r.State.ToV1())
	}
	r.State = r.State.ToV1()
	r.StorageState = r.StorageState.ToV1()
	return r
}

// Converts to v2beta1-compatible internal representation of run.
// This should be called before converting to v2beta1 API type or writing to a store.
func (r *Run) ToV2() *Run {
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
	if r.Conditions != "" && r.State == "" {
		r.State = RuntimeState(r.Conditions)
	}
	r.State = r.State.ToV2()
	r.StorageState = r.StorageState.ToV2()
	return r
}

// Stores runtime information about a pipeline run.
type RunDetails struct {
	CreatedAtInSec   int64 `gorm:"column:CreatedAtInSec; not null;"`
	ScheduledAtInSec int64 `gorm:"column:ScheduledAtInSec; default:0;"`
	FinishedAtInSec  int64 `gorm:"column:FinishedAtInSec; default:0;"`
	// Conditions were deprecated. Use State instead.
	Conditions         string           `gorm:"column:Conditions; not null;"`
	State              RuntimeState     `gorm:"column:State; default:null;"`
	StateHistoryString string           `gorm:"column:StateHistory; default:null; size:65535;"`
	StateHistory       []*RuntimeStatus `gorm:"-;"`
	// Serialized runtime details of a run in v2beta1
	PipelineRuntimeManifest string `gorm:"column:PipelineRuntimeManifest; not null; size:33554432;"`
	// Serialized Argo CRD in v1beta1
	WorkflowRuntimeManifest string `gorm:"column:WorkflowRuntimeManifest; not null; size:33554432;"`
	PipelineContextId       int64  `gorm:"column:PipelineContextId; default:0;"`
	PipelineRunContextId    int64  `gorm:"column:PipelineRunContextId; default:0;"`
	TaskDetails             []*Task
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
	"run_id":              "UUID",        // v2beta1 API
	"id":                  "UUID",        // v1beta1 API
	"display_name":        "DisplayName", // v2beta1 API
	"name":                "DisplayName", // v1beta1 API
	"created_at":          "CreatedAtInSec",
	"finished_at":         "FinishedAtInSec",
	"description":         "Description",
	"scheduled_at":        "ScheduledAtInSec",
	"storage_state":       "StorageState",
	"status":              "Conditions",
	"namespace":           "Namespace",               // v2beta1 API
	"experiment_id":       "ExperimentId",            // v2beta1 API
	"state":               "State",                   // v2beta1 API
	"state_history":       "StateHistory",            // v2beta1 API
	"runtime_details":     "PipelineRuntimeManifest", // v2beta1 API
	"recurring_run_id":    "RecurringRunId",          // v2beta1 API
	"pipeline_version_id": "PipelineVersionId",       // v2beta1 API
	"pipeline_id":         "PipelineId",              // v2beta1 API
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
	case "FinishedAtInSec":
		return r.RunDetails.FinishedAtInSec
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
