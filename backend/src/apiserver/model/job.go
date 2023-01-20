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
	StatusStateUnspecified   StatusState = "STATUS_UNSPECIFIED"
	StatusStateUnspecifiedV1 StatusState = "UNKNOWN_MODE"
	StatusStateEnabled       StatusState = "ENABLED"
	StatusStateEnabledV1     StatusState = "ENABLED"
	StatusStateDisabled      StatusState = "DISABLED"
	StatusStateDisabledV1    StatusState = "DISABLED"
)

func (s StatusState) ToV1() StatusState {
	return s.ToV2()
}

func (s StatusState) ToV2() StatusState {
	return StatusState(s.ToString())
}

func (s StatusState) ToUpper() StatusState {
	return StatusState(strings.ToUpper(string(s)))
}

func (s StatusState) ToString() string {
	switch s.ToUpper() {
	case StatusStateUnspecified, StatusStateUnspecifiedV1, "MODE_UNSPECIFIED", "NO_STATUS", "":
		return string(StatusStateUnspecified)
	case StatusStateEnabled, "ENABLE", "READY", "RUNNING", "SUCCEEDED":
		return string(StatusStateEnabled)
	case StatusStateDisabled, "DISABLE":
		return string(StatusStateDisabled)
	default:
		return ""
	}
}

func (s StatusState) IsValid() bool {
	switch s {
	case StatusStateUnspecified, StatusStateEnabled, StatusStateDisabled:
		return true
	default:
		return false
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
	UpdatedAtInSec int64  `gorm:"column:UpdatedAtInSec; not null;"`
	Enabled        bool   `gorm:"column:Enabled; not null;"`
	ExperimentId   string `gorm:"column:ExperimentUUID; not null;"`
	// ResourceReferences are deprecated. Use Namespace, ExperimentId
	// PipelineSpec.PipelineId, PipelineSpec.PipelineVersionId
	ResourceReferences []*ResourceReference
	Trigger
	PipelineSpec
	Conditions string `gorm:"column:Conditions; not null;"`
}

func (j *Job) ToV1() *Job {
	if j.ResourceReferences == nil {
		j.ResourceReferences = make([]*ResourceReference, 0)
	}
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
	j.Conditions = string(StatusState(j.Conditions).ToV1())
	return j
}

func (j *Job) ToV2() *Job {
	if j.ResourceReferences != nil {
		for _, ref := range j.ResourceReferences {
			switch ref.ReferenceType {
			case ExperimentResourceType:
				j.ExperimentId = ref.ReferenceUUID
			case NamespaceResourceType:
				j.Namespace = ref.ReferenceUUID
			}
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
	"id":          "UUID",
	"name":        "DisplayName",
	"created_at":  "CreatedAtInSec",
	"updated_at":  "UpdatedAtInSec",
	"description": "Description",
}

// APIToModelFieldMap returns a map from API names to field names for model Job.
func (k *Job) APIToModelFieldMap() map[string]string {
	return jobAPIToModelFieldMap
}

// GetModelName returns table name used as sort field prefix
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
	case "PipelineId":
		return j.PipelineId
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
