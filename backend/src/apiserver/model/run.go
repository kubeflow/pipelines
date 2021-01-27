// Copyright 2018 Google LLC
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

const (
	RunTerminatingConditions string = "Terminating"
)

type Run struct {
	UUID               string `gorm:"column:UUID; not null; primary_key"`
	ExperimentUUID     string `gorm:"column:ExperimentUUID; not null;"`
	DisplayName        string `gorm:"column:DisplayName; not null;"` /* The name that user provides. Can contain special characters*/
	Name               string `gorm:"column:Name; not null;"`        /* The name of the K8s resource. Follow regex '[a-z0-9]([-a-z0-9]*[a-z0-9])?'*/
	StorageState       string `gorm:"column:StorageState; not null;"`
	Namespace          string `gorm:"column:Namespace; not null;"`
	ServiceAccount     string `gorm:"column:ServiceAccount; not null;"`
	Description        string `gorm:"column:Description; not null;"`
	CreatedAtInSec     int64  `gorm:"column:CreatedAtInSec; not null;"`
	ScheduledAtInSec   int64  `gorm:"column:ScheduledAtInSec; default:0;"`
	FinishedAtInSec    int64  `gorm:"column:FinishedAtInSec; default:0;"`
	Conditions         string `gorm:"column:Conditions; not null"`
	Metrics            []*RunMetric
	ResourceReferences []*ResourceReference
	PipelineSpec
}

type PipelineRuntime struct {
	PipelineRuntimeManifest string `gorm:"column:PipelineRuntimeManifest; not null; size:65535"`
	/* Argo CRD. Set size to 65535 so it will be stored as longtext. https://dev.mysql.com/doc/refman/8.0/en/column-count-limit.html */
	WorkflowRuntimeManifest string `gorm:"column:WorkflowRuntimeManifest; not null; size:65535"`
}

type RunDetail struct {
	Run
	PipelineRuntime
}

type RunMetric struct {
	RunUUID     string  `gorm:"column:RunUUID; not null;primary_key"`
	NodeID      string  `gorm:"column:NodeID; not null; primary_key"`
	Name        string  `gorm:"column:Name; not null;primary_key"`
	NumberValue float64 `gorm:"column:NumberValue"`
	Format      string  `gorm:"column:Format"`
	Payload     string  `gorm:"column:Payload; not null; size:65535"`
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
	"id":            "UUID",
	"name":          "DisplayName",
	"created_at":    "CreatedAtInSec",
	"description":   "Description",
	"scheduled_at":  "ScheduledAtInSec",
	"storage_state": "StorageState",
	"status":        "Conditions",
}

// APIToModelFieldMap returns a map from API names to field names for model Run.
func (r *Run) APIToModelFieldMap() map[string]string {
	return runAPIToModelFieldMap
}

// GetModelName returns table name used as sort field prefix
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
		return r.CreatedAtInSec
	case "Description":
		return r.Description
	case "ScheduledAtInSec":
		return r.ScheduledAtInSec
	case "StorageState":
		return r.StorageState
	case "Conditions":
		return r.Conditions
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
