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
	"fmt"
	"strings"
)

type StatusState string

const (
	// V2 statuses
	StatusStateUnspecified StatusState = "STATUS_UNSPECIFIED"
	StatusStateEnabled     StatusState = "ENABLED"
	StatusStateDisabled    StatusState = "DISABLED"

	// V1 statuses
	StatusStateUnspecifiedV1 StatusState = "UNKNOWN_MODE"
	StatusStateEnabledV1     StatusState = "ENABLED"
	StatusStateDisabledV1    StatusState = "DISABLED"
)

// Checks is the status contains a valid value.
// This should be called before converting the data.
func (s StatusState) IsValid() bool {
	switch s {
	case StatusStateUnspecified, StatusStateEnabled, StatusStateDisabled:
		return true
	default:
		return false
	}
}

func (s StatusState) toUpper() StatusState {
	return StatusState(strings.ToUpper(string(s)))
}

// Converts the status into a string.
// This should be called before saving data to the database.
// The returned string is one of [STATUS_UNSPECIFIED, ENABLED, DISABLED]
func (s StatusState) ToString() string {
	return string(s.ToV2())
}

// Converts to v1beta1-compatible internal representation of job status.
// This should be called before converting to v1beta1 API type.
func (s StatusState) ToV1() StatusState {
	switch s.toUpper() {
	case StatusStateUnspecified, StatusStateUnspecifiedV1, StatusState(LegacyStateNoStatus).toUpper(), StatusState(LegacyStateEmpty).toUpper():
		return StatusStateUnspecifiedV1
	case StatusStateEnabled, StatusState(LegacyStateReady).toUpper(), StatusState(LegacyStateDone).toUpper(), StatusState(LegacyStateRunning).toUpper(), StatusState(LegacyStateSucceeded).toUpper():
		return StatusStateEnabledV1
	case StatusStateDisabled:
		return StatusStateDisabledV1
	default:
		return StatusStateUnspecifiedV1
	}
}

// Converts to v2beta1-compatible internal representation of job status.
// This should be called before converting to v2beta1 API type or writing to a store.
func (s StatusState) ToV2() StatusState {
	switch s.toUpper() {
	case StatusStateUnspecified, StatusStateUnspecifiedV1, StatusState(LegacyStateNoStatus).toUpper(), StatusState(LegacyStateEmpty).toUpper():
		return StatusStateUnspecified
	case StatusStateEnabled, StatusState(LegacyStateReady).toUpper(), StatusState(LegacyStateDone).toUpper(), StatusState(LegacyStateRunning).toUpper(), StatusState(LegacyStateSucceeded).toUpper():
		return StatusStateEnabled
	case StatusStateDisabled:
		return StatusStateDisabled
	default:
		return StatusStateUnspecified
	}
}

type Job struct {
	UUID           string `gorm:"column:UUID; not null; primary_key;"`
	DisplayName    string `gorm:"column:DisplayName; not null;"` /* The name that user provides. Can contain special characters*/
	K8SName        string `gorm:"column:Name; not null;"`        /* The name of the K8s resource. Follow regex '[a-z0-9]([-a-z0-9]*[a-z0-9])?'*/
	Namespace      string `gorm:"column:Namespace; not null;"`
	ServiceAccount string `gorm:"column:ServiceAccount; not null;"`
	Description    string `gorm:"column:Description; not null;"`
	MaxConcurrency int64  `gorm:"column:MaxConcurrency; not null;"`
	NoCatchup      bool   `gorm:"column:NoCatchup; not null;"`
	CreatedAtInSec int64  `gorm:"column:CreatedAtInSec; not null;"` /* The time this record is stored in DB*/
	UpdatedAtInSec int64  `gorm:"column:UpdatedAtInSec; default:0;"`
	Enabled        bool   `gorm:"column:Enabled; not null;"`
	ExperimentId   string `gorm:"column:ExperimentUUID; not null;"`
	// ResourceReferences are deprecated. Use Namespace, ExperimentId
	// PipelineSpec.PipelineId, PipelineSpec.PipelineVersionId
	ResourceReferences []*ResourceReference
	Trigger
	PipelineSpec
	Conditions string `gorm:"column:Conditions; not null;"`
}

// Converts to v1beta1-compatible internal representation of job.
// This should be called before converting to v1beta1 API type.
func (j *Job) ToV1() *Job {
	j.ResourceReferences = make([]*ResourceReference, 0)
	if j.Namespace != "" {
		j.ResourceReferences = append(
			j.ResourceReferences,
			&ResourceReference{
				ResourceUUID:  j.UUID,
				ResourceType:  JobResourceType,
				ReferenceUUID: j.Namespace,
				ReferenceType: NamespaceResourceType,
				Relationship:  OwnerRelationship,
			},
		)
	}
	if j.ExperimentId != "" {
		j.ResourceReferences = append(
			j.ResourceReferences,
			&ResourceReference{
				ResourceUUID:  j.UUID,
				ResourceType:  JobResourceType,
				ReferenceUUID: j.ExperimentId,
				ReferenceType: ExperimentResourceType,
				Relationship:  OwnerRelationship,
			},
		)
	}
	if j.PipelineSpec.PipelineId != "" {
		j.ResourceReferences = append(
			j.ResourceReferences,
			&ResourceReference{
				ResourceUUID:  j.UUID,
				ResourceType:  JobResourceType,
				ReferenceUUID: j.PipelineSpec.PipelineId,
				ReferenceType: PipelineResourceType,
				Relationship:  CreatorRelationship,
			},
		)
	}
	if j.PipelineSpec.PipelineVersionId != "" {
		j.ResourceReferences = append(
			j.ResourceReferences,
			&ResourceReference{
				ResourceUUID:  j.UUID,
				ResourceType:  JobResourceType,
				ReferenceUUID: j.PipelineSpec.PipelineVersionId,
				ReferenceType: PipelineVersionResourceType,
				Relationship:  CreatorRelationship,
			},
		)
	}
	j.Conditions = string(StatusState(j.Conditions).ToV1())
	return j
}

// Converts to v2beta1-compatible internal representation of job.
// This should be called before converting to v2beta1 API type.
func (j *Job) ToV2() *Job {
	for _, ref := range j.ResourceReferences {
		switch ref.ReferenceType {
		case NamespaceResourceType:
			j.Namespace = ref.ReferenceUUID
		case ExperimentResourceType:
			j.ExperimentId = ref.ReferenceUUID
		case PipelineResourceType:
			j.PipelineSpec.PipelineId = ref.ReferenceUUID
		case PipelineVersionResourceType:
			j.PipelineSpec.PipelineVersionId = ref.ReferenceUUID
		}
	}
	j.Conditions = StatusState(j.Conditions).ToString()
	return j
}

// Trigger specifies when to create a new workflow.
type Trigger struct {
	// Create workflows according to a cron schedule.
	CronSchedule
	// Create workflows periodically.
	PeriodicSchedule
}

type CronSchedule struct {
	// Time at which scheduling starts.
	// If no start time is specified, the StartTime is the creation time of the schedule.
	CronScheduleStartTimeInSec *int64 `gorm:"column:CronScheduleStartTimeInSec;"`

	// Time at which scheduling ends.
	// If no end time is specified, the EndTime is the end of time.
	CronScheduleEndTimeInSec *int64 `gorm:"column:CronScheduleEndTimeInSec;"`

	// Cron string describing when a workflow should be created within the
	// time interval defined by StartTime and EndTime.
	Cron *string `gorm:"column:Schedule;"`
}

type PeriodicSchedule struct {
	// Time at which scheduling starts.
	// If no start time is specified, the StartTime is the creation time of the schedule.
	PeriodicScheduleStartTimeInSec *int64 `gorm:"column:PeriodicScheduleStartTimeInSec;"`

	// Time at which scheduling ends.
	// If no end time is specified, the EndTime is the end of time.
	PeriodicScheduleEndTimeInSec *int64 `gorm:"column:PeriodicScheduleEndTimeInSec;"`

	// Interval describing when a workflow should be created within the
	// time interval defined by StartTime and EndTime.
	IntervalSecond *int64 `gorm:"column:IntervalSecond;"`
}

func (j Job) GetValueOfPrimaryKey() string {
	return fmt.Sprint(j.UUID)
}

func GetJobTablePrimaryKeyColumn() string {
	return "UUID"
}

// PrimaryKeyColumnName returns the primary key for model Job.
func (j *Job) PrimaryKeyColumnName() string {
	return "UUID"
}

// DefaultSortField returns the default sorting field for model Job.
func (j *Job) DefaultSortField() string {
	return "CreatedAtInSec"
}

var jobAPIToModelFieldMap = map[string]string{
	"id":                  "UUID",        // v1beta1 API
	"recurring_run_id":    "UUID",        // v2beta1 API
	"name":                "DisplayName", // v1beta1 API
	"display_name":        "DisplayName", // v2beta1 API
	"created_at":          "CreatedAtInSec",
	"updated_at":          "UpdatedAtInSec",
	"description":         "Description",
	"pipeline_name":       "PipelineName",      // v2beta1 API
	"pipeline_version_id": "PipelineVersionId", // v2beta1 API
	"pipeline_id":         "PipelineId",        // v2beta1 API
}

// APIToModelFieldMap returns a map from API names to field names for model Job.
func (k *Job) APIToModelFieldMap() map[string]string {
	return jobAPIToModelFieldMap
}

// GetModelName returns table name used as sort field prefix.
func (j *Job) GetModelName() string {
	return "jobs"
}

func (j *Job) GetField(name string) (string, bool) {
	if field, ok := jobAPIToModelFieldMap[name]; ok {
		return field, true
	}
	return "", false
}

func (j *Job) GetFieldValue(name string) interface{} {
	switch name {
	case "UUID":
		return j.UUID
	case "DisplayName":
		return j.DisplayName
	case "CreatedAtInSec":
		return j.CreatedAtInSec
	case "UpdatedAtInSec":
		return j.UpdatedAtInSec
	case "PipelineId":
		return j.PipelineId
	case "Description":
		return j.Description
	default:
		return nil
	}
}

func (j *Job) GetSortByFieldPrefix(name string) string {
	return "jobs."
}

func (j *Job) GetKeyFieldPrefix() string {
	return "jobs."
}
