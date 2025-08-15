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
	"encoding/json"
)

type Task struct {
	UUID      string `gorm:"column:UUID; not null; primaryKey; type:varchar(191);"`
	Namespace string `gorm:"column:Namespace; not null;"`
	// PipelineName was deprecated. Use RunId instead.
	PipelineName string `gorm:"column:PipelineName; not null;"`
	// RunId is limited to varchar(191) to make it indexable as a foreign key.
	// For details on type lengths and index safety, refer to comments in the Pipeline struct.
	// nolint:staticcheck // [ST1003] Field name matches upstream legacy naming
	RunId              string           `gorm:"column:RunUUID; type:varchar(191); not null; index:tasks_RunUUID_run_details_UUID_foreign;"`                            // Note: field name (RunId) ≠ column name (RunUUID). The former should be the foreign key instead of the letter.
	Run                Run              `gorm:"foreignKey:RunId;references:UUID;constraint:tasks_RunUUID_run_details_UUID_foreign,OnDelete:CASCADE,OnUpdate:CASCADE;"` // A Task belongs to a Run.
	PodName            string           `gorm:"column:PodName; not null;"`
	MLMDExecutionID    string           `gorm:"column:MLMDExecutionID; not null;"`
	CreatedTimestamp   int64            `gorm:"column:CreatedTimestamp; not null;"`
	StartedTimestamp   int64            `gorm:"column:StartedTimestamp; default:0;"`
	FinishedTimestamp  int64            `gorm:"column:FinishedTimestamp; default:0;"`
	Fingerprint        string           `gorm:"column:Fingerprint; not null;"`
	Name               string           `gorm:"column:Name; default:null"`
	ParentTaskId       string           `gorm:"column:ParentTaskUUID; default:null"`
	State              RuntimeState     `gorm:"column:State; default:null;"`
	StateHistoryString string           `gorm:"column:StateHistory; default:null; type:longtext;"`
	MLMDInputs         string           `gorm:"column:MLMDInputs; default:null; type:longtext;"`
	MLMDOutputs        string           `gorm:"column:MLMDOutputs; default:null; type:longtext;"`
	ChildrenPodsString string           `gorm:"column:ChildrenPods; default:null; type:longtext;"`
	StateHistory       []*RuntimeStatus `gorm:"-;"`
	ChildrenPods       []string         `gorm:"-;"`
	Payload            string           `gorm:"column:Payload; default:null; type:longtext;"`
}

func (t Task) ToString() string {
	task, err := json.Marshal(t)
	if err != nil {
		return ""
	} else {
		return string(task)
	}
}

func (t Task) PrimaryKeyColumnName() string {
	return "UUID"
}

func (t Task) DefaultSortField() string {
	return "CreatedTimestamp"
}

func (t Task) APIToModelFieldMap() map[string]string {
	return taskAPIToModelFieldMap
}

func (t Task) GetModelName() string {
	return "tasks"
}

func (t Task) GetSortByFieldPrefix(s string) string {
	return "tasks."
}

func (t Task) GetKeyFieldPrefix() string {
	return "tasks."
}

var taskAPIToModelFieldMap = map[string]string{
	"task_id":         "UUID", // v2beta1 API
	"id":              "UUID", // v1beta1 API
	"namespace":       "Namespace",
	"pipeline_name":   "PipelineName",      // v2beta1 API
	"pipelineName":    "PipelineName",      // v1beta1 API
	"run_id":          "RunUUID",           // v2beta1 API
	"runId":           "RunUUID",           // v1beta1 API
	"display_name":    "Name",              // v2beta1 API
	"execution_id":    "MLMDExecutionID",   // v2beta1 API
	"create_time":     "CreatedTimestamp",  // v2beta1 API
	"start_time":      "StartedTimestamp",  // v2beta1 API
	"end_time":        "FinishedTimestamp", // v2beta1 API
	"fingerprint":     "Fingerprint",
	"state":           "State",             // v2beta1 API
	"state_history":   "StateHistory",      // v2beta1 API
	"parent_task_id":  "ParentTaskUUID",    // v2beta1 API
	"mlmdExecutionID": "MLMDExecutionID",   // v1beta1 API
	"created_at":      "CreatedTimestamp",  // v1beta1 API
	"finished_at":     "FinishedTimestamp", // v1beta1 API
}

func (t Task) GetField(name string) (string, bool) {
	if field, ok := taskAPIToModelFieldMap[name]; ok {
		return field, true
	}
	return "", false
}

func (t Task) GetFieldValue(name string) interface{} {
	switch name {
	case "UUID":
		return t.UUID
	case "Namespace":
		return t.Namespace
	case "PipelineName":
		return t.PipelineName
	case "RunId":
		return t.RunId
	case "MLMDExecutionID":
		return t.MLMDExecutionID
	case "CreatedTimestamp":
		return t.CreatedTimestamp
	case "FinishedTimestamp":
		return t.FinishedTimestamp
	case "Fingerprint":
		return t.Fingerprint
	case "ParentTaskId":
		return t.ParentTaskId
	case "State":
		return t.State
	case "Name":
		return t.Name
	case "MLMDInputs":
		return t.MLMDInputs
	case "MLMDOutputs":
		return t.MLMDOutputs
	default:
		return nil
	}
}
