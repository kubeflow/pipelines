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

type Run struct {
	UUID               string `gorm:"column:UUID; not null; primary_key"`
	DisplayName        string `gorm:"column:DisplayName; not null;"` /* The name that user provides. Can contain special characters*/
	Name               string `gorm:"column:Name; not null;"`        /* The name of the K8s resource. Follow regex '[a-z0-9]([-a-z0-9]*[a-z0-9])?'*/
	StorageState       string `gorm:"column:StorageState; not null;"`
	Namespace          string `gorm:"column:Namespace; not null;"`
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
